// DOT source parsing, filtering and re-emission for the pr-graph dashboard.
//
// The dashboard consumes `gh team-kit pr-graph --format dot` output, whose
// node identifiers are `<type>:<name>` and whose edge labels are
// `<relation>` or `<relation> (<weight>)`. The parser is intentionally
// tolerant so that hand written or foreign DOT files still render.

/** Node type inferred from the Graphviz shape emitted by `pr-graph`. */
const NODE_TYPE_BY_SHAPE = {
    ellipse: "user",
    doubleoctagon: "team",
    hexagon: "label",
    folder: "directory",
    component: "submodule",
    box: "file",
};

/** Graphviz shape used when re-emitting a node of a known type. */
const SHAPE_BY_NODE_TYPE = {
    user: "ellipse",
    team: "doubleoctagon",
    label: "hexagon",
    directory: "folder",
    submodule: "component",
    file: "box",
    other: "box",
};

/** Node types in display order. */
export const NODE_TYPES = ["user", "team", "label", "directory", "submodule", "file", "other"];

/** Edge relations emitted by `pr-graph`, in display order. */
export const RELATIONS = [
    "approved",
    "changes-requested",
    "reviewed",
    "commented",
    "review-commented",
    "review-requested",
    "member-of",
    "changed",
    "in",
    "owned-by",
    "labeled",
];

const NODE_TYPE_PREFIX = /^(user|team|label|file|directory|submodule):/;
const IDENT_CHAR = /[A-Za-z0-9_.\u0080-\uFFFF]/;

/**
 * Splits DOT source into tokens. Comments are dropped, quoted strings are
 * unescaped, and `->` / `--` are reported as edge operators.
 */
function tokenize(source) {
    const tokens = [];
    const length = source.length;
    let i = 0;
    while (i < length) {
        const ch = source[i];
        if (ch === '"') {
            let j = i + 1;
            let text = "";
            while (j < length) {
                const c = source[j];
                if (c === "\\") {
                    const next = source[j + 1];
                    if (next === '"' || next === "\\") {
                        text += next;
                        j += 2;
                        continue;
                    }
                    if (next === "\n") {
                        j += 2;
                        continue;
                    }
                    text += c;
                    j += 1;
                    continue;
                }
                if (c === '"') {
                    j += 1;
                    break;
                }
                text += c;
                j += 1;
            }
            tokens.push({ kind: "id", value: text, quoted: true });
            i = j;
            continue;
        }
        if (ch === "<") {
            // HTML-like label: capture the balanced <...> block verbatim.
            let depth = 0;
            let j = i;
            let text = "";
            while (j < length) {
                if (source[j] === "<") depth += 1;
                else if (source[j] === ">") {
                    depth -= 1;
                    if (depth === 0) {
                        j += 1;
                        break;
                    }
                }
                text += source[j];
                j += 1;
            }
            tokens.push({ kind: "id", value: text.slice(1), quoted: true });
            i = j;
            continue;
        }
        if (ch === "/" && source[i + 1] === "/") {
            while (i < length && source[i] !== "\n") i += 1;
            continue;
        }
        if (ch === "/" && source[i + 1] === "*") {
            i += 2;
            while (i < length && !(source[i] === "*" && source[i + 1] === "/")) i += 1;
            i += 2;
            continue;
        }
        if (ch === "#") {
            while (i < length && source[i] !== "\n") i += 1;
            continue;
        }
        if (/\s/.test(ch)) {
            i += 1;
            continue;
        }
        if (ch === "-" && (source[i + 1] === ">" || source[i + 1] === "-")) {
            tokens.push({ kind: "edgeop", value: source.slice(i, i + 2) });
            i += 2;
            continue;
        }
        if (ch === "-" && /[0-9.]/.test(source[i + 1] ?? "")) {
            let j = i + 1;
            while (j < length && /[0-9.]/.test(source[j])) j += 1;
            tokens.push({ kind: "id", value: source.slice(i, j) });
            i = j;
            continue;
        }
        if ("{}[];,=:".includes(ch)) {
            tokens.push({ kind: "punct", value: ch });
            i += 1;
            continue;
        }
        let j = i;
        while (j < length && IDENT_CHAR.test(source[j])) j += 1;
        if (j === i) {
            i += 1;
            continue;
        }
        tokens.push({ kind: "id", value: source.slice(i, j) });
        i = j;
    }
    return tokens;
}

