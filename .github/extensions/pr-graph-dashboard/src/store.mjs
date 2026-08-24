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
    await mkdir(artifactsDir(), { recursive: true });
    const file = instancesFile();
    const all = await readJson(file, {});
    all[instanceId] = pointer;
    await writeFile(file, `${JSON.stringify(all, null, 2)}\n`, "utf-8");
}

/** Drops the persisted pointer for a closed instance. */
export async function removeInstancePointer(instanceId) {
    const file = instancesFile();
    const all = await readJson(file, null);
    if (!all || !(instanceId in all)) return;
    delete all[instanceId];
    await writeFile(file, `${JSON.stringify(all, null, 2)}\n`, "utf-8");
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
 * Writes generated DOT output to the artifacts directory and returns its path.
 * The path becomes the durable identity of the generated graph.
 */
export async function saveGeneratedDot(dot, label) {
    const dir = generatedDir();
    await mkdir(dir, { recursive: true });
    const stamp = new Date().toISOString().replace(/[:.]/g, "-");
    const file = path.join(dir, `${slug(label)}-${stamp}.dot`);
    await writeFile(file, dot, "utf-8");
    return file;
}
