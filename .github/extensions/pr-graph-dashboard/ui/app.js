// Dashboard front-end. Talks to the extension over loopback HTTP and receives
// push updates via Server-Sent Events.

const $ = (id) => document.getElementById(id);

const NODE_TYPE_LABELS = {
    user: "User",
    team: "Team",
    label: "Label",
    directory: "Directory",
    submodule: "Submodule",
    file: "File",
    other: "Other",
};

const QUICK_FLAGS = [
    "--limit 100",
    "--state merged",
    "--since 2025-01-01",
    "--no-bots",
    "--exclude-draft",
    "--depth 2",
    "--min-weight 2",
    "--edge-type approved,reviewed",
    "--keep-orphans",
];

let state = null;
let renderedRev = -1;
let lastSelectRev = null;
let nodeIds = [];
let checksSignature = "";
let statusTimer = null;
let glideTimer = null;
let skipCenter = 0;
let focusValue = null;
let savedView = null;
let pendingView = null;
let refreshing = null;
let refreshQueued = false;
let renderStart = 0;
let renderActive = false;
let generateArgsRev = null;
let renderHintTimer = null;
let renderTicker = null;
const RENDER_HINT_DELAY = 200;

const view = { scale: 1, x: 0, y: 0, size: { width: 0, height: 0 } };
const MIN_SCALE = 0.01;
const MAX_SCALE = 8;
const GLIDE_MS = 320;

// Per-instance token, handed to the panel in the page fragment (`#t=...`) so it
// is never sent to the server as a query on the top-level load nor leaked via
// Referer. Every API/SSE request must carry it as a `t` query parameter.
const API_TOKEN = new URLSearchParams(location.hash.slice(1)).get("t") ?? "";

function withToken(path) {
    if (!API_TOKEN) return path;
    return `${path}${path.includes("?") ? "&" : "?"}t=${encodeURIComponent(API_TOKEN)}`;
}

async function api(path, options = {}) {
    const response = await fetch(withToken(path), {
        ...options,
        headers: options.body ? { "Content-Type": "application/json" } : undefined,
    });
    const text = await response.text();
    let payload = {};
    if (text) {
        try {
            payload = JSON.parse(text);
        } catch {
            payload = { error: text };
        }
    }
    if (!response.ok || payload.ok === false) {
        throw new Error(payload.error || `request failed: ${response.status}`);
    }
    return payload;
}

const post = (path, body) => api(path, { method: "POST", body: JSON.stringify(body ?? {}) });

function setStatus(message, isError = false) {
    const element = $("status-message");
    element.textContent = message;
    element.classList.toggle("error", isError);
    clearTimeout(statusTimer);
    if (message && !isError) {
        statusTimer = setTimeout(() => {
            element.textContent = "";
        }, 6000);
    }
}

function run(promise) {
    return promise.catch((error) => setStatus(error.message, true));
}

/* ------------------------------------------------------------------ state */

/**
 * Refreshes the panel. Calls are serialised and coalesced: overlapping runs
 * would otherwise move the view while another one is still swapping the SVG.
 */
function refresh() {
    if (refreshing) {
        refreshQueued = true;
        return refreshing;
    }
    refreshing = (async () => {
        try {
            do {
                refreshQueued = false;
                await refreshOnce();
            } while (refreshQueued);
        } finally {
            refreshing = null;
        }
    })();
    return refreshing;
}

async function refreshOnce() {
    state = await api("/api/state");
    followFocus();
    renderState();
    if (state.renderRev !== renderedRev) await loadSvg();
    // A focus move wins over a selection move queued in the same update.
    const moved = applyPendingView();
    followSelection({ move: !moved });
}

/**
 * Tracks the focus filter. Focusing re-lays the graph out, so the view has to
 * travel to the focused node; clearing it goes back to where the user was.
 */
function followFocus() {
    const focus = state.filters.focus ?? "";
    if (focusValue === null) {
        focusValue = focus;
        return;
    }
    if (focus === focusValue) return;
    const wasFocused = Boolean(focusValue);
    focusValue = focus;
    if (focus) {
        if (!wasFocused) savedView = { x: view.x, y: view.y, scale: view.scale, layout: layoutSignature() };
        pendingView = { kind: "focus", node: focus };
        return;
    }
    // Only return to the remembered spot when the graph is laid out as it was.
    const restorable = savedView && savedView.layout === layoutSignature();
    pendingView = restorable ? { kind: "restore", view: savedView } : { kind: "fit" };
    savedView = null;
}

/** Everything except the focus node that changes how the graph is laid out. */
function layoutSignature() {
    const { focus, ...rest } = state.filters;
    const { total } = state.stats;
    return JSON.stringify([state.source.path, total.nodes, total.edges, rest, state.view]);
}

/**
 * Runs a view move that was waiting for the graph to finish re-rendering.
 * Returns true once the move has been applied.
 */
function applyPendingView() {
    if (!pendingView || state.rendering) return false;
    const request = pendingView;
    pendingView = null;
    // The graph content changed underneath, so animating the move is pointless.
    stopGlide();
    if (request.kind === "restore") {
        view.x = request.view.x;
        view.y = request.view.y;
        view.scale = request.view.scale;
        applyTransform();
        return true;
    }
    fitToView();
    if (request.kind === "focus") centerOnNode(request.node, { animate: false });
    return true;
}

/** Moves the view to the selected node whenever the selection is set anew. */
function followSelection({ move = true } = {}) {
    const rev = state.selectRev ?? 0;
    const moved = lastSelectRev !== null && rev !== lastSelectRev;
    lastSelectRev = rev;
    if (!moved) return;
    if (skipCenter > 0) {
        skipCenter -= 1;
        return;
    }
    if (!move || !state.selection) return;
    if (!centerOnNode(state.selection.id)) {
        setStatus(`${state.selection.id} is hidden by the current filters.`);
    }
}

/**
 * Selects a node. Picking one from a list moves the view to it; clicking it in
 * the graph does not, because it is already where the pointer is.
 */
function selectNode(nodeId, { center = true } = {}) {
    if (!center) skipCenter += 1;
    return run(
        post("/api/select", { nodeId }).catch((error) => {
            if (!center) skipCenter = Math.max(0, skipCenter - 1);
            throw error;
        }),
    );
}

async function loadSvg() {
    const rev = state.renderRev;
    const graph = $("graph");
    if (!state.hasSvg) {
        graph.innerHTML = "";
        renderedRev = rev;
        $("empty").hidden = Boolean(state.source.loaded);
        return;
    }
    const response = await fetch(withToken("/api/svg"));
    if (!response.ok) return;
    const svgText = (await response.text()).replace(/^[\s\S]*?(?=<svg\b)/, "");
    const hadContent = renderedRev >= 0 && graph.firstChild;
    graph.innerHTML = svgText;
    renderedRev = rev;
    $("empty").hidden = true;

    const svg = graph.querySelector("svg");
    if (svg) {
        const viewBox = (svg.getAttribute("viewBox") || "").split(/\s+/).map(Number);
        view.size = {
            width: viewBox.length === 4 ? viewBox[2] : svg.clientWidth,
            height: viewBox.length === 4 ? viewBox[3] : svg.clientHeight,
        };
        svg.removeAttribute("width");
        svg.removeAttribute("height");
        svg.setAttribute("width", `${view.size.width}`);
        svg.setAttribute("height", `${view.size.height}`);
    }
    if (!hadContent) fitToView();
    else applyTransform();
    applyHighlight();
}

/* --------------------------------------------------------------- rendering */