/** Splits an edge label such as `changed (3)` into its relation and weight. */
export function parseEdgeLabel(label) {
    if (!label) return { relation: "", weight: 1 };
    const match = /^(.*?)(?:\s*\((\d+)\))?$/.exec(label.trim());
    if (!match) return { relation: label.trim(), weight: 1 };
    return { relation: match[1].trim(), weight: match[2] ? Number(match[2]) : 1 };
}

/** Infers a node type from its identifier prefix, falling back to its shape. */
function inferNodeType(id, shape) {
    const prefix = NODE_TYPE_PREFIX.exec(id);
    if (prefix) return prefix[1];
    const byShape = NODE_TYPE_BY_SHAPE[(shape ?? "").toLowerCase()];
    return byShape ?? "other";
}

/**
 * Parses DOT source into a plain graph object.
 *
 * @returns {{directed: boolean, nodes: Array, edges: Array, warnings: string[]}}
 */
export function parseDot(source) {
    const tokens = tokenize(source);
    const warnings = [];
    const nodeMap = new Map();
    const edges = [];
    const defaults = { node: {}, edge: {} };
    let directed = true;
    let pos = 0;

    const peek = (offset = 0) => tokens[pos + offset];
    const isPunct = (token, value) => token && token.kind === "punct" && token.value === value;
    const isKeyword = (token, value) =>
        token && token.kind === "id" && !token.quoted && token.value.toLowerCase() === value;

    const ensureNode = (id) => {
        let node = nodeMap.get(id);
        if (!node) {
            node = { id, attrs: { ...defaults.node } };
            nodeMap.set(id, node);
        }
        return node;
    };

    const parseAttrLists = () => {
        const attrs = {};
        while (isPunct(peek(), "[")) {
            pos += 1;
            while (peek() && !isPunct(peek(), "]")) {
                if (isPunct(peek(), ",") || isPunct(peek(), ";")) {
                    pos += 1;
                    continue;
                }
                const key = peek();
                if (!key || key.kind !== "id") {
                    pos += 1;
                    continue;
                }
                pos += 1;
                if (isPunct(peek(), "=")) {
                    pos += 1;
                    const value = peek();
                    if (value && value.kind === "id") {
                        attrs[key.value.toLowerCase()] = value.value;
                        pos += 1;
                    }
                } else {
                    attrs[key.value.toLowerCase()] = "true";
                }
            }
            if (isPunct(peek(), "]")) pos += 1;
        }
        return attrs;
    };

    const skipPort = () => {
        while (isPunct(peek(), ":")) {
            pos += 1;
            if (peek() && peek().kind === "id") pos += 1;
        }
    };

    // Header: [strict] (graph|digraph) [name] {
    while (pos < tokens.length && !isPunct(peek(), "{")) {
        if (isKeyword(peek(), "graph")) directed = false;
        pos += 1;
    }
    if (isPunct(peek(), "{")) pos += 1;

    const parseStatements = () => {
        while (pos < tokens.length) {
            const token = peek();
            if (isPunct(token, "}")) {
                pos += 1;
                return;
            }
            if (isPunct(token, ";") || isPunct(token, ",")) {
                pos += 1;
                continue;
            }
            if (isPunct(token, "{")) {
                pos += 1;
                parseStatements();
                continue;
            }
            if (isKeyword(token, "subgraph")) {
                pos += 1;
                if (peek() && peek().kind === "id") pos += 1;
                if (isPunct(peek(), "{")) {
                    pos += 1;
                    parseStatements();
                }
                continue;
            }
            if (isKeyword(token, "node") || isKeyword(token, "edge") || isKeyword(token, "graph")) {
                const keyword = token.value.toLowerCase();
                pos += 1;
                const attrs = parseAttrLists();
                if (keyword === "node" || keyword === "edge") Object.assign(defaults[keyword], attrs);
                continue;
            }
            if (!token || token.kind !== "id") {
                pos += 1;
                continue;
            }

            pos += 1;
            skipPort();

            if (isPunct(peek(), "=")) {
                // Graph-level attribute assignment: ignored by the dashboard.
                pos += 1;
                if (peek()) pos += 1;
                continue;
            }

            const chain = [token.value];
            ensureNode(token.value);
            let malformed = false;
            while (peek() && peek().kind === "edgeop") {
                pos += 1;
                const endpoint = peek();
                if (!endpoint || endpoint.kind !== "id") {
                    warnings.push("skipped an edge with an unsupported endpoint (subgraph endpoints are not supported)");
                    malformed = true;
                    break;
                }
                pos += 1;
                skipPort();
                ensureNode(endpoint.value);
                chain.push(endpoint.value);
            }

            const attrs = parseAttrLists();
            if (malformed) continue;
            if (chain.length === 1) {
                Object.assign(ensureNode(chain[0]).attrs, attrs);
                continue;
            }
            for (let k = 0; k + 1 < chain.length; k += 1) {
                edges.push({ from: chain[k], to: chain[k + 1], attrs: { ...defaults.edge, ...attrs } });
            }
        }
    };

    parseStatements();

    const nodes = [...nodeMap.values()].map((node) => {
        const label = node.attrs.label ?? node.id;
        const type = inferNodeType(node.id, node.attrs.shape);
        return { id: node.id, name: label, type, shape: node.attrs.shape ?? SHAPE_BY_NODE_TYPE[type] };
    });

    const parsedEdges = edges.map((edge) => {
        const label = edge.attrs.label ?? "";
        const { relation, weight } = parseEdgeLabel(label);
        return { from: edge.from, to: edge.to, label, relation, weight };
    });

    return { directed, nodes, edges: parsedEdges, warnings: [...new Set(warnings)] };
}

