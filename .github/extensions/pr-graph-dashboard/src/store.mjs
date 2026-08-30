// Durable storage for the pr-graph dashboard.
//
// Panels (`instanceId`) are transient, so nothing the user expects to keep is
// keyed by them alone: the durable identity of a dashboard is the absolute
// path of the DOT file it renders. Generated graphs are written to the
// extension's artifacts directory so they survive reloads and can be reopened
// later. Only the lightweight "which file was this panel showing" pointer is
// stored per instance, so that a provider restart can rehydrate the view.

import { mkdir, readFile, writeFile } from "node:fs/promises";
import { homedir } from "node:os";
import path from "node:path";

const EXTENSION_NAME = "pr-graph-dashboard";

/** Absolute path of the extension's artifacts directory. */
export function artifactsDir() {
    const copilotHome = process.env.COPILOT_HOME || path.join(homedir(), ".copilot");
    return path.join(copilotHome, "extensions", EXTENSION_NAME, "artifacts");
}

/** Absolute path of the directory holding generated DOT files. */
export function generatedDir() {
    return path.join(artifactsDir(), "generated");
}

function instancesFile() {
    return path.join(artifactsDir(), "instances.json");
}

// Serializes every read-modify-write against the shared JSON files
// (instances.json and generate-history.json). Panels run concurrently and all
// share this module, so without a queue two overlapping saves could each read,
// mutate and write back, silently dropping the other's change.
let mutationQueue = Promise.resolve();
function enqueueMutation(task) {
    const run = mutationQueue.then(task, task);
    mutationQueue = run.then(
        () => {},
        () => {},
    );
    return run;
}

async function readJson(file, fallback) {
    try {
        return JSON.parse(await readFile(file, "utf-8"));
    } catch {
        return fallback;
    }
}

/** Returns the persisted pointer for an instance, or `null`. */
export async function loadInstancePointer(instanceId) {
    const all = await readJson(instancesFile(), {});
    return all[instanceId] ?? null;
}

/** Persists the pointer describing what an instance is currently showing. */
export async function saveInstancePointer(instanceId, pointer) {
    await enqueueMutation(async () => {
        await mkdir(artifactsDir(), { recursive: true });
        const file = instancesFile();
        const all = await readJson(file, {});
        all[instanceId] = pointer;
        await writeFile(file, `${JSON.stringify(all, null, 2)}\n`, "utf-8");
    });
}

/** Drops the persisted pointer for a closed instance. */
export async function removeInstancePointer(instanceId) {
    await enqueueMutation(async () => {
        const file = instancesFile();
        const all = await readJson(file, null);
        if (!all || !(instanceId in all)) return;
        delete all[instanceId];
        await writeFile(file, `${JSON.stringify(all, null, 2)}\n`, "utf-8");
    });
}

/** Builds a filesystem-safe slug from arbitrary text. */
function slug(text) {
    const cleaned = String(text ?? "")
        .replace(/[^A-Za-z0-9._-]+/g, "-")
        .replace(/^-+|-+$/g, "")
        .slice(0, 60);
    return cleaned || "graph";
}

/**
 * Returns the path a generated graph would be written to, without creating it.
 * Used by the "Run in terminal" flow, which has to name the destination before
 * any output exists.
 */
export async function reserveGeneratedDotPath(label) {
    const dir = generatedDir();
    await mkdir(dir, { recursive: true });
    const stamp = new Date().toISOString().replace(/[:.]/g, "-");
    return path.join(dir, `${slug(label)}-${stamp}.dot`);
}

/**
 * Writes generated DOT output to the artifacts directory and returns its path.
 * The path becomes the durable identity of the generated graph.
 */
export async function saveGeneratedDot(dot, label) {
    const file = await reserveGeneratedDotPath(label);
    await writeFile(file, dot, "utf-8");
    return file;
}

const HISTORY_LIMIT = 25;

function historyFile() {
    return path.join(artifactsDir(), "generate-history.json");
}

/**
 * Arguments previously handed to `gh team-kit pr-graph`, newest first.
 *
 * The list is shared by every panel and every session, so it is read from disk
 * on each call rather than cached: two dashboards open side by side would
 * otherwise show different histories.
 */
export async function loadGenerateHistory() {
    const items = await readJson(historyFile(), []);
    if (!Array.isArray(items)) return [];
    return items.filter((item) => item && typeof item.args === "string").slice(0, HISTORY_LIMIT);
}

/** Records one set of arguments, moving a repeat to the front instead of duplicating it. */
export async function recordGenerateHistory(args) {
    const value = String(args ?? "").trim();
    if (!value) return await loadGenerateHistory();
    return await enqueueMutation(async () => {
        const previous = await loadGenerateHistory();
        const items = [{ args: value, at: new Date().toISOString() }, ...previous.filter((item) => item.args !== value)];
        const kept = items.slice(0, HISTORY_LIMIT);
        await mkdir(artifactsDir(), { recursive: true });
        await writeFile(historyFile(), `${JSON.stringify(kept, null, 2)}\n`, "utf-8");
        return kept;
    });
}

/** Empties the history. */
export async function clearGenerateHistory() {
    await enqueueMutation(async () => {
        await mkdir(artifactsDir(), { recursive: true });
        await writeFile(historyFile(), "[]\n", "utf-8");
    });
    return [];
}
