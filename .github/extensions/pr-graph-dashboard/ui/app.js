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
let nodeIds = [];
let checksSignature = "";
let statusTimer = null;

const view = { scale: 1, x: 0, y: 0, size: { width: 0, height: 0 } };
const MIN_SCALE = 0.01;
const MAX_SCALE = 8;

async function api(path, options = {}) {
    const response = await fetch(path, {
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

async function refresh() {
    state = await api("/api/state");
    renderState();
    if (state.renderRev !== renderedRev) await loadSvg();
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
    const response = await fetch("/api/svg");
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

    renderStatusBar();
    renderChecks();
    renderInputs();
    renderFocus();
    void run(renderNodeList());
    renderDetails();
    renderPresets();
    applyHighlight();
}

function renderStatusBar() {
    const { total, visible } = state.stats;
    $("status-counts").textContent = state.source.loaded
        ? `${visible.nodes}/${total.nodes} nodes · ${visible.edges}/${total.edges} edges`
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
    askButton.textContent = "Ask agent";
    askButton.addEventListener("click", () => run(post("/api/ask", { presetId: "selection" }).then(onAsked)));
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
        button.disabled = preset.needsSelection && !state.selection;
        button.addEventListener("click", () =>
            run(post("/api/ask", { presetId: preset.id, prompt: $("agent-prompt").value.trim() }).then(onAsked)),
        );
        container.append(button);
    }
}

function onAsked() {
    $("agent-prompt").value = "";
    $("agent-status").textContent = "Sent to the agent. Check the chat for its reply.";
    setStatus("Message sent to the agent.");
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
        run(post("/api/select", { nodeId: id }));
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

    $("nodes-query").addEventListener("input", debounce(() => run(renderNodeList()), 200));
    $("node-list").addEventListener("click", (event) => {
        const item = event.target.closest("li[data-node-id]");
        if (item) run(post("/api/select", { nodeId: item.dataset.nodeId }));
    });
    $("node-list").addEventListener("dblclick", (event) => {
        const item = event.target.closest("li[data-node-id]");
        if (item) run(post("/api/filters", { focus: item.dataset.nodeId }));
    });
    $("details").addEventListener("click", (event) => {
        const peer = event.target.closest(".peer[data-node-id]");
        if (peer) run(post("/api/select", { nodeId: peer.dataset.nodeId }));
    });

    $("agent-send").addEventListener("click", () => {
        const prompt = $("agent-prompt").value.trim();
        if (!prompt) {
            setStatus("Enter a message first.", true);
            return;
        }
        run(post("/api/ask", { prompt }).then(onAsked));
    });
}

function wireToolbar() {
    $("zoom-in").addEventListener("click", () => zoomBy(1.25));
    $("zoom-out").addEventListener("click", () => zoomBy(1 / 1.25));
    $("zoom-reset").addEventListener("click", fitToView);
    $("btn-reload").addEventListener("click", () => run(post("/api/reload").then(() => setStatus("Reloaded."))));

    $("btn-open").addEventListener("click", async () => {
        const dialog = $("dialog-open");
        $("open-path").value = state?.source.path ?? "";
        const list = $("open-file-list");
        list.replaceChildren();
        dialog.showModal();
        try {
            const { files } = await api("/api/files");
            if (files.length === 0) {
                const item = document.createElement("li");
                item.className = "muted";
                item.textContent = "No .dot files found in the workspace or artifacts.";
                list.append(item);
            }
            for (const file of files) {
                const item = document.createElement("li");
                const name = document.createElement("span");
                name.className = "name";
                name.textContent = file.path;
                name.title = file.path;
                const origin = document.createElement("span");
                origin.className = "origin";
                origin.textContent = file.origin;
                item.append(name, origin);
                item.addEventListener("click", () => {
                    $("open-path").value = file.path;
                });
                item.addEventListener("dblclick", () => {
                    dialog.close();
                    run(post("/api/load", { path: file.path }));
                });
                list.append(item);
            }
        } catch (error) {
            setStatus(error.message, true);
        }
    });

    $("open-confirm").addEventListener("click", () => {
        const path = $("open-path").value.trim();
        if (!path) {
            setStatus("Enter a path.", true);
            return;
        }
        $("dialog-open").close();
        run(post("/api/load", { path }));
    });

    $("open-path").addEventListener("keydown", (event) => {
        if (event.key !== "Enter") return;
        event.preventDefault();
        $("open-confirm").click();
    });

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
        $("generate-args").value = state?.generateArgs ?? "";
        $("dialog-generate").showModal();
    });

    $("generate-confirm").addEventListener("click", () => {
        const args = $("generate-args").value.trim();
        $("dialog-generate").close();
        setStatus("Running gh team-kit pr-graph…");
        run(post("/api/generate", { args }).then((result) => setStatus(`Saved ${result.path}`)));
    });

    $("btn-export").addEventListener("click", () => {
        const kind = $("export-kind").value;
        $("export-path").value = defaultExportPath(kind);
        $("dialog-export").showModal();
    });

    $("export-kind").addEventListener("change", (event) => {
        $("export-path").value = defaultExportPath(event.target.value);
    });

    $("export-confirm").addEventListener("click", () => {
        const path = $("export-path").value.trim();
        if (!path) {
            setStatus("Enter a path.", true);
            return;
        }
        $("dialog-export").close();
        run(post("/api/export", { kind: $("export-kind").value, path }).then((result) => setStatus(`Saved ${result.path}`)));
    });
}

function defaultExportPath(kind) {
    const source = state?.source.path;
    const base = source ? source.replace(/\.(dot|gv)$/i, "") : "pr-graph";
    return kind === "dot" ? `${base}.filtered.dot` : `${base}.svg`;
}

function connectEvents() {
    const source = new EventSource("/api/events");
    source.addEventListener("message", () => void run(refresh()));
    source.addEventListener("error", () => setStatus("Lost connection to the extension; retrying…", true));
}

wireGraphInteractions();
wireSidebar();
wireResizer();
wireToolbar();
connectEvents();
window.addEventListener("resize", debounce(fitToView, 150));
void run(refresh());