/** Builds an undirected adjacency map keyed by node id. */
function buildAdjacency(graph) {
    const adjacency = new Map();
    for (const node of graph.nodes) adjacency.set(node.id, new Set());
    for (const edge of graph.edges) {
        if (!adjacency.has(edge.from)) adjacency.set(edge.from, new Set());
        if (!adjacency.has(edge.to)) adjacency.set(edge.to, new Set());
        adjacency.get(edge.from).add(edge.to);
        adjacency.get(edge.to).add(edge.from);
    }
    return adjacency;
}

/** Expands a seed set of node ids by `hops` undirected steps. */
function expand(graph, seeds, hops) {
    const adjacency = buildAdjacency(graph);
    const visited = new Set(seeds);
    let frontier = [...seeds];
    for (let hop = 0; hop < hops; hop += 1) {
        const next = [];
        for (const id of frontier) {
            for (const neighbour of adjacency.get(id) ?? []) {
                if (visited.has(neighbour)) continue;
                visited.add(neighbour);
                next.push(neighbour);
            }
        }
        if (next.length === 0) break;
        frontier = next;
    }
    return visited;
}

/**
 * Applies dashboard filters to a parsed graph, returning a new graph.
 *
 * @param {object} graph Parsed graph from {@link parseDot}.
 * @param {object} filters
 * @param {string[]} [filters.nodeTypes] Keep only these node types (empty = all).
 * @param {string[]} [filters.relations] Keep only these relations (empty = all).
 * @param {number} [filters.minWeight] Drop edges below this weight.
 * @param {string} [filters.search] Keep nodes matching this text and their neighbours.
 * @param {string} [filters.focus] Keep this node and everything within `hops`.
 * @param {number} [filters.hops] Neighbourhood radius for `focus` / `search`.
 * @param {boolean} [filters.keepOrphans] Keep nodes left without any edge.
 */
export function filterGraph(graph, filters = {}) {
    const nodeTypes = new Set(filters.nodeTypes ?? []);
    const relations = new Set(filters.relations ?? []);
    const minWeight = Number(filters.minWeight ?? 0) || 0;
    const hops = Math.max(0, Number(filters.hops ?? 1) || 0);
    const search = (filters.search ?? "").trim().toLowerCase();

    let nodes = graph.nodes;
    if (nodeTypes.size > 0) nodes = nodes.filter((node) => nodeTypes.has(node.type));
    let allowed = new Set(nodes.map((node) => node.id));

    let edges = graph.edges.filter((edge) => {
        if (relations.size > 0 && !relations.has(edge.relation)) return false;
        if (edge.weight < minWeight) return false;
        return allowed.has(edge.from) && allowed.has(edge.to);
    });

    let working = { ...graph, nodes, edges };

    const seeds = new Set();
    if (filters.focus && allowed.has(filters.focus)) seeds.add(filters.focus);
    if (search) {
        for (const node of nodes) {
            if (node.name.toLowerCase().includes(search) || node.id.toLowerCase().includes(search)) {
                seeds.add(node.id);
            }
        }
    }
    if ((filters.focus || search) && seeds.size > 0) {
        const keep = expand(working, seeds, hops);
        nodes = nodes.filter((node) => keep.has(node.id));
        allowed = new Set(nodes.map((node) => node.id));
        edges = edges.filter((edge) => allowed.has(edge.from) && allowed.has(edge.to));
        working = { ...working, nodes, edges };
    }

    if (!filters.keepOrphans) {
        const connected = new Set();
        for (const edge of edges) {
            connected.add(edge.from);
            connected.add(edge.to);
        }
        nodes = nodes.filter((node) => connected.has(node.id));
    }

    return { ...graph, nodes, edges };
}

