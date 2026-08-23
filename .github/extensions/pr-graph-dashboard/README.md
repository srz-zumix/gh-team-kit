# pr-graph-dashboard

A Copilot CLI canvas extension that renders `gh team-kit pr-graph --format dot` output as an
interactive graph in the app's side panel, and lets the dashboard hand work back to the agent.

## Requirements

- [Graphviz](https://graphviz.org/) on `PATH` (`brew install graphviz`) — used for layout.
- The `gh` CLI with the `team-kit` extension installed — used by the **Generate** button.

## Usage

Ask the agent to open the canvas, optionally naming a source:

```text
Open the PR graph canvas for docs/pr-graph.dot
Open the PR graph canvas and generate one with --limit 100 --state merged --no-bots
```

The agent opens it via `open_canvas` with `canvasId: "pr-graph"` and an optional input:

| Field   | Type     | Description                                                                   |
| ------- | -------- | ----------------------------------------------------------------------------- |
| `path`  | `string` | DOT file to load; relative paths resolve against the workspace.                |
| `dot`   | `string` | Inline DOT source to render instead of a file.                                 |
| `label` | `string` | Display label used together with `dot`.                                        |
| `args`  | `string` | Arguments for `gh team-kit pr-graph`; the graph is generated when this is set. |

## Dashboard

- **Open… / Generate… / Reload / Export…** — load a `.dot` file, run `gh team-kit pr-graph`,
  re-read the current file, or write the SVG / filtered DOT to disk.
- **Filters** — node types, edge relations, minimum edge weight, name search, neighbourhood
  radius, orphan handling, Graphviz layout engine and direction, and the render limit.
- **Nodes** — ranked node list; click to select and move the view to that node, double-click to focus
  the graph on it.
- **Details** — the selected node's incoming and outgoing relationships.
- **Agent** — preset prompts (summarize, analyze selection, review load, ownership risk, next
  query) and a free-text box. Messages are sent to the agent in the current session together
  with the graph context.

Click a node in the graph to select it, which highlights its neighbourhood and opens the Details tab;
double-click to focus on it. Selecting a node anywhere else — the Nodes list, a peer in Details, or the
`select_node` action — pans the graph to it; the ⌖ button repeats that move for the current selection.
Focusing re-lays the graph out, so the view travels to the focused node once the new layout is on
screen, and clearing the focus returns to the spot the view came from. Scroll to pan, and zoom either
with `Ctrl`/`Cmd` + scroll (or a trackpad
pinch) or by dragging up and down. Drag the divider between the graph and the side panel to resize
them, double-click it to reset, or focus it and use the arrow keys. The chosen size is remembered.

Large graphs take a while to lay out. While Graphviz is still working, a **Rendering…** badge with an
elapsed-seconds counter appears over the top-left of the viewport, the stale layout on screen is dimmed,
and the status bar appends `· rendering…` to the counts. The badge only appears once a render has taken
longer than 200 ms, so quick redraws do not flash it.

**Render limit** caps how long Graphviz may spend on a single layout: `30s`, `1m` (default), `2m`, `5m`
or `10m`. The badge counts up against it (`45s / 5m`), and a layout that runs out of time reports which
engine gave up. Changing the limit does not re-render what is already on screen. Layout cost varies
enormously by engine — on a 152-node / 1591-edge graph `sfdp`, `neato`, `fdp` and `twopi` finish in
about a second, `dot` takes ~36s, and `circo` needs ~210s — so raise the limit or pick a faster engine
when a render times out.

## Agent-callable actions

Invoke with `invoke_canvas_action({ instanceId, actionName, input })`:

| Action          | Input                                                                                   | Purpose                                    |
| --------------- | --------------------------------------------------------------------------------------- | ------------------------------------------ |
| `load_dot`      | `path` \| `dot`, `label`                                                                  | Load a DOT file or inline source.          |
| `generate`      | `args`                                                                                    | Run `gh team-kit pr-graph` and display it. |
| `get_state`     | —                                                                                         | Source, filters, layout, selection, counts.|
| `get_graph`     | `limit`, `includeNodes`, `includeEdges`, `useFilters`                                     | Graph data and the most connected nodes.   |
| `describe_node` | `node`, `useFilters`                                                                      | One node with its edges.                   |
| `set_filters`   | `nodeTypes`, `relations`, `minWeight`, `search`, `focus`, `hops`, `keepOrphans`, `reset`   | Change what is displayed.                  |
| `set_view`      | `engine`, `rankdir`, `timeoutMs`                                                          | Change layout engine / direction / limit.  |
| `select_node`   | `node`                                                                                    | Select a node and move the view to it.     |
| `export`        | `path`, `kind`, `filtered`                                                                | Write SVG or DOT to disk.                  |

## Storage

The durable identity of a dashboard is the absolute path of the DOT file it renders. Graphs
produced by **Generate** are written to
`$COPILOT_HOME/extensions/pr-graph-dashboard/artifacts/generated/` so they survive reloads and
can be reopened later; `artifacts/instances.json` only records which file each open panel was
showing, so a provider restart restores the same view.

## Layout

| File               | Responsibility                                       |
| ------------------ | ---------------------------------------------------- |
| `extension.mjs`    | Canvas declaration, actions, open/close wiring.      |
| `src/dot.mjs`      | DOT tokenizing, parsing, filtering, re-emission.     |
| `src/graphviz.mjs` | Graphviz layout engines.                             |
| `src/prgraph.mjs`  | `gh team-kit pr-graph` invocation.                   |
| `src/dashboard.mjs`| Per-panel state, rendering pipeline, agent prompts.  |
| `src/server.mjs`   | Loopback HTTP server and Server-Sent Events.         |
| `src/store.mjs`    | Durable artifacts and instance pointers.             |
| `ui/`              | Dashboard front-end.                                 |