function renderState() {
    $("source-label").textContent = state.source.loaded ? state.source.label : "No graph loaded";
    const pathElement = $("source-path");
    pathElement.textContent = state.source.path ?? "";
    pathElement.title = [state.source.path, state.source.command].filter(Boolean).join("\n");

    $("btn-reload").disabled = !state.source.path;
    $("btn-export").disabled = !state.source.loaded;
    $("zoom-center").disabled = !state.selection;

    renderStatusBar();
    renderChecks();
    renderInputs();
    renderFocus();
    void run(renderNodeList());
    renderDetails();
    renderPresets();
    syncAskButtons();
    syncRendering();
    syncGenerateArgs();
    applyHighlight();
}

/**
 * Shows how long Graphviz has been laying the graph out again. The picture on
 * screen is the previous layout until it finishes, so it is dimmed meanwhile.
 */
function syncRendering() {
    const active = Boolean(state.rendering);
    if (active === renderActive) return;
    renderActive = active;
    if (!active) {
        clearTimeout(renderHintTimer);
        clearInterval(renderTicker);
        renderHintTimer = null;
        renderTicker = null;
        renderStart = 0;
        $("rendering").hidden = true;
        $("rendering-elapsed").textContent = "";
        $("graph").classList.remove("stale");
        return;
    }
    renderStart = Date.now();
    // Short renders should not flash the badge.
    renderHintTimer = setTimeout(() => {
        $("rendering").hidden = false;
        $("graph").classList.add("stale");
        renderElapsed();
        renderTicker = setInterval(renderElapsed, 1000);
    }, RENDER_HINT_DELAY);
}

function renderElapsed() {
    const seconds = Math.round((Date.now() - renderStart) / 1000);
    // The limit is shown too, so a long wait has a visible end.
    $("rendering-elapsed").textContent = seconds >= 1 ? `${seconds}s / ${formatLimit(state.view.timeoutMs)}` : "";
}

function renderStatusBar() {
    const { total, visible } = state.stats;
    // Re-rendering a large graph takes a while, and the counts update first.
    const rendering = state.rendering ? " · rendering…" : "";
    $("status-counts").textContent = state.source.loaded
        ? `${visible.nodes}/${total.nodes} nodes · ${visible.edges}/${total.edges} edges${rendering}`
        : "";
    if (state.busy) {
        setStatus(state.busy);
        return;
    }
    const problems = [state.error, state.renderError, ...(state.warnings ?? [])].filter(Boolean);
    if (!state.graphviz?.available) {
        problems.unshift("Graphviz is not installed; run `brew install graphviz` to render graphs.");
    }
    if (problems.length > 0) setStatus(problems[0], true);
}

function buildCheckList(container, entries, selected, onChange) {
    container.replaceChildren();
    for (const entry of entries) {
        const label = document.createElement("label");
        const input = document.createElement("input");
        input.type = "checkbox";
        input.value = entry.value;
        input.checked = selected.includes(entry.value);
        input.addEventListener("change", () => {
            const values = [...container.querySelectorAll("input:checked")].map((element) => element.value);
            onChange(values);
        });
        const swatch = document.createElement("span");
        swatch.className = "swatch";
        swatch.style.background = entry.color;
        const text = document.createElement("span");
        text.textContent = entry.label;
        const count = document.createElement("span");
        count.className = "count";
        count.textContent = String(entry.count ?? 0);
        label.append(input, swatch, text, count);
        container.append(label);
    }
}

function renderChecks() {
    const signature = JSON.stringify([state.stats.nodeTypeCounts, state.stats.relationCounts]);
    const selectionChanged = JSON.stringify([state.filters.nodeTypes, state.filters.relations]);
    if (signature + selectionChanged === checksSignature) return;
    checksSignature = signature + selectionChanged;

    const nodeTypes = state.nodeTypes
        .filter((type) => (state.stats.nodeTypeCounts[type] ?? 0) > 0)
        .map((type) => ({
            value: type,
            label: NODE_TYPE_LABELS[type] ?? type,
            count: state.stats.nodeTypeCounts[type] ?? 0,
            color: `var(--nt-${type})`,
        }));
    buildCheckList($("filter-node-types"), nodeTypes, state.filters.nodeTypes, (values) =>
        run(post("/api/filters", { nodeTypes: values })),
    );

    const known = new Set(state.relations);
    const extra = Object.keys(state.stats.relationCounts).filter((relation) => relation && !known.has(relation));
    const relations = [...state.relations, ...extra]
        .filter((relation) => (state.stats.relationCounts[relation] ?? 0) > 0)
        .map((relation) => ({
            value: relation,
            label: relation,
            count: state.stats.relationCounts[relation] ?? 0,
            color: `var(--rel-${relation}, var(--rel-unknown))`,
        }));
    buildCheckList($("filter-relations"), relations, state.filters.relations, (values) =>
        run(post("/api/filters", { relations: values })),
    );
}

function setValueUnlessFocused(element, value) {
    if (document.activeElement === element) return;
    element.value = value;
}

function renderInputs() {
    setValueUnlessFocused($("filter-search"), state.filters.search);
    setValueUnlessFocused($("filter-hops"), String(state.filters.hops));
    setValueUnlessFocused($("filter-min-weight"), String(state.filters.minWeight));
    $("filter-keep-orphans").checked = state.filters.keepOrphans;

    const engine = $("view-engine");
    if (engine.options.length !== state.engines.length) {
        engine.replaceChildren(
            ...state.engines.map((name) => new Option(name, name)),
        );
    }
    engine.value = state.view.engine;

    const rankdir = $("view-rankdir");
    if (rankdir.options.length !== state.rankdirs.length) {
        rankdir.replaceChildren(...state.rankdirs.map((name) => new Option(name, name)));
    }
    rankdir.value = state.view.rankdir;

    const timeout = $("view-timeout");
    if (timeout.options.length !== state.renderLimits.length) {
        timeout.replaceChildren(
            ...state.renderLimits.map((ms) => new Option(formatLimit(ms), String(ms))),
        );
    }
    timeout.value = String(state.view.timeoutMs);
}

/** Formats a render limit in milliseconds as a compact label such as `2m`. */
function formatLimit(ms) {
    const seconds = Math.round(ms / 1000);
    return seconds < 60 ? `${seconds}s` : `${Math.round(seconds / 60)}m`;
}

function renderFocus() {
    const field = $("focus-field");
    field.hidden = !state.filters.focus;
    if (state.filters.focus) $("focus-chip").textContent = state.filters.focus;
}

async function renderNodeList() {
    const query = $("nodes-query").value.trim();
    const payload = await api(`/api/nodes?limit=200&q=${encodeURIComponent(query)}`);
    nodeIds = payload.nodes.map((node) => node.id);
    const list = $("node-list");
    list.replaceChildren();
    for (const node of payload.nodes) {
        const item = document.createElement("li");
        item.dataset.nodeId = node.id;
        if (state.selection?.id === node.id) item.classList.add("selected");
        const swatch = document.createElement("span");
        swatch.className = "swatch";
        swatch.style.background = `var(--nt-${node.type})`;
        const name = document.createElement("span");
        name.className = "name";
        name.textContent = node.name;
        name.title = node.id;
        const weight = document.createElement("span");
        weight.className = "weight";
        weight.textContent = `${node.in + node.out} / w${node.weight}`;
        item.append(swatch, name, weight);
        list.append(item);
    }
    if (payload.total > payload.nodes.length) {
        const item = document.createElement("li");
        item.className = "muted";
        item.textContent = `…and ${payload.total - payload.nodes.length} more`;
        list.append(item);
    }
}