/** Escapes a string for use as a quoted DOT identifier. */
function quote(value) {
    return `"${String(value).replace(/\\/g, "\\\\").replace(/"/g, '\\"')}"`;
}

/** Escapes a value for use inside an SVG class attribute. */
function cssToken(value) {
    return String(value).replace(/[^A-Za-z0-9_-]/g, "-");
}

/**
 * Re-emits a graph as DOT, adding per-node and per-edge `class` attributes so
 * the dashboard stylesheet can colour the rendered SVG by node type and
 * relation, and a transparent background so the app theme shows through.
 */
export function emitDot(graph, options = {}) {
    const rankdir = options.rankdir ?? "LR";
    const lines = [];
    lines.push("digraph prgraph {");
    lines.push(
        `    graph [bgcolor="transparent" rankdir=${quote(rankdir)} fontname="Helvetica" nodesep=0.25 ranksep=0.5]`,
    );
    lines.push('    node [fontname="Helvetica" fontsize=11 margin="0.08,0.04"]');
    lines.push('    edge [fontname="Helvetica" fontsize=9]');
    for (const node of graph.nodes) {
        const shape = node.shape ?? SHAPE_BY_NODE_TYPE[node.type] ?? "box";
        lines.push(
            `    ${quote(node.id)} [label=${quote(node.name)} shape=${quote(shape)} class=${quote(`nt-${cssToken(node.type)}`)}]`,
        );
    }
    for (const edge of graph.edges) {
        const label = edge.label || edge.relation || "";
        const classes = `rel rel-${cssToken(edge.relation || "unknown")}`;
        lines.push(`    ${quote(edge.from)} -> ${quote(edge.to)} [label=${quote(label)} class=${quote(classes)}]`);
    }
    lines.push("}");
    return lines.join("\n");
}

/** Computes counts, degrees and neighbour lists used by the UI and by actions. */
export function summarize(graph) {
    const nodeTypeCounts = {};
    const relationCounts = {};
    const degrees = new Map();
    for (const node of graph.nodes) {
        nodeTypeCounts[node.type] = (nodeTypeCounts[node.type] ?? 0) + 1;
        degrees.set(node.id, { in: 0, out: 0, weight: 0 });
    }
    for (const edge of graph.edges) {
        relationCounts[edge.relation] = (relationCounts[edge.relation] ?? 0) + 1;
        const from = degrees.get(edge.from);
        if (from) {
            from.out += 1;
            from.weight += edge.weight;
        }
        const to = degrees.get(edge.to);
        if (to) {
            to.in += 1;
            to.weight += edge.weight;
        }
    }
    return { nodeCount: graph.nodes.length, edgeCount: graph.edges.length, nodeTypeCounts, relationCounts, degrees };
}

/** Returns the top nodes by total edge weight, optionally restricted by type. */
export function topNodes(graph, { limit = 10, nodeType } = {}) {
    const { degrees } = summarize(graph);
    return graph.nodes
        .filter((node) => !nodeType || node.type === nodeType)
        .map((node) => {
            const degree = degrees.get(node.id) ?? { in: 0, out: 0, weight: 0 };
            return { id: node.id, name: node.name, type: node.type, ...degree };
        })
        .sort((a, b) => b.weight - a.weight || b.in + b.out - (a.in + a.out) || a.name.localeCompare(b.name))
        .slice(0, limit);
}

/** Returns a node's incoming and outgoing edges resolved to node names. */
export function describeNode(graph, nodeId) {
    const node = graph.nodes.find((candidate) => candidate.id === nodeId);
    if (!node) return null;
    const nameOf = (id) => graph.nodes.find((candidate) => candidate.id === id)?.name ?? id;
    const outgoing = graph.edges
        .filter((edge) => edge.from === nodeId)
        .map((edge) => ({ to: edge.to, name: nameOf(edge.to), relation: edge.relation, weight: edge.weight }));
    const incoming = graph.edges
        .filter((edge) => edge.to === nodeId)
        .map((edge) => ({ from: edge.from, name: nameOf(edge.from), relation: edge.relation, weight: edge.weight }));
    return { ...node, outgoing, incoming };
}
