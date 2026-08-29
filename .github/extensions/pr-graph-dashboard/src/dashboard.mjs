// Per-instance dashboard state: DOT source, filters, layout and rendered SVG.

import { EventEmitter } from "node:events";
import { readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { resolveUnder } from "./browse.mjs";
import { describeNode, emitDot, filterGraph, parseDot, summarize, topNodes } from "./dot.mjs";
import { ENGINES, renderSvg } from "./graphviz.mjs";
import { prGraphHelp, prGraphShellCommand, runPrGraph, shellQuote } from "./prgraph.mjs";
import {
    clearGenerateHistory,
    loadGenerateHistory,
    recordGenerateHistory,
    reserveGeneratedDotPath,
    saveGeneratedDot,
} from "./store.mjs";
import { EXPORT_BACKGROUND, graphPalette } from "./theme.mjs";

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

/**
 * Render limits offered by the dashboard, in menu order (milliseconds).
 * The layout runs in a child process, so a long limit stalls nothing but the
 * graph itself; a six figure node count genuinely needs the upper end.
 */
export const RENDER_LIMITS = [30_000, 60_000, 120_000, 300_000, 600_000, 1_800_000, 3_600_000, 10_800_000];

/**
 * Size above which a layout is not started automatically. Graphviz needs
 * minutes and hundreds of megabytes on a graph this size, and the resulting
 * SVG is heavy enough to make the whole app sluggish, so it takes an explicit
 * request instead of happening on every filter change.
 */
export const RENDER_BUDGET = { nodes: 2_000, edges: 6_000 };

/**
 * Prepares a rendered SVG for life outside the canvas. The graph is laid out on
 * a transparent background so the app theme shows through; a file needs an
 * opaque one of its own, or its dark labels vanish in a dark viewer.
 */
function standaloneSvg(svg) {
    const open = svg.match(/<svg\b[^>]*>/);
    if (!open) return svg;
    const at = open.index + open[0].length;
    return `${svg.slice(0, at)}\n<rect width="100%" height="100%" fill="${EXPORT_BACKGROUND}"/>${svg.slice(at)}`;
}

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
        this.generateArgsRev = 0;
        this.busy = "";
        this.error = "";
        this.svg = "";
        this.renderRev = 0;
        this.renderError = "";
        this.stateRev = 0;

        this.renderSignature = "";
        this.renderPending = false;
        this.rendering = false;
        this.renderSkipped = null;
        this.renderOverride = false;
        this.renderAbort = null;
    }

    /** Resolves a possibly relative path against the workspace. */
    resolvePath(target) {
        const value = String(target ?? "").trim();
        if (!value) throw new Error("a file path is required");
        return resolveUnder(value, this.workspacePath);
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

    /**
     * Asks the agent to run the same command in the app's Terminal canvas.
     *
     * The extension cannot open host canvases itself, so this hands the work to
     * the agent: it runs the command where the user can watch it, then loads the
     * DOT file this method reserved back into the dashboard.
     */
    async runInTerminal(args) {
        this.generateArgs = String(args ?? "").trim();
        this.touch();
        await recordGenerateHistory(this.generateArgs);
        const outFile = await reserveGeneratedDotPath(this.generateArgs || "pr-graph");
        const command = prGraphShellCommand({ args: this.generateArgs, outFile });
        const body = [
            "Run this pr-graph command in a Terminal canvas so I can watch it.",
            "",
            "```sh",
            `cd ${shellQuote(this.workspacePath)}`,
            command,
            "```",
            "",
            "Steps:",
            '1. open_canvas with canvasId "terminal" and a fresh instanceId.',
            '2. send_terminal_input with the two lines above (the `cd` first).',
            "3. The command is slow and prints nothing until it finishes, so poll" +
                " read_terminal_output every 30 seconds or so until the shell prompt returns.",
            `4. When it succeeds, call invoke_canvas_action on ${this.instanceId} with actionName` +
                ` "load_dot" and path "${outFile}" so this panel shows the result.`,
            "5. If it fails, tell me what the terminal reported instead of retrying blindly.",
        ].join("\n");
        await this.ask(body);
        return { path: outFile, command };
    }

    /**
     * Arguments previously run, newest first. Read from disk on every call so
     * that two panels open at once do not drift apart.
     */
    async generateHistory() {
        return await loadGenerateHistory();
    }

    /** Empties the argument history. */
    async resetGenerateHistory() {
        return await clearGenerateHistory();
    }

    /**
     * Fills the Generate dialog without running anything. The revision lets the
     * panel notice a proposal that arrived while the dialog was closed.
     */
    setGenerateArgs(args) {
        this.generateArgs = String(args ?? "").trim();
        this.generateArgsRev += 1;
        this.touch();
        return this.generateArgs;
    }

    /** Asks the agent to turn a plain-language request into pr-graph arguments. */
    async askForArgs(request) {
        const text = String(request ?? "").trim();
        if (!text) throw new Error("describe what you want the graph to show");
        let help = "";
        try {
            help = await prGraphHelp({ cwd: this.workspacePath });
        } catch {
            // Without the help text the agent can still read it itself.
        }
        const body = [
            "Propose arguments for the Generate dialog of this dashboard.",
            `What I want: ${text}`,
            this.generateArgs ? `Arguments currently in the dialog: ${this.generateArgs}` : "",
            'Answer by calling invoke_canvas_action with actionName "set_generate_args" and the argument' +
                " string, then say in one short sentence why you chose those flags. Do not call the" +
                ' "generate" action unless I ask for the graph to be built.',
            "Omit --format: the dashboard always appends `--format dot`.",
            help ? "Reference — `gh team-kit pr-graph --help`:" : "",
            help ? "```" : "",
            help,
            help ? "```" : "",
        ]
            .filter(Boolean)
            .join("\n");
        return this.ask(body);
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
        if (this.rendering) {
            // Whatever is laying out now is already out of date; stop it rather
            // than let it finish and burn a core in the meantime.
            this.renderAbort?.abort();
            return;
        }
        void this.renderNow();
    }

    /** Renders once past the size budget, at the user's explicit request. */
    renderAnyway() {
        const nodes = this.filteredGraph().nodes.length;
        // A second click must not abort the layout the first one started.
        if (this.rendering) return { nodes, alreadyRendering: true };
        this.renderOverride = true;
        this.renderSignature = "";
        // Drop the notice straight away so the click has a visible effect.
        this.renderSkipped = null;
        this.scheduleRender();
        return { nodes };
    }

    /**
     * Throws the drawing away. A big layout leaves tens of thousands of live
     * elements in the panel, which bogs every other control down; this is the
     * way back out without reloading or re-filtering.
     */    clearRender() {
        this.renderAbort?.abort();
        const filtered = this.filteredGraph();
        this.svg = "";
        this.renderError = "";
        this.renderOverride = false;
        this.renderSkipped = { nodes: filtered.nodes.length, edges: filtered.edges.length, reason: "cleared" };
        // Force the next render to run: the filters have not changed, so the
        // signature alone would short-circuit it.
        this.renderSignature = "";
        this.renderRev += 1;
        this.touch();
        return { nodes: filtered.nodes.length, edges: filtered.edges.length };
    }

    /** Renders the filtered graph to SVG unless the result is already current. */
    async renderNow() {
        this.rendering = true;
        // Announce the start before the filtering and DOT generation below, which
        // are synchronous and take seconds on a large graph.
        this.touch();
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
                    this.renderSkipped = null;
                    this.renderSignature = signature;
                    this.renderRev += 1;
                    this.touch();
                    continue;
                }
                const overBudget =
                    filtered.nodes.length > RENDER_BUDGET.nodes || filtered.edges.length > RENDER_BUDGET.edges;
                if (overBudget && !this.renderOverride) {
                    this.svg = "";
                    this.renderError = "";
                    this.renderSkipped = { nodes: filtered.nodes.length, edges: filtered.edges.length, reason: "budget" };
                    this.renderSignature = signature;
                    this.renderRev += 1;
                    this.touch();
                    continue;
                }
                this.renderOverride = false;
                this.renderSkipped = null;
                this.renderAbort = new AbortController();
                try {
                    const svg = await renderSvg(
                        emitDot(filtered, {
                            rankdir: this.view.rankdir,
                            engine: this.view.engine,
                            palette: await graphPalette(),
                        }),
                        {
                            engine: this.view.engine,
                            timeoutMs: this.view.timeoutMs,
                            signal: this.renderAbort.signal,
                        },
                    );
                    this.svg = svg;
                    this.renderError = "";
                } catch (error) {
                    // A cancelled layout is not a failure: the loop is about to
                    // start the one that replaced it.
                    if (this.renderAbort.signal.aborted) continue;
                    this.svg = "";
                    this.renderError = error instanceof Error ? error.message : String(error);
                }
                this.renderSignature = signature;
                this.renderRev += 1;
                this.touch();
            }
        } finally {
            this.renderAbort = null;
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
        await writeFile(file, standaloneSvg(this.svg), "utf-8");
        return file;
    }

    /** Writes the current (filtered) DOT source to disk. */
    async exportDot(target, { filtered = true } = {}) {
        if (!this.dotSource) throw new Error("there is no graph to export");
        const file = this.resolvePath(target);
        const source = filtered
            ? emitDot(this.filteredGraph(), {
                  rankdir: this.view.rankdir,
                  engine: this.view.engine,
                  palette: await graphPalette(),
              })
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
            generateArgsRev: this.generateArgsRev,
            busy: this.busy,
            error: this.error,
            renderError: this.renderError,
            renderSkipped: this.renderSkipped,
            renderBudget: RENDER_BUDGET,
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
            "Use invoke_canvas_action on this instanceId (get_graph, describe_node, set_filters, select_node, load_dot, generate, set_generate_args) to inspect or drive the dashboard.",
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