function relationList(entries, peerKey) {
    const list = document.createElement("ul");
    const sorted = [...entries].sort((a, b) => b.weight - a.weight || a.name.localeCompare(b.name));
    for (const entry of sorted) {
        const item = document.createElement("li");
        const relation = document.createElement("span");
        relation.className = "relation";
        relation.textContent = entry.weight > 1 ? `${entry.relation} ×${entry.weight}` : entry.relation;
        const peer = document.createElement("span");
        peer.className = "peer";
        peer.textContent = entry.name;
        peer.title = entry[peerKey];
        peer.dataset.nodeId = entry[peerKey];
        item.append(relation, peer);
        list.append(item);
    }
    return list;
}

function renderDetails() {
    const container = $("details");
    container.className = "details";
    container.replaceChildren();
    const selection = state.selection;
    if (!selection) {
        const hint = document.createElement("p");
        hint.className = "muted";
        hint.textContent = "Select a node in the graph or in the Nodes tab.";
        container.append(hint);
        return;
    }

    const title = document.createElement("div");
    title.className = "details-title";
    const swatch = document.createElement("span");
    swatch.className = "swatch";
    swatch.style.background = `var(--nt-${selection.type})`;
    const name = document.createElement("span");
    name.textContent = selection.name;
    title.append(swatch, name);

    const meta = document.createElement("p");
    meta.className = "hint muted";
    meta.textContent = `${NODE_TYPE_LABELS[selection.type] ?? selection.type} · ${selection.id}`;

    const buttons = document.createElement("div");
    buttons.className = "chip-row";
    const focusButton = document.createElement("button");
    focusButton.type = "button";
    focusButton.textContent = state.filters.focus === selection.id ? "Clear focus" : "Focus graph here";
    focusButton.addEventListener("click", () =>
        run(post("/api/filters", { focus: state.filters.focus === selection.id ? "" : selection.id })),
    );
    const askButton = document.createElement("button");
    askButton.type = "button";
    askButton.dataset.askButton = "1";
    askButton.dataset.askKey = "selection";
    askButton.dataset.askLabel = "Ask agent";
    askButton.textContent = "Ask agent";
    askButton.addEventListener("click", () =>
        askAgent("selection", { presetId: "selection" }, `Analyse ${selection.name}`),
    );
    buttons.append(focusButton, askButton);

    container.append(title, meta, buttons);

    if (selection.outgoing.length > 0) {
        const heading = document.createElement("h3");
        heading.textContent = `Outgoing (${selection.outgoing.length})`;
        container.append(heading, relationList(selection.outgoing, "to"));
    }
    if (selection.incoming.length > 0) {
        const heading = document.createElement("h3");
        heading.textContent = `Incoming (${selection.incoming.length})`;
        container.append(heading, relationList(selection.incoming, "from"));
    }
}

function renderPresets() {
    const container = $("presets");
    container.replaceChildren();
    for (const preset of state.presets) {
        const button = document.createElement("button");
        button.type = "button";
        button.className = "chip";
        button.textContent = preset.label;
        button.dataset.askButton = "1";
        button.dataset.askKey = `preset:${preset.id}`;
        button.dataset.askLabel = preset.label;
        button.dataset.askBaseDisabled = String(Boolean(preset.needsSelection && !state.selection));
        button.disabled = preset.needsSelection && !state.selection;
        button.addEventListener("click", () =>
            askAgent(`preset:${preset.id}`, { presetId: preset.id, prompt: $("agent-prompt").value.trim() }, preset.label),
        );
        container.append(button);
    }
}

// The agent's reply lands in the chat panel, not in this canvas, so a send is
// otherwise invisible from here and gets repeated. The guard lives in module
// state rather than on the buttons because the panels re-render on push
// updates, which would swap a disabled button for a fresh enabled one.
const ASK_CONFIRM_MS = 2500;
const ASK_LOG_KEY = "pr-graph-dashboard:ask-log";
const ASK_LOG_MAX = 8;
const askState = { busy: false, sentAt: 0, key: "" };
let askSyncTimer = null;
let toastTimer = null;
let askLog = [];

function askConfirming() {
    return askState.sentAt > 0 && Date.now() - askState.sentAt < ASK_CONFIRM_MS;
}

function syncAskButtons() {
    const confirming = askConfirming();
    for (const button of document.querySelectorAll("[data-ask-button]")) {
        const key = button.dataset.askKey ?? "";
        const base = button.dataset.askLabel ?? button.textContent;
        const mine = key === askState.key;
        button.textContent = askState.busy && mine ? "Sending…" : confirming && mine ? "Sent ✓" : base;
        button.disabled = askState.busy || confirming || button.dataset.askBaseDisabled === "true";
    }
    clearTimeout(askSyncTimer);
    if (confirming) {
        askSyncTimer = setTimeout(syncAskButtons, ASK_CONFIRM_MS - (Date.now() - askState.sentAt) + 20);
    }
}

function showToast(title, detail, isError = false) {
    const toast = $("toast");
    toast.replaceChildren();
    const strong = document.createElement("strong");
    strong.textContent = title;
    toast.append(strong);
    if (detail) {
        const span = document.createElement("span");
        span.textContent = detail;
        toast.append(span);
    }
    toast.classList.toggle("error", isError);
    toast.hidden = false;
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => {
        toast.hidden = true;
    }, isError ? 8000 : 6000);
}

function loadAskLog() {
    try {
        const stored = JSON.parse(localStorage.getItem(ASK_LOG_KEY) ?? "[]");
        if (Array.isArray(stored)) askLog = stored.slice(0, ASK_LOG_MAX);
    } catch {
        askLog = [];
    }
    renderAskLog();
}

function renderAskLog() {
    const list = $("ask-log");
    list.replaceChildren();
    for (const entry of askLog) {
        const item = document.createElement("li");
        const time = document.createElement("time");
        time.textContent = entry.time;
        const label = document.createElement("span");
        label.className = "label";
        label.textContent = entry.label;
        label.title = entry.label;
        item.append(time, label);
        list.append(item);
    }
    $("ask-log-wrap").hidden = askLog.length === 0;
}

function recordAsk(label) {
    const now = new Date();
    askLog.unshift({ time: now.toLocaleTimeString(), label });
    askLog = askLog.slice(0, ASK_LOG_MAX);
    try {
        localStorage.setItem(ASK_LOG_KEY, JSON.stringify(askLog));
    } catch {
        /* storage is optional */
    }
    renderAskLog();
}

async function askAgent(key, payload, label) {
    if (askState.busy || askConfirming()) return;
    askState.busy = true;
    askState.key = key;
    syncAskButtons();
    try {
        await post("/api/ask", payload);
        askState.sentAt = Date.now();
        onAsked(label);
    } catch (error) {
        setStatus(error.message, true);
        showToast("Could not send", error.message, true);
    } finally {
        askState.busy = false;
        syncAskButtons();
    }
}

function onAsked(label) {
    $("agent-prompt").value = "";
    const time = new Date().toLocaleTimeString();
    $("agent-status").textContent = `Sent at ${time}.`;
    setStatus("Sent to the agent — its reply appears in the chat.");
    showToast("Sent to the agent", "The reply appears in the chat panel, not here.");
    recordAsk(label ?? "Message");
    // Draw attention to the log if the user is looking at another tab.
    const agentVisible = $("tabs").querySelector('[data-tab="agent"]').classList.contains("active");
    $("agent-badge").hidden = agentVisible;
}

/* ------------------------------------------------------------- graph view */

function applyTransform() {
    clampView();
    $("graph").style.transform = `translate(${view.x}px, ${view.y}px) scale(${view.scale})`;
}

/** Keeps part of the graph inside the viewport so it cannot be scrolled away. */
function clampView() {
    const container = $("graph-scroll");
    const width = view.size.width * view.scale;
    const height = view.size.height * view.scale;
    if (!width || !height) return;
    const margin = 60;
    view.x = Math.max(margin - width, Math.min(container.clientWidth - margin, view.x));
    view.y = Math.max(margin - height, Math.min(container.clientHeight - margin, view.y));
}

