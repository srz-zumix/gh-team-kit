// Thin wrapper around the local Graphviz layout engines.

import { execFile, spawn } from "node:child_process";

/** Layout engines offered by the dashboard, in menu order. */
export const ENGINES = ["dot", "neato", "fdp", "sfdp", "circo", "twopi"];

let availability;

/**
 * Layout processes currently running. A Graphviz child outlives the extension
 * process that spawned it, so a reload would otherwise leave a core pinned by a
 * layout nobody is waiting for any more.
 */
const running = new Set();

/** Kills every layout in flight. Called when the extension shuts down. */
export function killLayouts() {
    for (const child of running) child.kill("SIGKILL");
    running.clear();
}

/** Resolves whether Graphviz is installed, caching the probe result. */
export async function checkGraphviz() {
    if (availability) return availability;
    availability = new Promise((resolve) => {
        execFile("dot", ["-V"], { timeout: 10_000 }, (error, _stdout, stderr) => {
            if (error) {
                resolve({ available: false, version: "", error: error.message });
                return;
            }
            resolve({ available: true, version: (stderr || "").trim(), error: "" });
        });
    });
    return availability;
}

/**
 * Graphviz escapes hyphens in `class` attributes as `&#45;`. Decoding them up
 * front keeps CSS selectors such as `.nt-user` working regardless of how the
 * markup is parsed or written to disk.
 */
function normalizeClasses(svg) {
    return svg.replace(/class="([^"]*)"/g, (_match, value) => `class="${value.replace(/&#45;/g, "-")}"`);
}

/**
 * Renders DOT source to SVG using the requested layout engine.
 *
 * @param {string} source DOT source.
 * @param {{engine?: string, timeoutMs?: number, signal?: AbortSignal}} [options]
 * @returns {Promise<string>} SVG markup.
 */
export function renderSvg(source, options = {}) {
    const engine = ENGINES.includes(options.engine) ? options.engine : "dot";
    const timeoutMs = options.timeoutMs ?? 60_000;
    const signal = options.signal;
    return new Promise((resolve, reject) => {
        if (signal?.aborted) {
            reject(new Error("layout cancelled"));
            return;
        }
        const child = spawn(engine, ["-Tsvg"], { stdio: ["pipe", "pipe", "pipe"] });
        running.add(child);
        let stdout = "";
        let stderr = "";
        let settled = false;
        // A superseded layout is worthless, and on a large graph it keeps a core
        // busy for minutes, so cancellation kills the process rather than
        // waiting for it.
        const onAbort = () => {
            if (settled) return;
            settled = true;
            clearTimeout(timer);
            child.kill("SIGKILL");
            running.delete(child);
            reject(new Error("layout cancelled"));
        };
        const finish = (fn) => (value) => {
            running.delete(child);
            signal?.removeEventListener("abort", onAbort);
            fn(value);
        };
        const done = finish(resolve);
        const fail = finish(reject);
        const timer = setTimeout(() => {
            if (settled) return;
            settled = true;
            child.kill("SIGKILL");
            fail(
                new Error(
                    `${engine} timed out after ${Math.round(timeoutMs / 1000)}s; try a faster layout engine ` +
                        `(sfdp or neato), stronger filters, or a longer render limit`,
                ),
            );
        }, timeoutMs);
        // Registered only after `timer` exists so the abort handler never
        // observes it in the temporal dead zone.
        signal?.addEventListener("abort", onAbort, { once: true });

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
            fail(
                error.code === "ENOENT"
                    ? new Error(`Graphviz layout engine "${engine}" was not found; install Graphviz (e.g. brew install graphviz)`)
                    : error,
            );
        });
        child.on("close", (code) => {
            if (settled) return;
            settled = true;
            clearTimeout(timer);
            if (code !== 0) {
                fail(new Error(stderr.trim() || `${engine} exited with code ${code}`));
                return;
            }
            done(normalizeClasses(stdout));
        });

        child.stdin.on("error", () => {
            // Ignore EPIPE: the close handler reports the real failure.
        });
        child.stdin.end(source, "utf-8");
    });
}
