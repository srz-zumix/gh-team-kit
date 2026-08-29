// Runs `gh team-kit pr-graph` and captures its DOT output.

import { spawn } from "node:child_process";

/**
 * Splits a command-line argument string into tokens, honouring single and
 * double quotes and backslash escapes. Arguments are passed to `gh` without a
 * shell, so no shell metacharacter is interpreted.
 */
export function splitArgs(input) {
    const args = [];
    let current = "";
    let quote = "";
    let started = false;
    for (let i = 0; i < input.length; i += 1) {
        const ch = input[i];
        if (quote) {
            if (ch === "\\" && quote === '"' && (input[i + 1] === '"' || input[i + 1] === "\\")) {
                current += input[i + 1];
                i += 1;
                continue;
            }
            if (ch === quote) {
                quote = "";
                continue;
            }
            current += ch;
            continue;
        }
        if (ch === '"' || ch === "'") {
            quote = ch;
            started = true;
            continue;
        }
        if (ch === "\\" && input[i + 1]) {
            current += input[i + 1];
            started = true;
            i += 1;
            continue;
        }
        if (/\s/.test(ch)) {
            if (started) {
                args.push(current);
                current = "";
                started = false;
            }
            continue;
        }
        current += ch;
        started = true;
    }
    if (started) args.push(current);
    return args;
}

/** Removes any user supplied `--format` flag so the dashboard always gets DOT. */
function stripFormat(args) {
    const result = [];
    for (let i = 0; i < args.length; i += 1) {
        const arg = args[i];
        if (arg === "--format" || arg === "-f") {
            i += 1;
            continue;
        }
        if (arg.startsWith("--format=")) continue;
        result.push(arg);
    }
    return result;
}

/**
 * Builds the full argument vector for `gh`, always ending in `--format dot`.
 *
 * @param {string|string[]} [args] User supplied arguments
 * @returns {string[]}
 */
export function prGraphArgv(args) {
    const raw = Array.isArray(args) ? args : splitArgs(args ?? "");
    return ["team-kit", "pr-graph", ...stripFormat(raw), "--format", "dot"];
}

/** Quotes a token for a POSIX shell, leaving safe tokens untouched. */
export function shellQuote(token) {
    return /^[A-Za-z0-9_@%+=:,./-]+$/.test(token) ? token : `'${String(token).replace(/'/g, `'\\''`)}'`;
}

/**
 * Renders the command a human would type to produce the same DOT file, for the
 * "Run in terminal" path where the agent, not this process, runs `gh`.
 *
 * @param {{args?: string|string[], outFile?: string}} [options]
 * @returns {string}
 */
export function prGraphShellCommand(options = {}) {
    const command = ["gh", ...prGraphArgv(options.args)].map(shellQuote).join(" ");
    return options.outFile ? `${command} > ${shellQuote(options.outFile)}` : command;
}

/**
 * Runs `gh team-kit pr-graph <args> --format dot`.
 *
 * @param {{args?: string|string[], cwd?: string, timeoutMs?: number}} options
 * @returns {Promise<{dot: string, command: string, stderr: string}>}
 */
export function runPrGraph(options = {}) {
    const args = prGraphArgv(options.args);
    const timeoutMs = options.timeoutMs ?? 600_000;
    return new Promise((resolve, reject) => {
        const child = spawn("gh", args, {
            cwd: options.cwd,
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
            reject(new Error(`gh team-kit pr-graph timed out after ${Math.round(timeoutMs / 1000)}s`));
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
                    ? new Error("the `gh` CLI was not found on PATH")
                    : error,
            );
        });
        child.on("close", (code) => {
            if (settled) return;
            settled = true;
            clearTimeout(timer);
            const command = `gh ${args.join(" ")}`;
            if (code !== 0) {
                reject(new Error(`${command} failed with code ${code}: ${stderr.trim() || "no stderr output"}`));
                return;
            }
            if (!stdout.includes("digraph")) {
                reject(new Error(`${command} produced no DOT output: ${stderr.trim() || "empty stdout"}`));
                return;
            }
            resolve({ dot: stdout, command, stderr: stderr.trim() });
        });
    });
}

let helpText;

/**
 * Returns `gh team-kit pr-graph --help`, cached for the life of the process.
 * It is embedded in the prompt that asks the agent for arguments so the agent
 * does not have to look the flags up itself.
 *
 * @param {{cwd?: string, timeoutMs?: number}} [options]
 * @returns {Promise<string>}
 */
export function prGraphHelp(options = {}) {
    if (helpText) return helpText;
    helpText = new Promise((resolve, reject) => {
        const child = spawn("gh", ["team-kit", "pr-graph", "--help"], {
            cwd: options.cwd,
            stdio: ["ignore", "pipe", "pipe"],
            env: { ...process.env, GH_PROMPT_DISABLED: "1", NO_COLOR: "1", CLICOLOR: "0" },
        });
        let stdout = "";
        let stderr = "";
        const timer = setTimeout(() => {
            child.kill("SIGKILL");
            reject(new Error("gh team-kit pr-graph --help timed out"));
        }, options.timeoutMs ?? 30_000);
        child.stdout.setEncoding("utf-8");
        child.stdout.on("data", (chunk) => (stdout += chunk));
        child.stderr.setEncoding("utf-8");
        child.stderr.on("data", (chunk) => (stderr += chunk));
        child.on("error", (error) => {
            clearTimeout(timer);
            reject(error.code === "ENOENT" ? new Error("the `gh` CLI was not found on PATH") : error);
        });
        child.on("close", (code) => {
            clearTimeout(timer);
            // Cobra prints help to stdout, but tolerate builds that use stderr.
            const text = (stdout || stderr).trim();
            if (code !== 0 && !text) {
                reject(new Error(`gh team-kit pr-graph --help failed with code ${code}`));
                return;
            }
            resolve(text);
        });
    });
    helpText.catch(() => {
        helpText = undefined;
    });
    return helpText;
}