function fitToView() {
    const container = $("graph-scroll");
    const { width, height } = view.size;
    if (!width || !height) return;
    const fitted = Math.min(container.clientWidth / width, container.clientHeight / height, 1.5);
    view.scale = Math.min(MAX_SCALE, Math.max(MIN_SCALE, fitted || 1));
    view.x = Math.max(0, (container.clientWidth - width * view.scale) / 2);
    view.y = Math.max(0, (container.clientHeight - height * view.scale) / 2);
    applyTransform();
}

/** Finds the rendered SVG group of a node, or null when it is filtered out. */
function nodeElement(nodeId) {
    for (const element of $("graph").querySelectorAll(".node")) {
        if ((element.querySelector("title")?.textContent ?? "") === nodeId) return element;
    }
    return null;
}

/** Pans the view so the given node sits in the middle of the viewport. */
function centerOnNode(nodeId, { animate = true } = {}) {
    const element = nodeElement(nodeId);
    if (!element) return false;
    const container = $("graph-scroll");
    const node = element.getBoundingClientRect();
    const box = container.getBoundingClientRect();
    view.x += box.left + box.width / 2 - (node.left + node.width / 2);
    view.y += box.top + box.height / 2 - (node.top + node.height / 2);
    if (animate) glide();
    applyTransform();
    return true;
}

/** Animates the next transform so a long jump stays easy to follow. */
function glide() {
    const graph = $("graph");
    graph.classList.add("gliding");
    clearTimeout(glideTimer);
    glideTimer = setTimeout(() => graph.classList.remove("gliding"), GLIDE_MS);
}

/** Drops the animation so panning and zooming stay immediate. */
function stopGlide() {
    if (glideTimer === null) return;
    clearTimeout(glideTimer);
    glideTimer = null;
    $("graph").classList.remove("gliding");
}

function zoomBy(factor, originX, originY) {
    const container = $("graph-scroll");
    const rect = container.getBoundingClientRect();
    zoomTo(view.scale * factor, originX ?? rect.width / 2, originY ?? rect.height / 2);
}

/** Applies an absolute scale while keeping the container point (cx, cy) fixed. */
function zoomTo(scale, cx, cy, base = view) {
    const next = Math.min(MAX_SCALE, Math.max(MIN_SCALE, scale));
    const ratio = next / base.scale;
    view.x = cx - (cx - base.x) * ratio;
    view.y = cy - (cy - base.y) * ratio;
    view.scale = next;
    applyTransform();
}

/** Converts a wheel delta to pixels regardless of the reported delta mode. */
function wheelDelta(value, mode, pageSize) {
    if (mode === 1) return value * 16;
    if (mode === 2) return value * pageSize;
    return value;
}

/** Splits a Graphviz edge title such as `user:a->file:b` into its endpoints. */
function splitEdgeTitle(title) {
    const known = new Set(nodeIds);
    let index = title.indexOf("->");
    while (index >= 0) {
        const from = title.slice(0, index);
        const to = title.slice(index + 2);
        if (known.size === 0 || known.has(from) || known.has(to)) return [from, to];
        index = title.indexOf("->", index + 1);
    }
    const fallback = title.split("->");
    return fallback.length === 2 ? fallback : null;
}

function applyHighlight() {
    const graph = $("graph");
    const selection = state?.selection ?? null;
    graph.classList.toggle("has-selection", Boolean(selection));
    for (const element of graph.querySelectorAll(".node, .edge")) {
        element.classList.remove("hl", "selected");
    }
    if (!selection) return;

    const neighbours = new Set([selection.id]);
    for (const entry of selection.outgoing) neighbours.add(entry.to);
    for (const entry of selection.incoming) neighbours.add(entry.from);

    for (const element of graph.querySelectorAll(".node")) {
        const id = element.querySelector("title")?.textContent ?? "";
        if (id === selection.id) element.classList.add("hl", "selected");
        else if (neighbours.has(id)) element.classList.add("hl");
    }
    for (const element of graph.querySelectorAll(".edge")) {
        const title = element.querySelector("title")?.textContent ?? "";
        const endpoints = splitEdgeTitle(title);
        if (endpoints && (endpoints[0] === selection.id || endpoints[1] === selection.id)) {
            element.classList.add("hl");
        }
    }
}

/* ------------------------------------------------------------- side panel */

const SIDEBAR_STORAGE_KEY = "pr-graph-dashboard:sidebar";
const MIN_SIDEBAR = 200;
const MIN_VIEWPORT = 160;

/** True while the layout stacks the side panel below the graph. */
function isStacked() {
    return window.matchMedia("(max-width: 780px)").matches;
}

function sidebarSize() {
    const body = document.querySelector(".body");
    return isStacked() ? $("sidebar").offsetHeight : $("sidebar").offsetWidth || body.offsetWidth * 0.3;
}

function setSidebarSize(size) {
    const body = document.querySelector(".body");
    const stacked = isStacked();
    const divider = stacked ? $("resizer").offsetHeight : $("resizer").offsetWidth;
    const total = (stacked ? body.clientHeight : body.clientWidth) - divider;
    const clamped = Math.max(MIN_SIDEBAR, Math.min(total - MIN_VIEWPORT, size));
    body.style.setProperty(stacked ? "--sidebar-height" : "--sidebar-width", `${Math.round(clamped)}px`);
    try {
        localStorage.setItem(`${SIDEBAR_STORAGE_KEY}:${stacked ? "height" : "width"}`, String(Math.round(clamped)));
    } catch {
        // Storage can be unavailable; the size simply is not remembered.
    }
    applyTransform();
}

function restoreSidebarSize() {
    const body = document.querySelector(".body");
    const stacked = isStacked();
    let stored = null;
    try {
        stored = localStorage.getItem(`${SIDEBAR_STORAGE_KEY}:${stacked ? "height" : "width"}`);
    } catch {
        stored = null;
    }
    const size = Number(stored);
    if (!Number.isFinite(size) || size <= 0) return;
    const total = stacked ? body.clientHeight : body.clientWidth;
    if (total <= MIN_SIDEBAR + MIN_VIEWPORT) return;
    setSidebarSize(size);
}

// --- Generate dialog: argument completion and agent-proposed arguments ---

// Candidates come from the CLI's own shell-completion machinery, so they never
// drift from the installed version of `gh team-kit`.
const completion = { open: false, items: [], index: -1, seq: 0, timer: null };
const COMPLETE_DEBOUNCE_MS = 150;
const COMPLETE_HINT =
    "Completion comes from the CLI itself. Ctrl+Space to list, Tab or Enter to accept, Esc to dismiss.";
const ASK_ARGS_HINT = "The agent replies by filling the field above; nothing runs until you press Run.";

function openGenerateDialog(args) {
    $("generate-args").value = args;
    $("generate-ask").value = "";
    $("generate-ask-hint").textContent = ASK_ARGS_HINT;
    $("generate-complete-hint").textContent = COMPLETE_HINT;
    closeSuggestions();
    const dialog = $("dialog-generate");
    if (!dialog.open) dialog.showModal();
    // Preselect: the app intercepts Cmd+A, so typing over the old arguments is
    // the quickest way to replace them.
    selectWholeField($("generate-args"));
}

function selectWholeField(field) {
    field.focus();
    field.setSelectionRange(0, field.value.length);
}

/** The whitespace-delimited token the caret currently sits in. */
function partialAtCaret(field) {
    const caret = field.selectionStart ?? field.value.length;
    return (field.value.slice(0, caret).match(/\S*$/) ?? [""])[0];
}

