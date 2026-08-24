// Thin wrapper around the local Graphviz layout engines.

import { execFile, spawn } from "node:child_process";

/** Layout engines offered by the dashboard, in menu order. */
export const ENGINES = ["dot", "neato", "fdp", "sfdp", "circo", "twopi"];

let availability;

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
 * @param {{engine?: string, timeoutMs?: number}} [options]
 * @returns {Promise<string>} SVG markup.
 */
export function renderSvg(source, options = {}) {
    const engine = ENGINES.includes(options.engine) ? options.engine : "dot";
    const timeoutMs = options.timeoutMs ?? 60_000;
    return new Promise((resolve, reject) => {
        const child = spawn(engine, ["-Tsvg"], { stdio: ["pipe", "pipe", "pipe"] });
        let stdout = "";
        let stderr = "";
        let settled = false;
        const timer = setTimeout(() => {
            if (settled) return;
            settled = true;
            child.kill("SIGKILL");
            reject(
                new Error(
                    `${engine} timed out after ${Math.round(timeoutMs / 1000)}s; try a faster layout engine ` +
                        `(sfdp or neato), stronger filters, or a longer render limit`,
                ),
            );
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
                    ? new Error(`Graphviz layout engine "${engine}" was not found; install Graphviz (e.g. brew install graphviz)`)
                    : error,
            );
        });
        child.on("close", (code) => {
            if (settled) return;
            settled = true;
            clearTimeout(timer);
            if (code !== 0) {
                reject(new Error(stderr.trim() || `${engine} exited with code ${code}`));
                return;
            }
            resolve(normalizeClasses(stdout));
        });

        child.stdin.on("error", () => {
            // Ignore EPIPE: the close handler reports the real failure.
        });
        child.stdin.end(source, "utf-8");
    });
}
