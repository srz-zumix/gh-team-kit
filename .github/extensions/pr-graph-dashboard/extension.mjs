// Extension: pr-graph-dashboard
//
// Renders `gh team-kit pr-graph --format dot` output as an interactive graph in
// a canvas, and lets the dashboard hand work back to the agent.
//
// This entry point only wires things together: DOT handling lives in
// src/dot.mjs, layout in src/graphviz.mjs, CLI invocation in src/prgraph.mjs,
// per-panel state in src/dashboard.mjs, the loopback server in src/server.mjs,
// and durable storage in src/store.mjs.

import { CanvasError, createCanvas, joinSession } from "@github/copilot-sdk/extension";
import { Dashboard, RENDER_LIMITS } from "./src/dashboard.mjs";
import { NODE_TYPES, RELATIONS } from "./src/dot.mjs";
import { ENGINES, killLayouts } from "./src/graphviz.mjs";
import { startServer } from "./src/server.mjs";
import { loadInstancePointer, removeInstancePointer, saveInstancePointer } from "./src/store.mjs";

const CANVAS_ID = "pr-graph";

/** Live panels, keyed by the host-supplied instance id. */
const instances = new Map();

let session;

function log(message, level = "info") {
    session?.log(`[pr-graph-dashboard] ${message}`, { level });
}

/** Returns the dashboard for an instance, or throws a structured canvas error. */
function requireInstance(instanceId) {
    const entry = instances.get(instanceId);
    if (!entry) {
        throw new CanvasError("canvas_instance_not_found", `no open pr-graph canvas with instanceId "${instanceId}"`);
    }
    return entry.dashboard;
}

/** Wraps an action handler so failures surface as structured canvas errors. */
function action(name, description, inputSchema, handler) {
    return {
        name,
        description,
        inputSchema,
        handler: async (ctx) => {
            const dashboard = requireInstance(ctx.instanceId);
            try {
                return await handler(dashboard, ctx.input ?? {}, ctx);
            } catch (error) {
                if (error instanceof CanvasError) throw error;
                throw new CanvasError(name, error instanceof Error ? error.message : String(error));
            }
        },
    };
}

/** Stores the pointer that lets a restarted provider restore the same graph. */
async function persist(dashboard) {
    if (dashboard.closed || !dashboard.sourcePath) return;
    await saveInstancePointer(dashboard.instanceId, {
        path: dashboard.sourcePath,
        command: dashboard.sourceCommand,
        filters: dashboard.filters,
        view: dashboard.view,
        selection: dashboard.selection,
        updatedAt: new Date().toISOString(),
    });
}

/**
 * Persists on every state change, coalescing bursts. Without this a provider
 * restart drops the filters, layout and selection the user set up, because
 * only explicit load/generate calls wrote the pointer.
 *
 * Returns a dispose function that detaches the listener and cancels any pending
 * write, so a closing panel cannot resurrect its instances.json pointer after
 * onClose has removed it.
 */
function autoPersist(dashboard, log) {
    let timer = null;
    const onChanged = () => {
        clearTimeout(timer);
        timer = setTimeout(() => {
            void persist(dashboard).catch((error) => {
                log(`failed to persist dashboard state: ${error instanceof Error ? error.message : String(error)}`, "warning");
            });
        }, 400);
    };
    dashboard.on("changed", onChanged);
    return () => {
        clearTimeout(timer);
        dashboard.off("changed", onChanged);
    };
}

/** Restores the filters, layout and selection saved for the same graph. */
function restoreViewState(dashboard, pointer) {
    if (!pointer || pointer.path !== dashboard.sourcePath) return;
    // Best effort: a saved value can no longer apply if the file changed, and
    // that must not stop the graph from opening.
    for (const [value, apply] of [
        [pointer.filters, (v) => dashboard.setFilters(v)],
        [pointer.view, (v) => dashboard.setView(v)],
        [pointer.selection, (v) => dashboard.select(v)],
    ]) {
        if (!value) continue;
        try {
            apply(value);
        } catch {
            /* the saved value no longer fits this graph */
        }
    }
}

/** Applies an open input (or a persisted pointer) to a dashboard. */
async function applySource(dashboard, input, { workspaceOnly = false } = {}) {
    if (input.dot) {
        dashboard.setDot(String(input.dot), { sourceLabel: input.label ? String(input.label) : "inline DOT" });
        return;
    }
    if (input.path) {
        await dashboard.loadFile(input.path, { workspaceOnly });
        return;
    }
    if (input.args !== undefined) {
        await dashboard.generate(input.args);
    }
}

/** Applies a source, recording any failure on the dashboard instead of throwing. */
async function applyOrRecord(dashboard, requested, options) {
    try {
        await applySource(dashboard, requested, options);
        return true;
    } catch (error) {
        dashboard.error = error instanceof Error ? error.message : String(error);
        dashboard.touch();
        log(`failed to load graph: ${dashboard.error}`, "warning");
        return false;
    }
}

