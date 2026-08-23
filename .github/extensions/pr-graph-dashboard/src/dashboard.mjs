// Per-instance dashboard state: DOT source, filters, layout and rendered SVG.

import { EventEmitter } from "node:events";
import { readFile, readdir, writeFile } from "node:fs/promises";
import path from "node:path";
import { describeNode, emitDot, filterGraph, parseDot, summarize, topNodes } from "./dot.mjs";
import { ENGINES, renderSvg } from "./graphviz.mjs";
import { runPrGraph } from "./prgraph.mjs";
import { generatedDir, saveGeneratedDot } from "./store.mjs";

const DEFAULT_FILTERS = {
    nodeTypes: [],
    relations: [],
    minWeight: 0,
    search: "",
    focus: "",
    hops: 1,
    keepOrphans: false,
};

const DEFAULT_VIEW = { engine: "dot", rankdir: "LR", timeoutMs: 60_000 };

const RANKDIRS = ["LR", "TB", "RL", "BT"];

const SKIP_DIRS = new Set(["node_modules", ".git", "vendor", "dist", "build", ".venv", "target"]);

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
            if (results.length >= 200) return;
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

/** Normalizes an incoming filter patch against the defaults. */
function normalizeFilters(current, patch = {}) {
    const next = { ...current };
    if (Array.isArray(patch.nodeTypes)) next.nodeTypes = [...new Set(patch.nodeTypes.map(String))];
    if (Array.isArray(patch.relations)) next.relations = [...new Set(patch.relations.map(String))];
    if (patch.minWeight !== undefined) next.minWeight = Math.max(0, Number(patch.minWeight) || 0);
    if (patch.search !== undefined) next.search = String(patch.search ?? "");
    if (patch.focus !== undefined) next.focus = String(patch.focus ?? "");
    if (patch.hops !== undefined) next.hops = Math.min(6, Math.max(0, Number(patch.hops) || 0));
    if (patch.keepOrphans !== undefined) next.keepOrphans = Boolean(patch.keepOrphans);
    return next;
}

/** Render limits offered by the dashboard, in menu order (milliseconds). */
export const RENDER_LIMITS = [30_000, 60_000, 120_000, 300_000, 600_000];

/** Normalizes an incoming layout patch against the defaults. */
function normalizeView(current, patch = {}) {
    const next = { ...current };
    if (patch.engine !== undefined && ENGINES.includes(patch.engine)) next.engine = patch.engine;
    if (patch.rankdir !== undefined && RANKDIRS.includes(String(patch.rankdir).toUpperCase())) {
        next.rankdir = String(patch.rankdir).toUpperCase();
    }
    if (patch.timeoutMs !== undefined) {
        const ms = Math.round(Number(patch.timeoutMs));
        if (Number.isFinite(ms)) {
            next.timeoutMs = Math.min(RENDER_LIMITS.at(-1), Math.max(RENDER_LIMITS[0], ms));
        }
    }
    return next;
}

/** Holds everything one open canvas panel renders and exposes. */
export class Dashboard extends EventEmitter {
    constructor({ instanceId, canvasId, workspacePath, sendToAgent, log }) {
        super();
        this.instanceId = instanceId;
        this.canvasId = canvasId;
        this.workspacePath = workspacePath ?? process.cwd();
        this.sendToAgent = sendToAgent ?? (async () => {});
        this.log = log ?? (() => {});

        this.sourcePath = null;
        this.sourceLabel = "";
        this.sourceCommand = "";
        this.dotSource = "";
        this.graph = { nodes: [], edges: [], warnings: [] };
        this.filters = { ...DEFAULT_FILTERS };
        this.view = { ...DEFAULT_VIEW };
        this.selection = null;
        this.selectRev = 0;
        this.generateArgs = "";
        this.busy = "";
        this.error = "";
        this.svg = "";
        this.renderRev = 0;
        this.renderError = "";
        this.stateRev = 0;

        this.renderSignature = "";
        this.renderPending = false;
        this.rendering = false;
    }

    /** Resolves a possibly relative path against the workspace. */
    resolvePath(target) {
        const value = String(target ?? "").trim();
        if (!value) throw new Error("a file path is required");
        const expanded = value.startsWith("~/") ? path.join(process.env.HOME ?? "", value.slice(2)) : value;
        return path.isAbsolute(expanded) ? path.normalize(expanded) : path.resolve(this.workspacePath, expanded);
    }

    /** Emits a state-changed event so connected SSE clients refresh. */
    touch() {
        this.stateRev += 1;
        this.emit("changed");
    }

    /** Replaces the current DOT source and re-parses it. */
    setDot(dotSource, { sourcePath = null, sourceLabel = "", sourceCommand = "" } = {}) {
        this.dotSource = dotSource;
        this.graph = parseDot(dotSource);
        this.sourcePath = sourcePath;
        this.sourceLabel = sourceLabel || (sourcePath ? path.basename(sourcePath) : "inline DOT");
        this.sourceCommand = sourceCommand;
        this.error = "";
        if (this.filters.focus && !this.graph.nodes.some((node) => node.id === this.filters.focus)) {
            this.filters.focus = "";
        }
        if (this.selection && !this.graph.nodes.some((node) => node.id === this.selection)) {
            this.selection = null;
        }
        this.scheduleRender();
        this.touch();
    }

