// Filesystem browsing for the Open and Export dialogs.
//
// The canvas iframe cannot fall back to a native picker: `<input type="file">`
// never exposes an absolute path inside the host webview, and it cannot pick a
// save destination at all. The dialogs therefore browse the filesystem through
// this module, which the loopback server exposes read-only.

import { readdir, stat } from "node:fs/promises";
import { homedir } from "node:os";
import path from "node:path";
import { generatedDir } from "./store.mjs";

/** Pseudo-directory collecting every DOT file worth opening. */
export const FOUND_DIR = "@found";

const MAX_ENTRIES = 1000;
const MAX_FOUND = 200;

const SKIP_DIRS = new Set(["node_modules", ".git", "vendor", "dist", "build", ".venv", "target"]);

/** Expands a leading `~` against the current user's home directory. */
export function expandHome(target) {
    const value = String(target ?? "").trim();
    if (value === "~") return homedir();
    if (value.startsWith("~/")) return path.join(homedir(), value.slice(2));
    return value;
}

/** Resolves a possibly relative or `~`-prefixed path against a base directory. */
export function resolveUnder(target, base) {
    const expanded = expandHome(target);
    if (!expanded) return base;
    return path.isAbsolute(expanded) ? path.normalize(expanded) : path.resolve(base, expanded);
}

async function isDirectory(target) {
    try {
        return (await stat(target)).isDirectory();
    } catch {
        return false;
    }
}

/**
 * Finds candidate DOT files: everything under the workspace (bounded depth)
 * plus previously generated artifacts.
 */
export async function listDotCandidates(workspacePath) {
    const results = [];
    const walk = async (dir, depth, origin) => {
        let entries;
        try {
            entries = await readdir(dir, { withFileTypes: true });
        } catch {
            return;
        }
        for (const entry of entries) {
            if (results.length >= MAX_FOUND) return;
            const full = path.join(dir, entry.name);
            if (entry.isDirectory()) {
                if (depth <= 0 || entry.name.startsWith(".") || SKIP_DIRS.has(entry.name)) continue;
                await walk(full, depth - 1, origin);
                continue;
            }
            if (!entry.isFile()) continue;
            if (!/\.(dot|gv)$/i.test(entry.name)) continue;
            results.push({ path: full, name: entry.name, origin });
        }
    };
    if (workspacePath) await walk(workspacePath, 4, "workspace");
    await walk(generatedDir(), 1, "generated");
    return results.sort((a, b) => a.origin.localeCompare(b.origin) || a.path.localeCompare(b.path));
}

/** Resolves one directory entry, following symlinks to learn what they point at. */
async function classify(dir, entry, extensions) {
    const full = path.join(dir, entry.name);
    let kind = entry.isDirectory() ? "dir" : entry.isFile() ? "file" : null;
    if (entry.isSymbolicLink()) {
        try {
            const info = await stat(full);
            kind = info.isDirectory() ? "dir" : info.isFile() ? "file" : null;
        } catch {
            return null;
        }
    }
    if (!kind) return null;
    const extension = path.extname(entry.name).replace(/^\./, "").toLowerCase();
    return {
        name: entry.name,
        path: full,
        kind,
        hidden: entry.name.startsWith("."),
        // Directories always match so the tree stays navigable under a filter.
        match: kind === "dir" || extensions.length === 0 || extensions.includes(extension),
    };
}

/** Lists a single directory, directories first and then files by name. */
export async function listDirectory(dir, { extensions = [] } = {}) {
    const raw = await readdir(dir, { withFileTypes: true });
    const entries = [];
    for (const entry of raw) {
        const resolved = await classify(dir, entry, extensions);
        if (resolved) entries.push(resolved);
    }
    entries.sort((a, b) => {
        if (a.kind !== b.kind) return a.kind === "dir" ? -1 : 1;
        return a.name.localeCompare(b.name, undefined, { sensitivity: "base" });
    });
    return entries;
}

/** Builds the shortcut list shown above the file list. */
async function listPlaces({ workspacePath, sourcePath }) {
    const candidates = [
        { id: "found", label: "Found", path: FOUND_DIR, virtual: true },
        { id: "workspace", label: "Workspace", path: workspacePath },
        { id: "generated", label: "Generated", path: generatedDir() },
        { id: "current", label: "Current file", path: sourcePath ? path.dirname(sourcePath) : null },
        { id: "home", label: "Home", path: homedir() },
    ];
    const places = [];
    for (const place of candidates) {
        if (!place.path) continue;
        if (!place.virtual && !(await isDirectory(place.path))) continue;
        if (places.some((existing) => existing.path === place.path)) continue;
        places.push(place);
    }
    return places;
}

/**
 * Lists one directory for the file browser. A file path is accepted too: its
 * parent is listed and the file comes back as `select` so the UI can highlight
 * it.
 *
 * @param {{dir?: string, workspacePath: string, sourcePath?: string|null, extensions?: string[]}} options
 */
export async function browse({ dir, workspacePath, sourcePath = null, extensions = [] } = {}) {
    const base = workspacePath || process.cwd();
    const places = await listPlaces({ workspacePath: base, sourcePath });

    if (String(dir ?? "").trim() === FOUND_DIR) {
        const found = await listDotCandidates(base);
        return {
            dir: FOUND_DIR,
            label: "Found DOT files",
            virtual: true,
            parent: null,
            sep: path.sep,
            entries: found.map((file) => ({
                name: file.name,
                path: file.path,
                kind: "file",
                detail: `${file.origin} · ${path.dirname(file.path)}`,
                hidden: false,
                match: true,
            })),
            select: null,
            truncated: found.length >= MAX_FOUND,
            places,
        };
    }

    const target = resolveUnder(dir, base);
    let info;
    try {
        info = await stat(target);
    } catch {
        throw new Error(`no such file or directory: ${target}`);
    }
    if (!info.isDirectory() && !info.isFile()) throw new Error(`not a directory: ${target}`);

    const current = info.isFile() ? path.dirname(target) : target;
    const parent = path.dirname(current);
    const entries = await listDirectory(current, { extensions });
    return {
        dir: current,
        label: current,
        virtual: false,
        parent: parent === current ? null : parent,
        sep: path.sep,
        entries: entries.slice(0, MAX_ENTRIES),
        select: info.isFile() ? target : null,
        truncated: entries.length > MAX_ENTRIES,
        places,
    };
}