const canvas = createCanvas({
    id: CANVAS_ID,
    displayName: "PR graph",
    description:
        "Interactive viewer for `gh team-kit pr-graph --format dot` output: filter by node type, relation and edge weight, focus a node's neighbourhood, and generate new graphs.",
    inputSchema: {
        type: "object",
        additionalProperties: false,
        properties: {
            path: { type: "string", description: "Path to a DOT file; must resolve inside the workspace (use the dashboard's Open… browser for files elsewhere)" },
            dot: { type: "string", description: "Inline DOT source to render instead of a file" },
            label: { type: "string", description: "Display label used when `dot` is supplied" },
            args: {
                type: "string",
                description: "Arguments for `gh team-kit pr-graph`; the graph is generated on open when set",
            },
        },
    },
    actions: [
        action(
            "load_dot",
            "Load a DOT file (or inline DOT source) into the dashboard.",
            {
                type: "object",
                additionalProperties: false,
                properties: {
                    path: {
                        type: "string",
                        description: "Path to a DOT file; must resolve inside the workspace (use the dashboard's Open… browser for files elsewhere)",
                    },
                    dot: { type: "string", description: "Inline DOT source to render instead of a file" },
                    label: { type: "string", description: "Display label used when `dot` is supplied" },
                },
            },
            async (dashboard, input) => {
                if (!input.path && !input.dot) throw new Error("either `path` or `dot` is required");
                await applySource(dashboard, input, { workspaceOnly: true });
                await persist(dashboard);
                return { source: dashboard.snapshot().source, stats: dashboard.stats() };
            },
        ),
        action(
            "generate",
            "Run `gh team-kit pr-graph <args> --format dot` and display the result.",
            {
                type: "object",
                additionalProperties: false,
                properties: {
                    args: {
                        type: "string",
                        description: 'Arguments for `gh team-kit pr-graph`, e.g. "--limit 100 --state merged --no-bots"',
                    },
                },
            },
            async (dashboard, input) => {
                const result = await dashboard.generate(input.args ?? "");
                await persist(dashboard);
                return { ...result, stats: dashboard.stats() };
            },
        ),
        action(
            "set_generate_args",
            "Fill the dashboard's Generate dialog with `gh team-kit pr-graph` arguments for the user to review, without running anything. Use this when the user asks for arguments rather than for the graph itself.",
            {
                type: "object",
                additionalProperties: false,
                required: ["args"],
                properties: {
                    args: {
                        type: "string",
                        description:
                            'Arguments for `gh team-kit pr-graph`, e.g. "--since 2026-01-01 --state merged --no-bots". Omit --format.',
                    },
                },
            },
            (dashboard, input) => ({ args: dashboard.setGenerateArgs(input.args) }),
        ),
        action(
            "get_state",
            "Return what the dashboard is currently showing: source, filters, layout, selection and counts.",
            { type: "object", additionalProperties: false, properties: {} },
            (dashboard) => dashboard.snapshot(),
        ),
        action(
            "get_graph",
            "Return the graph data: counts per node type and relation, the most connected nodes, and optionally every node and edge.",
            {
                type: "object",
                additionalProperties: false,
                properties: {
                    limit: {
                        type: "integer",
                        minimum: 1,
                        maximum: 200,
                        description: "Number of top nodes to return (default 15)",
                    },
                    includeNodes: { type: "boolean", description: "Include the full node list (default false)" },
                    includeEdges: { type: "boolean", description: "Include the full edge list (default false)" },
                    useFilters: { type: "boolean", description: "Apply the dashboard filters (default true)" },
                },
            },
            (dashboard, input) => dashboard.graphPayload(input),
        ),
        action(
            "describe_node",
            "Return one node with its incoming and outgoing edges, resolved to node names.",
            {
                type: "object",
                additionalProperties: false,
                required: ["node"],
                properties: {
                    node: { type: "string", description: 'Node id, e.g. "user:octocat" or "directory:cmd"' },
                    useFilters: { type: "boolean", description: "Apply the dashboard filters (default false)" },
                },
            },
            (dashboard, input) => {
                const described = dashboard.describe(input.node, input);
                if (!described) throw new Error(`node "${input.node}" is not present in the graph`);
                return described;
            },
        ),
        action(
            "set_filters",
            "Change which nodes and edges the dashboard shows.",
            {
                type: "object",
                additionalProperties: false,
                properties: {
                    nodeTypes: {
                        type: "array",
                        items: { type: "string", enum: NODE_TYPES },
                        description: "Keep only these node types; an empty array keeps all",
                    },
                    relations: {
                        type: "array",
                        items: { type: "string", enum: RELATIONS },
                        description: "Keep only these edge relations; an empty array keeps all",
                    },
                    minWeight: { type: "integer", minimum: 0, description: "Drop edges below this weight" },
                    search: {
                        type: "string",
                        description: "Keep nodes whose name matches this text, plus their neighbourhood",
                    },
                    focus: { type: "string", description: "Node id to focus the graph on; empty string clears the focus" },
                    hops: { type: "integer", minimum: 0, maximum: 6, description: "Neighbourhood radius for search/focus" },
                    keepOrphans: { type: "boolean", description: "Keep nodes left without any edge" },
                    reset: { type: "boolean", description: "Reset every filter to its default" },
                },
            },
            (dashboard, input) => {
                const filters = input.reset ? dashboard.resetFilters() : dashboard.setFilters(input);
                return { filters, stats: dashboard.stats() };
            },
        ),
        action(
            "set_view",
            "Change the Graphviz layout engine, graph direction, or how long one layout may take.",
            {
                type: "object",
                additionalProperties: false,
                properties: {
                    engine: { type: "string", enum: ENGINES, description: "Graphviz layout engine" },
                    rankdir: { type: "string", enum: ["LR", "TB", "RL", "BT"], description: "Graph direction" },
                    timeoutMs: {
                        type: "number",
                        enum: RENDER_LIMITS,
                        description: "How long Graphviz may spend on one layout, in milliseconds",
                    },
                },
            },
            (dashboard, input) => ({ view: dashboard.setView(input) }),
        ),
        action(
            "select_node",
            "Select a node in the dashboard and move the view to it, or clear the selection.",
            {
                type: "object",
                additionalProperties: false,
                properties: { node: { type: "string", description: "Node id; omit or pass an empty string to clear" } },
            },
            (dashboard, input) => ({ selection: dashboard.select(input.node || null) }),
        ),
        action(
            "export",
            "Write the current graph to disk as SVG or DOT.",
            {
                type: "object",
                additionalProperties: false,
                required: ["path"],
                properties: {
                    path: { type: "string", description: "Destination path; must resolve inside the workspace (use the dashboard's Export… browser for other locations)" },
                    kind: { type: "string", enum: ["svg", "dot"], description: "Output format (default svg)" },
                    filtered: { type: "boolean", description: "For DOT, export the filtered graph (default true)" },
                },
            },
            async (dashboard, input) => {
                const file =
                    input.kind === "dot"
                        ? await dashboard.exportDot(input.path, { filtered: input.filtered !== false, workspaceOnly: true })
                        : await dashboard.exportSvg(input.path, { workspaceOnly: true });
                return { path: file };
            },
        ),
    ],

    open: async (ctx) => {
        const input = ctx.input ?? {};
        let entry = instances.get(ctx.instanceId);

        if (!entry) {
            const dashboard = new Dashboard({
                instanceId: ctx.instanceId,
                canvasId: ctx.canvasId,
                // `session.workspacePath` points at the session's artifact
                // directory, not the repository. The extension process is
                // forked with the project session's checkout as its working
                // directory, which is what `gh team-kit pr-graph` and
                // workspace-relative paths need.
                workspacePath: process.cwd(),
                sendToAgent: (prompt) => session.send({ prompt }),
                log,
            });
            const server = await startServer(dashboard);
            entry = { dashboard, server };
            instances.set(ctx.instanceId, entry);

            // Rehydrate from the persisted pointer when the caller did not name
            // a source, so a provider restart restores the same graph.
            const pointer = await loadInstancePointer(ctx.instanceId);
            const fromInput = Boolean(input.path || input.dot || input.args !== undefined);
            const requested = fromInput ? input : { path: pointer?.path };
            if (requested.path || requested.dot || requested.args !== undefined) {
                // Caller-supplied paths are agent-controlled, so confine them to
                // the workspace; the trusted pointer path may point at an
                // absolute artifact and is restored as-is.
                if (await applyOrRecord(dashboard, requested, { workspaceOnly: fromInput })) {
                    restoreViewState(dashboard, pointer);
                }
            }
            entry.disposePersist = autoPersist(dashboard, log);
        } else if (input.path || input.dot || input.args !== undefined) {
            await applyOrRecord(entry.dashboard, input, { workspaceOnly: true });
        }

        await persist(entry.dashboard);
        const snapshot = entry.dashboard.snapshot();
        return {
            title: snapshot.source.loaded ? `PR graph — ${snapshot.source.label}` : "PR graph",
            status: snapshot.source.loaded
                ? `${snapshot.stats.visible.nodes} nodes / ${snapshot.stats.visible.edges} edges`
                : "no graph loaded",
            url: entry.server.url,
        };
    },

    onClose: async (ctx) => {
        const entry = instances.get(ctx.instanceId);
        if (!entry) return;
        instances.delete(ctx.instanceId);
        // Mark closed and cancel the auto-persist listener/timer first, so no
        // pending write can resurrect the pointer we are about to remove.
        entry.dashboard.closed = true;
        entry.disposePersist?.();
        await entry.server.close();
        await removeInstancePointer(ctx.instanceId);
    },
});

session = await joinSession({ canvases: [canvas] });

// A reload replaces this process, but a Graphviz child it spawned would survive
// and keep a core busy on a layout nobody can collect any more.
for (const signal of ["SIGTERM", "SIGINT", "SIGHUP"]) {
    process.on(signal, () => {
        killLayouts();
        process.exit(0);
    });
}
process.on("exit", killLayouts);