    /** Loads DOT from a file on disk. */
    async loadFile(target) {
        const file = this.resolvePath(target);
        const contents = await readFile(file, "utf-8");
        this.setDot(contents, { sourcePath: file });
        return file;
    }

    /** Runs `gh team-kit pr-graph`, stores the output as an artifact and loads it. */
    async generate(args) {
        this.generateArgs = String(args ?? "");
        this.busy = "Running gh team-kit pr-graph…";
        this.error = "";
        this.touch();
        try {
            const { dot, command } = await runPrGraph({ args: this.generateArgs, cwd: this.workspacePath });
            const file = await saveGeneratedDot(dot, this.generateArgs || "pr-graph");
            this.setDot(dot, { sourcePath: file, sourceCommand: command });
            return { path: file, command };
        } finally {
            this.busy = "";
            this.touch();
        }
    }

    /** Updates filters and re-renders. */
    setFilters(patch) {
        this.filters = normalizeFilters(this.filters, patch);
        this.scheduleRender();
        this.touch();
        return this.filters;
    }

    /** Resets filters to their defaults. */
    resetFilters() {
        this.filters = { ...DEFAULT_FILTERS };
        this.scheduleRender();
        this.touch();
        return this.filters;
    }

    /** Updates the layout engine / direction and re-renders. */
    setView(patch) {
        this.view = normalizeView(this.view, patch);
        this.scheduleRender();
        this.touch();
        return this.view;
    }

    /** Selects a node (or clears the selection when given a falsy id). */
    select(nodeId) {
        const id = nodeId ? String(nodeId) : null;
        if (id && !this.graph.nodes.some((node) => node.id === id)) {
            throw new Error(`node "${id}" is not present in the current graph`);
        }
        this.selection = id;
        // Bumped on every call so the panel can move the view to the node again
        // even when the same node is picked twice.
        this.selectRev += 1;
        this.touch();
        return id;
    }

    /** The graph after the current filters are applied. */
    filteredGraph() {
        return filterGraph(this.graph, this.filters);
    }

    /** Queues a re-render, coalescing bursts of state changes. */
    scheduleRender() {
        this.renderPending = true;
        if (this.rendering) return;
        void this.renderNow();
    }

    /** Renders the filtered graph to SVG unless the result is already current. */
    async renderNow() {
        this.rendering = true;
        try {
            while (this.renderPending) {
                this.renderPending = false;
                const filtered = this.filteredGraph();
                const signature = JSON.stringify([
                    this.sourcePath,
                    this.dotSource.length,
                    this.filters,
                    // The render limit is deliberately left out: changing it must
                    // not invalidate a layout that is already on screen.
                    this.view.engine,
                    this.view.rankdir,
                    filtered.nodes.length,
                    filtered.edges.length,
                ]);
                if (signature === this.renderSignature && this.svg) continue;
                if (!this.dotSource) {
                    this.svg = "";
                    this.renderError = "";
                    this.renderSignature = signature;
                    this.renderRev += 1;
                    this.touch();
                    continue;
                }
                try {
                    const svg = await renderSvg(
                        emitDot(filtered, { rankdir: this.view.rankdir, engine: this.view.engine }),
                        {
                            engine: this.view.engine,
                            timeoutMs: this.view.timeoutMs,
                        },
                    );
                    this.svg = svg;
                    this.renderError = "";
                } catch (error) {
                    this.svg = "";
                    this.renderError = error instanceof Error ? error.message : String(error);
                }
                this.renderSignature = signature;
                this.renderRev += 1;
                this.touch();
            }
        } finally {
            this.rendering = false;
            // Announce the end of the batch so the panel knows the SVG on screen
            // now matches the current filters.
            this.touch();
        }
    }

    /** Writes the current SVG to disk. */
    async exportSvg(target) {
        if (!this.svg) throw new Error("there is no rendered graph to export");
        const file = this.resolvePath(target);
        await writeFile(file, this.svg, "utf-8");
        return file;
    }

    /** Writes the current (filtered) DOT source to disk. */
    async exportDot(target, { filtered = true } = {}) {
        if (!this.dotSource) throw new Error("there is no graph to export");
        const file = this.resolvePath(target);
        const source = filtered
            ? emitDot(this.filteredGraph(), { rankdir: this.view.rankdir, engine: this.view.engine })
            : this.dotSource;
        await writeFile(file, `${source}\n`, "utf-8");
        return file;
    }

    /** Compact statistics for the full and filtered graphs. */
    stats() {
        const full = summarize(this.graph);
        const filtered = summarize(this.filteredGraph());
        return {
            total: { nodes: full.nodeCount, edges: full.edgeCount },
            visible: { nodes: filtered.nodeCount, edges: filtered.edgeCount },
            nodeTypeCounts: full.nodeTypeCounts,
            relationCounts: full.relationCounts,
        };
    }