function closeSuggestions() {
    clearTimeout(completion.timer);
    completion.open = false;
    completion.items = [];
    completion.index = -1;
    const list = $("generate-suggestions");
    list.replaceChildren();
    list.hidden = true;
}

function scheduleSuggestions() {
    clearTimeout(completion.timer);
    completion.timer = setTimeout(() => void fetchSuggestions(), COMPLETE_DEBOUNCE_MS);
}

async function fetchSuggestions() {
    const field = $("generate-args");
    const caret = field.selectionStart ?? field.value.length;
    const line = field.value.slice(0, caret);
    const seq = (completion.seq += 1);
    try {
        const result = await api(`/api/complete?line=${encodeURIComponent(line)}`);
        if (seq !== completion.seq) return; // a newer keystroke superseded this one
        completion.items = result.candidates ?? [];
        completion.index = completion.items.length ? 0 : -1;
        renderSuggestions();
    } catch (error) {
        if (seq !== completion.seq) return;
        closeSuggestions();
        $("generate-complete-hint").textContent = `Completion unavailable: ${error.message}`;
    }
}

function renderSuggestions() {
    const list = $("generate-suggestions");
    list.replaceChildren();
    for (const [index, item] of completion.items.entries()) {
        const li = document.createElement("li");
        li.setAttribute("role", "option");
        li.classList.toggle("active", index === completion.index);
        const value = document.createElement("span");
        value.className = "value";
        value.textContent = item.value;
        li.append(value);
        if (item.description) {
            const description = document.createElement("span");
            description.className = "description";
            description.textContent = item.description;
            description.title = item.description;
            li.append(description);
        }
        // mousedown, not click: the textarea must not lose focus before we insert.
        li.addEventListener("mousedown", (event) => {
            event.preventDefault();
            acceptSuggestion(index);
        });
        list.append(li);
    }
    completion.open = completion.items.length > 0;
    list.hidden = !completion.open;
    if (completion.open) scrollSuggestionIntoView();
}

function scrollSuggestionIntoView() {
    const active = $("generate-suggestions").children[completion.index];
    active?.scrollIntoView({ block: "nearest" });
}

function moveSuggestion(delta) {
    if (!completion.items.length) return;
    const count = completion.items.length;
    completion.index = (completion.index + delta + count) % count;
    for (const [index, li] of [...$("generate-suggestions").children].entries()) {
        li.classList.toggle("active", index === completion.index);
    }
    scrollSuggestionIntoView();
}

function acceptSuggestion(index) {
    const item = completion.items[index];
    const field = $("generate-args");
    if (!item) return;
    const caret = field.selectionStart ?? field.value.length;
    const start = caret - partialAtCaret(field).length;
    const insert = `${item.value} `;
    field.value = field.value.slice(0, start) + insert + field.value.slice(caret);
    const next = start + insert.length;
    field.setSelectionRange(next, next);
    closeSuggestions();
    field.focus();
}

function wireGenerateCompletion() {
    const field = $("generate-args");
    field.addEventListener("input", scheduleSuggestions);
    field.addEventListener("blur", () => closeSuggestions());
    field.addEventListener("keydown", (event) => {
        if (event.key === " " && (event.ctrlKey || event.metaKey)) {
            event.preventDefault();
            void fetchSuggestions();
            return;
        }
        if (!completion.open) return;
        if (event.key === "ArrowDown") {
            event.preventDefault();
            moveSuggestion(1);
        } else if (event.key === "ArrowUp") {
            event.preventDefault();
            moveSuggestion(-1);
        } else if (event.key === "Tab" || event.key === "Enter") {
            event.preventDefault();
            acceptSuggestion(completion.index);
        } else if (event.key === "Escape") {
            event.preventDefault();
            closeSuggestions();
        }
    });
}

async function askForArgs() {
    const prompt = $("generate-ask").value.trim();
    if (!prompt) return;
    const button = $("generate-ask-send");
    button.disabled = true;
    $("generate-ask-hint").textContent = "Asking the agent…";
    try {
        await post("/api/generate/ask", { prompt });
        recordAsk(`args: ${prompt}`);
        closeSuggestions();
        $("dialog-generate").close();
        setStatus("Asked the agent for arguments…");
        showToast("Asked the agent", "The dialog reopens when arguments are proposed.");
    } catch (error) {
        $("generate-ask-hint").textContent = error.message;
        showToast("Could not send", error.message, true);
    } finally {
        button.disabled = false;
    }
}

/** Reopen the dialog whenever the agent proposes a new set of arguments. */
function syncGenerateArgs() {
    const rev = state?.generateArgsRev ?? 0;
    if (rev === generateArgsRev) return;
    const known = generateArgsRev !== null;
    generateArgsRev = rev;
    if (!known) return; // first snapshot after a reload: adopt the revision silently
    openGenerateDialog(state.generateArgs ?? "");
    $("generate-ask-hint").textContent = "The agent proposed these arguments. Review, then press Run.";
    showToast("Arguments proposed", state.generateArgs || "(empty)");
}

const TEXT_INPUT_TYPES = new Set(["text", "search", "url", "email", "tel", "number", "password"]);

function isTextEntry(element) {
    if (!element) return false;
    if (element.tagName === "TEXTAREA") return true;
    return element.tagName === "INPUT" && TEXT_INPUT_TYPES.has(element.type);
}

// The host delivers some keys to the canvas as text rather than as key events:
// arrows arrive as the classic Mac control codes (U+001C-U+001F), and macOS
// function keys as private-use characters. Both would land in the field as
// unprintable garbage, so they are filtered out here.
const STRAY_CHARS = /[\u0000-\u0008\u000B\u000C\u000E-\u001F\u007F-\u009F\uF700-\uF8FF]/;
const STRAY_CHARS_ALL = /[\u0000-\u0008\u000B\u000C\u000E-\u001F\u007F-\u009F\uF700-\uF8FF]/g;
const CARET_MOVES = {
    "\u001c": { key: "ArrowLeft", delta: -1 },
    "\u001d": { key: "ArrowRight", delta: 1 },
    "\uF702": { key: "ArrowLeft", delta: -1 },
    "\uF703": { key: "ArrowRight", delta: 1 },
};
let lastArrowKey = { key: "", at: 0 };

function moveCaret(field, delta) {
    try {
        const start = field.selectionStart ?? 0;
        const end = field.selectionEnd ?? start;
        // A collapsed selection steps; a range collapses to the chosen side.
        const next = start === end ? Math.min(Math.max(start + delta, 0), field.value.length) : delta < 0 ? start : end;
        field.setSelectionRange(next, next);
    } catch {
        /* not every input type exposes a selection */
    }
}

function setCaret(field, position) {
    try {
        field.setSelectionRange(position, position);
    } catch {
        /* not every input type exposes a selection */
    }
}

/** Keeps stray control and function-key characters out of the text fields. */
function wireInputSanitizer() {
    window.addEventListener(
        "keydown",
        (event) => {
            if (event.key?.startsWith("Arrow")) lastArrowKey = { key: event.key, at: performance.now() };
        },
        true,
    );
    document.addEventListener(
        "beforeinput",
        (event) => {
            const field = event.target;
            if (!isTextEntry(field) || typeof event.data !== "string" || !STRAY_CHARS.test(event.data)) return;
            const cleaned = event.data.replace(STRAY_CHARS_ALL, "");
            // Mixed content (a paste, say) is let through and scrubbed afterwards.
            if (cleaned) return;
            event.preventDefault();
            const move = CARET_MOVES[event.data];
            // Only stand in for the arrow key when the key event itself never
            // arrived; otherwise the browser has already moved the caret.
            const handled = move && lastArrowKey.key === move.key && performance.now() - lastArrowKey.at < 100;
            if (move && !handled) moveCaret(field, move.delta);
        },
        true,
    );
    // Fallback for hosts that write the character without a cancellable event.
    document.addEventListener(
        "input",
        (event) => {
            const field = event.target;
            if (!isTextEntry(field) || !STRAY_CHARS.test(field.value)) return;
            const caret = field.selectionStart ?? field.value.length;
            const before = field.value.slice(0, caret).replace(STRAY_CHARS_ALL, "").length;
            field.value = field.value.replace(STRAY_CHARS_ALL, "");
            setCaret(field, before);
        },
        true,
    );
}

