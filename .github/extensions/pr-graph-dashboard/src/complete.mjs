// Argument completion for `gh team-kit pr-graph`, backed by cobra's hidden
// `__complete` command.

import { spawn } from "node:child_process";
import { splitArgs } from "./prgraph.mjs";

const CACHE_LIMIT = 200;
const cache = new Map();

/** Splits typed text into the finished words and the word under the caret. */
export function splitForCompletion(line) {
    const text = String(line ?? "");
    const words = splitArgs(text);
    // A trailing space (or empty input) means the caret starts a new word.
    const partial = text === "" || /\s$/.test(text) ? "" : (words.pop() ?? "");
    return { words, partial };
}

/**
 * Parses `__complete` output. Candidates come one per line as
 * `value<TAB>description`, terminated by a `:<directive>` line.
 */
export function parseCompletions(stdout) {
    const candidates = [];
    for (const line of String(stdout).split("\n")) {
        if (line.startsWith(":")) break;
        if (!line.trim()) continue;
        const [value, description = ""] = line.split("\t");
        candidates.push({ value, description: description.trim() });
    }
    return candidates;
}

function runComplete(words, partial, { cwd, timeoutMs = 15_000 } = {}) {
    return new Promise((resolve, reject) => {
        const child = spawn("gh", ["team-kit", "__complete", "pr-graph", ...words, partial], {
            cwd,
            stdio: ["ignore", "pipe", "pipe"],
            env: { ...process.env, GH_PROMPT_DISABLED: "1", NO_COLOR: "1", CLICOLOR: "0" },
        });
        let stdout = "";
        let stderr = "";
        let settled = false;
        const timer = setTimeout(() => {
            if (settled) return;
            settled = true;
            child.kill("SIGKILL");
            reject(new Error(`completion timed out after ${Math.round(timeoutMs / 1000)}s`));
        }, timeoutMs);
        child.stdout.setEncoding("utf-8");
        child.stdout.on("data", (chunk) => {
            stdout += chunk;
        });
        child.stderr.setEncoding("utf-8");
        child.stderr.on("data", (chunk) => {
            stderr += chunk;
        });
        child.on("error", (error) => {
            if (settled) return;
            settled = true;
            clearTimeout(timer);
            reject(
                error.code === "ENOENT"
                    ? new Error("the GitHub CLI was not found; install gh to complete arguments")
                    : error,
            );
        });
        child.on("close", (code) => {
            if (settled) return;
            settled = true;
            clearTimeout(timer);
            if (code !== 0) {
                reject(new Error(stderr.trim().split("\n")[0] || `gh team-kit __complete exited with code ${code}`));
                return;
            }
            resolve(parseCompletions(stdout));
        });
    });
}

/**
 * Completes the argument being typed at the end of `line`.
 *
 * @param {string} line Arguments typed so far, up to the caret.
 * @param {{cwd?: string, timeoutMs?: number}} [options]
 * @returns {Promise<{partial: string, candidates: {value: string, description: string}[]}>}
 */
export async function completeArgs(line, options = {}) {
    const { words, partial } = splitForCompletion(line);
    // Flag names are asked for once per position: cobra returns every flag for
    // `--`, so longer prefixes are filtered here instead of spawning `gh` again.
    const probe = partial.startsWith("--") ? "--" : partial;
    const key = JSON.stringify([options.cwd ?? "", words, probe]);
    let pending = cache.get(key);
    if (!pending) {
        pending = runComplete(words, probe, options);
        pending.catch(() => cache.delete(key));
        if (cache.size >= CACHE_LIMIT) cache.delete(cache.keys().next().value);
        cache.set(key, pending);
    }
    const candidates = await pending;
    return { partial, candidates: candidates.filter((candidate) => candidate.value.startsWith(partial)) };
}

/** Forgets cached completions, used by the tests. */
export function clearCompletionCache() {
    cache.clear();
}