    /** Full JSON snapshot consumed by the UI and by `get_state`. */
    snapshot() {
        const stats = this.stats();
        const selection = this.selection ? describeNode(this.graph, this.selection) : null;
        return {
            instanceId: this.instanceId,
            canvasId: this.canvasId,
            workspacePath: this.workspacePath,
            source: {
                path: this.sourcePath,
                label: this.sourceLabel,
                command: this.sourceCommand,
                loaded: Boolean(this.dotSource),
            },
            filters: this.filters,
            view: this.view,
            engines: ENGINES,
            rankdirs: RANKDIRS,
            renderLimits: RENDER_LIMITS,
            selection,
            selectRev: this.selectRev,
            stats,
            warnings: this.graph.warnings ?? [],
            generateArgs: this.generateArgs,
            busy: this.busy,
            error: this.error,
            renderError: this.renderError,
            renderRev: this.renderRev,
            rendering: this.rendering || this.renderPending,
            stateRev: this.stateRev,
            hasSvg: Boolean(this.svg),
        };
    }

    /** Graph payload for the agent: stats, top nodes and optionally every edge. */
    graphPayload({ limit = 15, includeNodes = false, includeEdges = false, useFilters = true } = {}) {
        const graph = useFilters ? this.filteredGraph() : this.graph;
        const stats = summarize(graph);
        const payload = {
            source: { path: this.sourcePath, label: this.sourceLabel, command: this.sourceCommand },
            filters: useFilters ? this.filters : null,
            nodeCount: stats.nodeCount,
            edgeCount: stats.edgeCount,
            nodeTypeCounts: stats.nodeTypeCounts,
            relationCounts: stats.relationCounts,
            topNodes: topNodes(graph, { limit }),
        };
        if (includeNodes) payload.nodes = graph.nodes;
        if (includeEdges) {
            payload.edges = graph.edges.map((edge) => ({
                from: edge.from,
                to: edge.to,
                relation: edge.relation,
                weight: edge.weight,
            }));
        }
        return payload;
    }

    /** Describes one node and its neighbours. */
    describe(nodeId, { useFilters = false } = {}) {
        const graph = useFilters ? this.filteredGraph() : this.graph;
        return describeNode(graph, nodeId);
    }

    /** Sends a prompt to the agent, prefixed with the canvas context. */
    async ask(prompt) {
        const text = String(prompt ?? "").trim();
        if (!text) throw new Error("a prompt is required");
        // Stamp every prompt: the agent otherwise cannot tell a fresh send from
        // a re-delivery of an earlier one, because the body is identical when
        // the same preset is used twice.
        this.askSeq = (this.askSeq ?? 0) + 1;
        const stats = this.stats().visible;
        const context = [
            "[pr-graph dashboard]",
            `send #${this.askSeq} at ${new Date().toISOString()}`,
            `canvasId: ${this.canvasId}, instanceId: ${this.instanceId}`,
            this.sourcePath ? `DOT file: ${this.sourcePath}` : "DOT file: (none loaded)",
            this.sourceCommand ? `generated by: ${this.sourceCommand}` : "",
            `visible graph: ${stats.nodes} nodes / ${stats.edges} edges`,
            this.selection ? `selected node: ${this.selection}` : "",
            "Use invoke_canvas_action on this instanceId (get_graph, describe_node, set_filters, select_node, load_dot, generate) to inspect or drive the dashboard.",
            "",
            text,
        ]
            .filter(Boolean)
            .join("\n");
        await this.sendToAgent(context);
        return { prompt: text, seq: this.askSeq };
    }
}

/** Ready-made prompts offered as buttons in the dashboard. */
export const PROMPT_PRESETS = [
    {
        id: "summary",
        label: "Summarize graph",
        needsSelection: false,
        prompt: "Summarize this PR activity graph: who the main reviewers and authors are, which code areas attract the most activity, and anything notable about the relationships.",
    },
    {
        id: "selection",
        label: "Analyze selection",
        needsSelection: true,
        prompt: "Analyze the selected node: what it connects to, how strong those relationships are, and what that says about the review workflow.",
    },
    {
        id: "review-load",
        label: "Review load balance",
        needsSelection: false,
        prompt: "Look at the review, approval and review-request relationships and tell me whether review load is unevenly distributed. Name the reviewers who are over- or under-loaded and suggest concrete rebalancing steps.",
    },
    {
        id: "ownership",
        label: "Ownership risk",
        needsSelection: false,
        prompt: "Look at the changed / in / owned-by relationships and identify code areas with a low bus factor (files or directories touched or owned by very few people). Suggest where to spread knowledge.",
    },
    {
        id: "next-query",
        label: "Suggest next query",
        needsSelection: false,
        prompt: "Based on what this graph shows, suggest the next `gh team-kit pr-graph` invocation worth running (flags and why), then run it on this canvas with the `generate` action.",
    },
];