function wireResizer() {
    const resizer = $("resizer");
    let drag = null;

    resizer.addEventListener("pointerdown", (event) => {
        if (event.button !== 0) return;
        event.preventDefault();
        drag = { x: event.clientX, y: event.clientY, size: sidebarSize(), stacked: isStacked() };
        resizer.setPointerCapture(event.pointerId);
        resizer.classList.add("active");
    });

    resizer.addEventListener("pointermove", (event) => {
        if (!drag) return;
        // The side panel sits after the divider, so it grows as the pointer
        // moves towards the start of the axis.
        const delta = drag.stacked ? drag.y - event.clientY : drag.x - event.clientX;
        setSidebarSize(drag.size + delta);
    });

    const stop = (event) => {
        if (!drag) return;
        drag = null;
        resizer.classList.remove("active");
        if (resizer.hasPointerCapture?.(event.pointerId)) resizer.releasePointerCapture(event.pointerId);
    };
    resizer.addEventListener("pointerup", stop);
    resizer.addEventListener("pointercancel", stop);

    resizer.addEventListener("dblclick", () => setSidebarSize(300));

    resizer.addEventListener("keydown", (event) => {
        const step = event.shiftKey ? 40 : 12;
        const grow = isStacked() ? "ArrowUp" : "ArrowLeft";
        const shrink = isStacked() ? "ArrowDown" : "ArrowRight";
        if (event.key !== grow && event.key !== shrink) return;
        event.preventDefault();
        setSidebarSize(sidebarSize() + (event.key === grow ? step : -step));
    });

    restoreSidebarSize();
}

/* ----------------------------------------------------------------- events */

function debounce(fn, delay) {
    let timer = null;
    return (...args) => {
        clearTimeout(timer);
        timer = setTimeout(() => fn(...args), delay);
    };
}

function wireGraphInteractions() {
    const container = $("graph-scroll");
    let dragging = null;
    let lastClick = null;

    container.addEventListener("wheel", (event) => {
        event.preventDefault();
        stopGlide();
        const rect = container.getBoundingClientRect();
        const dy = wheelDelta(event.deltaY, event.deltaMode, rect.height);
        // Trackpad pinch gestures are reported as a wheel event with ctrlKey set.
        if (event.ctrlKey || event.metaKey) {
            zoomTo(view.scale * Math.exp(-dy / 320), event.clientX - rect.left, event.clientY - rect.top);
            return;
        }
        view.x -= wheelDelta(event.deltaX, event.deltaMode, rect.width);
        view.y -= dy;
        applyTransform();
    }, { passive: false });

    container.addEventListener("pointerdown", (event) => {
        if (event.button !== 0) return;
        stopGlide();
        const rect = container.getBoundingClientRect();
        dragging = {
            x: event.clientX,
            y: event.clientY,
            cx: event.clientX - rect.left,
            cy: event.clientY - rect.top,
            base: { x: view.x, y: view.y, scale: view.scale },
            // Pointer capture retargets later events to the container, so the
            // element actually under the pointer has to be captured up front.
            target: event.target,
            moved: false,
        };
        container.setPointerCapture(event.pointerId);
        container.classList.add("dragging");
    });

    container.addEventListener("pointermove", (event) => {
        if (!dragging) return;
        const dy = event.clientY - dragging.y;
        if (Math.abs(event.clientX - dragging.x) > 3 || Math.abs(dy) > 3) dragging.moved = true;
        // Dragging upwards zooms in, matching the Ctrl+wheel direction.
        zoomTo(dragging.base.scale * Math.exp(-dy / 220), dragging.cx, dragging.cy, dragging.base);
    });

    const endDrag = (event) => {
        if (!dragging) return;
        const { moved, target } = dragging;
        dragging = null;
        container.classList.remove("dragging");
        if (container.hasPointerCapture?.(event.pointerId)) container.releasePointerCapture(event.pointerId);
        if (moved) return;
        const node = target?.closest?.(".node");
        const id = node?.querySelector("title")?.textContent ?? null;
        // Double clicks are detected here rather than through a `dblclick`
        // listener, because pointer capture retargets the derived mouse events
        // away from the node that was actually clicked.
        if (id && lastClick && lastClick.id === id && event.timeStamp - lastClick.time < 400) {
            lastClick = null;
            run(post("/api/filters", { focus: id }));
            return;
        }
        lastClick = id ? { id, time: event.timeStamp } : null;
        if (id) activateTab("details");
        selectNode(id, { center: false });
    };
    container.addEventListener("pointerup", endDrag);
    container.addEventListener("pointercancel", () => {
        dragging = null;
        container.classList.remove("dragging");
    });
}

/** Activates a sidebar tab by its `data-tab` name. */
function activateTab(name) {
    for (const tab of $("tabs").querySelectorAll("button")) {
        tab.classList.toggle("active", tab.dataset.tab === name);
    }
    for (const panel of document.querySelectorAll(".panel")) {
        panel.classList.toggle("active", panel.dataset.panel === name);
    }
    if (name === "agent") $("agent-badge").hidden = true;
}

function wireSidebar() {
    $("tabs").addEventListener("click", (event) => {
        const button = event.target.closest("button[data-tab]");
        if (!button) return;
        activateTab(button.dataset.tab);
    });

    $("btn-sidebar").addEventListener("click", () => {
        document.querySelector(".body").classList.toggle("collapsed");
        setTimeout(fitToView, 0);
    });

    const pushSearch = debounce((value) => run(post("/api/filters", { search: value })), 300);
    $("filter-search").addEventListener("input", (event) => pushSearch(event.target.value));
    $("filter-hops").addEventListener("change", (event) => run(post("/api/filters", { hops: event.target.value })));
    $("filter-min-weight").addEventListener("change", (event) =>
        run(post("/api/filters", { minWeight: event.target.value })),
    );
    $("filter-keep-orphans").addEventListener("change", (event) =>
        run(post("/api/filters", { keepOrphans: event.target.checked })),
    );
    $("filters-reset").addEventListener("click", () => run(post("/api/filters", { reset: true })));
    $("focus-clear").addEventListener("click", () => run(post("/api/filters", { focus: "" })));
    $("view-engine").addEventListener("change", (event) => run(post("/api/view", { engine: event.target.value })));
    $("view-rankdir").addEventListener("change", (event) => run(post("/api/view", { rankdir: event.target.value })));
    $("view-timeout").addEventListener("change", (event) =>
        run(post("/api/view", { timeoutMs: Number(event.target.value) })),
    );

    $("nodes-query").addEventListener("input", debounce(() => run(renderNodeList()), 200));
    $("node-list").addEventListener("click", (event) => {
        const item = event.target.closest("li[data-node-id]");
        if (item) selectNode(item.dataset.nodeId);
    });
    $("node-list").addEventListener("dblclick", (event) => {
        const item = event.target.closest("li[data-node-id]");
        if (item) run(post("/api/filters", { focus: item.dataset.nodeId }));
    });
    $("details").addEventListener("click", (event) => {
        const peer = event.target.closest(".peer[data-node-id]");
        if (peer) selectNode(peer.dataset.nodeId);
    });

    $("agent-send").addEventListener("click", () => {
        const prompt = $("agent-prompt").value.trim();
        if (!prompt) {
            setStatus("Enter a message first.", true);
            return;
        }
        askAgent("free", { prompt }, prompt.length > 60 ? `${prompt.slice(0, 60)}…` : prompt);
    });
}

