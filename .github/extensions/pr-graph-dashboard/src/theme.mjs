// Graph palette shared by the canvas stylesheet and the files written to disk.
//
// On screen the graph is coloured entirely by `ui/styles.css`, which resolves
// the app theme tokens the host mirrors onto the canvas document. An exported
// file carries no stylesheet, so Graphviz would fall back to black on white.
// To keep a single source of truth, the palette is read back out of the same
// `:root` block and baked into the DOT source as literal colour attributes.

import { readFile } from "node:fs/promises";

const STYLESHEET = new URL("../ui/styles.css", import.meta.url);

/** Node fill opacity; mirrors the `color-mix` percentage in `ui/styles.css`. */
export const NODE_FILL_ALPHA = 0.14;

/** Background painted behind exported SVGs so dark labels stay readable. */
export const EXPORT_BACKGROUND = "#ffffff";

const NEUTRAL = "#8b949e";

const DEFAULT_PALETTE = { node: {}, relation: {}, text: "#1f2328", mutedText: "#59636e" };

let pending = null;

/** Picks the literal colour out of a declaration such as `var(--x, #4c8eda)`. */
function literalColor(value) {
    const matches = String(value).match(/#[0-9a-fA-F]{3,8}\b/g);
    return matches ? matches.at(-1).toLowerCase() : null;
}

/** Expands `#rgb` to `#rrggbb` so an alpha channel can be appended. */
function expand(color) {
    if (!/^#[0-9a-f]{3}$/i.test(color)) return color;
    const [, r, g, b] = color;
    return `#${r}${r}${g}${g}${b}${b}`;
}

/** Appends an alpha channel; Graphviz understands `#rrggbbaa`. */
export function withAlpha(color, alpha) {
    const base = expand(color);
    if (!/^#[0-9a-f]{6}$/i.test(base)) return base;
    const byte = Math.max(0, Math.min(255, Math.round(alpha * 255)));
    return `${base}${byte.toString(16).padStart(2, "0")}`;
}

/** Reads the custom properties declared in the stylesheet's `:root` block. */
function parseRoot(css) {
    const block = css.match(/:root\s*\{([^}]*)\}/);
    const vars = new Map();
    if (!block) return vars;
    for (const declaration of block[1].split(";")) {
        const match = declaration.match(/(--[A-Za-z0-9-]+)\s*:\s*([^;]+)/);
        if (match) vars.set(match[1], match[2].trim());
    }
    return vars;
}

/** Builds the palette from the dashboard stylesheet. */
async function load() {
    try {
        const vars = parseRoot(await readFile(STYLESHEET, "utf-8"));
        const palette = { node: {}, relation: {}, text: DEFAULT_PALETTE.text, mutedText: DEFAULT_PALETTE.mutedText };
        for (const [name, value] of vars) {
            const color = literalColor(value);
            if (!color) continue;
            if (name.startsWith("--nt-")) palette.node[name.slice(5)] = color;
            else if (name.startsWith("--rel-")) palette.relation[name.slice(6)] = color;
            else if (name === "--graph-text") palette.text = color;
            else if (name === "--graph-muted-text") palette.mutedText = color;
        }
        return palette;
    } catch {
        // A missing or unreadable stylesheet must not break exporting.
        return DEFAULT_PALETTE;
    }
}

/** Resolves the graph palette, reading the stylesheet at most once. */
export function graphPalette() {
    if (!pending) pending = load();
    return pending;
}

/** Colour of a node type, falling back to the neutral tone. */
export function nodeColor(palette, token) {
    return palette.node[token] ?? palette.node.other ?? NEUTRAL;
}

/** Colour of an edge relation, falling back to the neutral tone. */
export function edgeColor(palette, token) {
    return palette.relation[token] ?? palette.relation.unknown ?? NEUTRAL;
}