/* ---------------------------------------------------------- file browser */

// The host webview offers the canvas no usable native picker — `<input
// type="file">` hides the absolute path and cannot choose a save destination —
// so both dialogs navigate the filesystem themselves through /api/browse.

const FOUND_DIR = "@found";

const FOLDER_ICON =
    '<svg viewBox="0 0 16 16" width="13" height="13" aria-hidden="true"><path fill="currentColor" d="M1.5 2.5h4.2l1.3 1.6h7.5c.3 0 .5.2.5.5v8.4c0 .3-.2.5-.5.5h-13c-.3 0-.5-.2-.5-.5V3c0-.3.2-.5.5-.5Z"/></svg>';
const FILE_ICON =
    '<svg viewBox="0 0 16 16" width="13" height="13" aria-hidden="true"><path fill="currentColor" d="M3.5 1h6l3 3v10.5c0 .3-.2.5-.5.5H3.5a.5.5 0 0 1-.5-.5v-13c0-.3.2-.5.5-.5Zm5.8 1.2v2.4h2.4L9.3 2.2Z"/></svg>';

/**
 * Renders a directory browser inside `root`.
 *
 * @param {HTMLElement} root
 * @param {{extensions?: string[], showFound?: boolean, onSelect?: Function, onConfirm?: Function}} options
 */
function createFileBrowser(root, { extensions = [], showFound = false, onSelect, onConfirm } = {}) {
    root.innerHTML = `
        <div class="browser-bar">
            <button type="button" class="browser-up" title="Parent directory" aria-label="Parent directory">↑</button>
            <input type="text" class="browser-dir" spellcheck="false" autocomplete="off"
                aria-label="Current directory" />
        </div>
        <div class="browser-places"></div>
        <ul class="browser-list" tabindex="0"></ul>
        <div class="browser-foot">
            <label class="inline"><input type="checkbox" class="browser-all" /> Show every file</label>
            <span class="browser-note muted"></span>
        </div>`;

    const upButton = root.querySelector(".browser-up");
    const dirInput = root.querySelector(".browser-dir");
    const placesRow = root.querySelector(".browser-places");
    const list = root.querySelector(".browser-list");
    const allToggle = root.querySelector(".browser-all");
    const note = root.querySelector(".browser-note");

    const own = {
        dir: "",
        sep: "/",
        parent: null,
        virtual: false,
        entries: [],
        selected: null,
        extensions,
        shownDir: "",
        ticket: 0,
    };

    const setNote = (text, isError = false) => {
        note.textContent = text;
        note.classList.toggle("error", isError);
    };

    const visibleEntries = () =>
        own.entries.filter((entry) => allToggle.checked || (entry.match && !entry.hidden));

    const entryByPath = (target) => own.entries.find((entry) => entry.path === target) ?? null;

    function renderList() {
        list.replaceChildren();
        const entries = visibleEntries();
        if (entries.length === 0) {
            const empty = document.createElement("li");
            empty.className = "empty muted";
            empty.textContent = own.entries.length ? "Nothing matches the filter." : "This folder is empty.";
            list.append(empty);
            return;
        }
        for (const entry of entries) {
            const item = document.createElement("li");
            item.dataset.path = entry.path;
            item.className = entry.kind === "dir" ? "dir" : "file";
            if (!entry.match) item.classList.add("dim");
            if (entry.path === own.selected) item.classList.add("selected");

            const icon = document.createElement("span");
            icon.className = "icon";
            icon.innerHTML = entry.kind === "dir" ? FOLDER_ICON : FILE_ICON;

            const name = document.createElement("span");
            name.className = "name";
            name.textContent = entry.name;
            name.title = entry.path;

            item.append(icon, name);
            if (entry.detail) {
                const detail = document.createElement("span");
                detail.className = "detail";
                detail.textContent = entry.detail;
                detail.title = entry.detail;
                item.append(detail);
            }
            list.append(item);
        }
    }

    function renderPlaces(places = []) {
        placesRow.replaceChildren();
        for (const place of places) {
            if (place.path === FOUND_DIR && !showFound) continue;
            const chip = document.createElement("button");
            chip.type = "button";
            chip.className = "chip";
            chip.textContent = place.label;
            chip.title = place.path;
            if (place.path === own.dir) chip.classList.add("active");
            chip.addEventListener("click", () => void go(place.path));
            placesRow.append(chip);
        }
    }

    function select(entry) {
        own.selected = entry?.path ?? null;
        for (const item of list.children) {
            item.classList?.toggle("selected", item.dataset?.path === own.selected);
        }
        list.querySelector("li.selected")?.scrollIntoView({ block: "nearest" });
        onSelect?.(entry ?? null);
    }

    async function go(target, { select: preselect } = {}) {
        // Overlapping navigations resolve out of order, so only the newest wins.
        const ticket = (own.ticket += 1);
        setNote("Loading…");
        try {
            const query = new URLSearchParams({ dir: target ?? "" });
            if (own.extensions.length) query.set("ext", own.extensions.join(","));
            const data = await api(`/api/browse?${query}`);
            if (ticket !== own.ticket) return;
            own.dir = data.dir;
            own.parent = data.parent;
            own.sep = data.sep || "/";
            own.virtual = Boolean(data.virtual);
            own.entries = data.entries ?? [];
            own.selected = null;
            // A listing that lands while the location bar is being edited must
            // not overwrite what the user typed.
            const edited = document.activeElement === dirInput && dirInput.value !== own.shownDir;
            own.shownDir = data.label ?? data.dir;
            if (!edited) dirInput.value = own.shownDir;
            dirInput.disabled = own.virtual;
            upButton.disabled = !data.parent;
            renderPlaces(data.places);
            renderList();
            const wanted = preselect ?? data.select;
            const entry = wanted ? entryByPath(wanted) : null;
            if (entry) select(entry);
            else onSelect?.(null);
            setNote(data.truncated ? "Showing the first entries only." : "");
        } catch (error) {
            if (ticket === own.ticket) setNote(error.message, true);
        }
    }

    function clear() {
        own.entries = [];
        own.selected = null;
        list.replaceChildren();
        upButton.disabled = true;
        setNote("Loading…");
        onSelect?.(null);
    }

    function step(delta) {
        const entries = visibleEntries();
        if (entries.length === 0) return;
        const current = entries.findIndex((entry) => entry.path === own.selected);
        const next = current < 0 ? (delta > 0 ? 0 : entries.length - 1) : current + delta;
        select(entries[Math.min(entries.length - 1, Math.max(0, next))]);
    }

    upButton.addEventListener("click", () => {
        if (own.parent) void go(own.parent);
    });

    dirInput.addEventListener("keydown", (event) => {
        if (event.key !== "Enter") return;
        event.preventDefault();
        void go(dirInput.value.trim());
    });

    allToggle.addEventListener("change", renderList);

    list.addEventListener("click", (event) => {
        const item = event.target.closest("li[data-path]");
        if (!item) return;
        const entry = entryByPath(item.dataset.path);
        if (!entry) return;
        list.focus();
        if (entry.kind === "dir") void go(entry.path);
        else select(entry);
    });

    list.addEventListener("dblclick", (event) => {
        const item = event.target.closest("li[data-path]");
        const entry = item ? entryByPath(item.dataset.path) : null;
        if (entry?.kind === "file") onConfirm?.(entry);
    });

    list.addEventListener("keydown", (event) => {
        if (event.key === "ArrowDown") step(1);
        else if (event.key === "ArrowUp") step(-1);
        else if (event.key === "Backspace" && own.parent) void go(own.parent);
        else if (event.key === "Enter") {
            const entry = own.selected ? entryByPath(own.selected) : null;
            if (!entry) return;
            if (entry.kind === "dir") void go(entry.path);
            else onConfirm?.(entry);
        } else return;
        event.preventDefault();
    });

    return {
        go,
        open(dir, options = {}) {
            if (options.extensions) own.extensions = options.extensions;
            clear();
            return go(dir, options);
        },
        dir: () => own.dir,
        sep: () => own.sep,
        selected: () => (own.selected ? entryByPath(own.selected) : null),
        focus: () => list.focus(),
        setExtensions(next) {
            own.extensions = next;
            return go(own.dir);
        },
    };
}

/** Joins a browser directory with a typed file name. */
function joinPath(dir, name, sep = "/") {
    const value = name.trim();
    if (!value) return "";
    if (value.startsWith("/") || value.startsWith("~")) return value;
    if (!dir) return value;
    return dir.endsWith(sep) ? `${dir}${value}` : `${dir}${sep}${value}`;
}

let openBrowser = null;
let exportBrowser = null;

function wireOpenDialog() {
    const dialog = $("dialog-open");
    const picked = $("open-picked");

    openBrowser = createFileBrowser($("open-browser"), {
        extensions: ["dot", "gv"],
        showFound: true,
        onSelect: (entry) => {
            picked.textContent = entry ? entry.path : "nothing yet";
            picked.classList.toggle("muted", !entry);
        },
        onConfirm: (entry) => {
            dialog.close();
            run(post("/api/load", { path: entry.path }));
        },
    });

    $("btn-open").addEventListener("click", () => {
        dialog.showModal();
        const source = state?.source.path;
        void openBrowser.open(source || FOUND_DIR, { select: source ?? undefined }).then(() => openBrowser.focus());
    });

    $("open-confirm").addEventListener("click", () => {
        const entry = openBrowser.selected();
        if (!entry) {
            setStatus("Pick a file first.", true);
            return;
        }
        dialog.close();
        run(post("/api/load", { path: entry.path }));
    });
}

function wireExportDialog() {
    const dialog = $("dialog-export");
    const nameField = $("export-name");
    const picked = $("export-picked");

    const syncPicked = () => {
        const target = joinPath(exportBrowser.dir(), nameField.value, exportBrowser.sep());
        picked.textContent = target || "…";
    };

    const save = () => {
        const target = joinPath(exportBrowser.dir(), nameField.value, exportBrowser.sep());
        if (!target) {
            setStatus("Enter a file name.", true);
            nameField.focus();
            return;
        }
        dialog.close();
        run(post("/api/export", { kind: $("export-kind").value, path: target }).then((result) => setStatus(`Saved ${result.path}`)));
    };

    exportBrowser = createFileBrowser($("export-browser"), {
        extensions: ["svg"],
        onSelect: (entry) => {
            // Picking an existing file means "overwrite this one".
            if (entry) nameField.value = entry.name;
            syncPicked();
        },
        onConfirm: save,
    });

    nameField.addEventListener("input", syncPicked);
    nameField.addEventListener("keydown", (event) => {
        if (event.key !== "Enter") return;
        event.preventDefault();
        save();
    });

    $("btn-export").addEventListener("click", () => {
        const kind = $("export-kind").value;
        nameField.value = defaultExportName(kind);
        dialog.showModal();
        void exportBrowser
            .open(defaultExportDir(), { extensions: exportExtensions(kind) })
            .then(syncPicked);
    });

    $("export-kind").addEventListener("change", (event) => {
        nameField.value = defaultExportName(event.target.value);
        void exportBrowser.setExtensions(exportExtensions(event.target.value)).then(syncPicked);
    });

    $("export-confirm").addEventListener("click", save);
}

function wireToolbar() {
    $("zoom-in").addEventListener("click", () => zoomBy(1.25));
    $("zoom-out").addEventListener("click", () => zoomBy(1 / 1.25));
    $("zoom-reset").addEventListener("click", fitToView);
    $("zoom-center").addEventListener("click", () => {
        const selection = state?.selection;
        if (!selection) return;
        if (!centerOnNode(selection.id)) setStatus(`${selection.id} is hidden by the current filters.`, true);
    });
    $("btn-reload").addEventListener("click", () => run(post("/api/reload").then(() => setStatus("Reloaded."))));

    const chips = $("generate-chips");
    for (const flag of QUICK_FLAGS) {
        const button = document.createElement("button");
        button.type = "button";
        button.className = "chip";
        button.textContent = flag;
        button.addEventListener("click", () => {
            const field = $("generate-args");
            field.value = `${field.value.trim()} ${flag}`.trim();
            field.focus();
        });
        chips.append(button);
    }

    $("btn-generate").addEventListener("click", () => {
        openGenerateDialog(state?.generateArgs ?? "");
    });

    wireGenerateCompletion();

    $("generate-select-all").addEventListener("click", () => {
        closeSuggestions();
        selectWholeField($("generate-args"));
    });

    $("generate-clear").addEventListener("click", () => {
        const field = $("generate-args");
        field.value = "";
        closeSuggestions();
        field.focus();
    });

    $("generate-ask-send").addEventListener("click", () => void askForArgs());
    $("generate-ask").addEventListener("keydown", (event) => {
        if (event.key !== "Enter") return;
        // The dialog's form would otherwise submit and close on Enter.
        event.preventDefault();
        void askForArgs();
    });

    $("dialog-generate").addEventListener("cancel", (event) => {
        // Escape dismisses the suggestion list before it dismisses the dialog.
        if (!completion.open) return;
        event.preventDefault();
        closeSuggestions();
    });

    $("generate-confirm").addEventListener("click", () => {
        const args = $("generate-args").value.trim();
        closeSuggestions();
        $("dialog-generate").close();
        setStatus("Running gh team-kit pr-graph…");
        run(post("/api/generate", { args }).then((result) => setStatus(`Saved ${result.path}`)));
    });
}

/** File extensions the export browser highlights for a format. */
function exportExtensions(kind) {
    return kind === "dot" ? ["dot", "gv"] : ["svg"];
}

/** Directory the export dialog starts in: next to the source file. */
function defaultExportDir() {
    const source = state?.source.path;
    if (!source) return "";
    const cut = Math.max(source.lastIndexOf("/"), source.lastIndexOf("\\"));
    return cut > 0 ? source.slice(0, cut) : "";
}

function defaultExportName(kind) {
    const source = state?.source.path ?? "";
    const cut = Math.max(source.lastIndexOf("/"), source.lastIndexOf("\\"));
    const file = cut >= 0 ? source.slice(cut + 1) : source;
    const base = file ? file.replace(/\.(dot|gv)$/i, "") : "pr-graph";
    return kind === "dot" ? `${base}.filtered.dot` : `${base}.svg`;
}

function connectEvents() {
    const source = new EventSource(withToken("/api/events"));
    source.addEventListener("message", () => void run(refresh()));
    source.addEventListener("error", () => setStatus("Lost connection to the extension; retrying…", true));
}

wireGraphInteractions();
wireSidebar();
wireResizer();
wireToolbar();
wireOpenDialog();
wireExportDialog();
wireInputSanitizer();
loadAskLog();
connectEvents();
window.addEventListener("resize", debounce(fitToView, 150));
void run(refresh());
