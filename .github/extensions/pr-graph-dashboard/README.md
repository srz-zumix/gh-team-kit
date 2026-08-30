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

- **Open… / Generate… / Reload / Clear graph / Export…** — pick a `.dot` file from a file browser, run
  `gh team-kit pr-graph` in a terminal (with completion for its flags, or with arguments written by
  the agent), re-read the current file, throw the current drawing away, or write the SVG / filtered
  DOT to a browsed destination.
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

Two things keep a huge graph from bogging the machine down. Anything past **2000 nodes or 6000 edges**
is not laid out automatically: the viewport shows the counts and a **Render anyway** button instead, so
narrowing the filters costs nothing and only a deliberate click starts a layout that needs minutes and
hundreds of megabytes. And when a filter, engine or direction change supersedes a layout that is still
running, its Graphviz process is killed rather than left to finish a result nobody will see. The same
happens when the extension itself stops or reloads, so a layout in flight cannot survive as an orphan
process burning a core in the background.

The layout is only half the cost: the drawing it produces puts one element per node and per edge into
the panel, so a graph of tens of thousands of edges leaves every other control crawling even after
Graphviz is done. **Clear graph** in the toolbar throws that drawing away — the file stays loaded and
the filters stay as they were, so the way back is one click on **Draw again**, or any filter change,
which redraws automatically.

**Render limit** caps how long Graphviz may spend on a single layout: `30s`, `1m` (default), `2m`, `5m`,
`10m`, `30m`, `1h` or `3h`. The badge counts up against it (`45s / 5m`), and a layout that runs out of
time reports which engine gave up. Changing the limit does not re-render what is already on screen.
Layout cost varies enormously by engine — on a 152-node / 1591-edge graph `sfdp`, `neato`, `fdp` and
`twopi` finish in about a second, `dot` takes ~36s, and `circo` needs ~210s — so raise the limit or
pick a faster engine when a render times out. The layout runs in a child process, so the panel stays
responsive on the long settings; a graph with tens of thousands of nodes is what the upper end is for.

Every engine other than `dot` packs nodes tightly enough that they overlap (`neato` produced 1406
overlapping node pairs on that graph, `twopi` 2519), so those layouts are emitted with
`overlap="prism" sep="+16" esep="+4"`, which removes the overlaps and is no slower. `dot` places nodes
in ranks itself and is left alone.

### Choosing files

**Open…** and **Export…** both browse the filesystem instead of asking for a typed path: the host
webview gives the canvas no usable native picker, since `<input type="file">` hides the absolute path
and cannot choose a save destination at all.

The browser lists directories first and then files, dims and hides names that do not match the
dialog's format, and shows shortcut chips for **Found** (every `.dot`/`.gv` file under the workspace
plus previously generated artifacts), **Workspace**, **Generated**, the directory of the file that is
open, and **Home**. Click a directory to enter it, `↑` or `Backspace` to leave it, `↑`/`↓` to move
through the list, `Enter` to enter a directory or accept a file, and double-click to open or save in
one step. **Show every file** reveals the non-matching and dot-prefixed entries. The path above the
list stays editable so a path can be pasted and jumped to with `Enter`; a listing that arrives while
it is being edited no longer overwrites what was typed.

**Open…** starts in the directory of the current file — with that file preselected — or in **Found**
when nothing is loaded yet. **Export…** starts next to the current file with a proposed name; the
destination shown under the name field is the browsed directory plus that name, and clicking an
existing file fills its name in to overwrite it.

### Exported colours

On screen the graph is coloured by `ui/styles.css`, which follows the app theme. Files written to disk
have no stylesheet, so `src/theme.mjs` reads the palette back out of the same `:root` block and bakes
literal colours into the DOT source as Graphviz attributes. Exported SVG and DOT therefore carry the
node-type and relation colours, and an exported SVG also gets an opaque white background — the canvas
renders on a transparent one so the app theme shows through, but a file needs its own or its dark
labels disappear in a dark viewer.

Adding or changing a colour means editing the `--nt-*`, `--rel-*` and `--graph-*` variables in
`ui/styles.css` only; every declaration there must end in a literal `#rrggbb` value, optionally wrapped
in a host theme token such as `var(--true-color-blue, #4c8eda)`, because that fallback is what the
exporter picks up.

### Generate

The **Generate…** dialog builds the `gh team-kit pr-graph` command line for you in two ways.

The argument field completes against the CLI's own shell completion (`gh team-kit __complete`), so the
flags and values it offers always match the installed version. Suggestions appear as you type;
`Ctrl`/`Cmd` + `Space` lists them on demand, `↑`/`↓` move through them, `Tab` or `Enter` accepts the
highlighted one, and `Esc` dismisses the list without closing the dialog. Values are completed too —
typing `--format ` offers `dot`, `json` and the rest.

The app intercepts `Cmd`+`A` before the canvas ever sees the key, so the arguments field is focused and
preselected when the dialog opens — typing replaces it — and **Select all** / **Clear** sit next to the
label for the same job. The host also delivers the arrow keys to the canvas as control characters
(`→` arrives as U+001D) on top of the real key events; those characters are discarded rather than
inserted, so they no longer show up as `□` in the field.

**Ask the agent for arguments** takes a plain-language description instead: "merged PRs from the last
three months, no bots". The dialog closes and the request goes to the agent with `pr-graph --help`
attached; when the agent answers via the `set_generate_args` action the dialog reopens with the
proposed arguments filled in. Nothing runs until you press **Run in terminal**, so the proposal is
always reviewable — ask the agent for `generate` instead if you want it to build the graph directly.

**Recent** lists the argument sets that were actually run, newest first, up to 25 of them. Click one to
put it back in the field. Re-running the same arguments moves the entry to the front instead of adding
a duplicate, and **Clear history** empties the list. The history lives with the other artifacts rather
than in a panel, so it survives reloads and is shared by every dashboard and every session; only
**Run in terminal** adds to it, so arguments you merely typed or that the agent merely proposed are not
recorded.

**Run in terminal** is the only button that starts a run. The extension has no way to open a host
canvas itself, so it reserves the destination file, sends the agent the exact command line and asks it
to run that in a Terminal canvas and call `load_dot` with the reserved path afterwards. The command is
visible and interruptible there, and the graph lands in the panel when it finishes. The reply arrives
in the chat, not in the dashboard.

There is deliberately no in-process Run button. `gh team-kit pr-graph` prints nothing until it
finishes, so an in-process run could only show a spinner, and it was capped by the extension's ten
minute timeout — which a wide date range with `--limit 0` exceeds. The `generate` action still runs
in-process for agents that ask for it explicitly.

### Text input and the IME

Every text field lives inside a `<form method="dialog">`, so an unhandled `Enter` submits the form and
closes the dialog. That made the `Enter` which commits a Japanese (or any other IME) conversion send
the request and dismiss the dialog mid-sentence.

The key that commits a conversion is not reported the same way everywhere: Chromium marks it
`isComposing` with `keyCode` 229, while WebKit — which is what the host webview runs on — ends the
composition first and then delivers a plain `Enter`. The dashboard therefore tracks
`compositionstart` / `compositionend` itself and ignores any `Enter` that arrives while a composition
is running or within 50 ms of one ending; that window is far shorter than the gap between two
deliberate key presses. The completion list steps aside entirely during a composition so the IME keeps
its own arrow keys and `Enter`. This applies to the argument field, the ask-the-agent field, the
browser location bar and the export file name.

## Agent-callable actions

Invoke with `invoke_canvas_action({ instanceId, actionName, input })`:

| Action              | Input                                                                                   | Purpose                                    |
| ------------------- | --------------------------------------------------------------------------------------- | ------------------------------------------ |
| `load_dot`          | `path` \| `dot`, `label`                                                                  | Load a DOT file or inline source.          |
| `generate`          | `args`                                                                                    | Run `gh team-kit pr-graph` and display it. |
| `set_generate_args` | `args`                                                                                    | Propose arguments in the Generate dialog.  |
| `get_state`         | —                                                                                         | Source, filters, layout, selection, counts.|
| `get_graph`         | `limit`, `includeNodes`, `includeEdges`, `useFilters`                                     | Graph data and the most connected nodes.   |
| `describe_node`     | `node`, `useFilters`                                                                      | One node with its edges.                   |
| `set_filters`       | `nodeTypes`, `relations`, `minWeight`, `search`, `focus`, `hops`, `keepOrphans`, `reset`   | Change what is displayed.                  |
| `set_view`          | `engine`, `rankdir`, `timeoutMs`                                                          | Change layout engine / direction / limit.  |
| `select_node`       | `node`                                                                                    | Select a node and move the view to it.     |
| `export`            | `path`, `kind`, `filtered`                                                                | Write SVG or DOT to disk.                  |

## Storage

The durable identity of a dashboard is the absolute path of the DOT file it renders. Graphs
produced by **Generate** are written to
`$COPILOT_HOME/extensions/pr-graph-dashboard/artifacts/generated/` so they survive reloads and
can be reopened later; `artifacts/instances.json` only records which file each open panel was
showing, so a provider restart restores the same view. `artifacts/generate-history.json` holds the
argument history, which is read from disk on every use so that panels open side by side agree.

## Layout

| File                | Responsibility                                      |
| ------------------- | --------------------------------------------------- |
| `extension.mjs`     | Canvas declaration, actions, open/close wiring.      |
| `src/browse.mjs`    | Filesystem listing behind the Open/Export browsers.  |
| `src/dot.mjs`       | DOT tokenizing, parsing, filtering, re-emission.     |
| `src/graphviz.mjs`  | Graphviz layout engines.                             |
| `src/prgraph.mjs`   | `gh team-kit pr-graph` invocation.                   |
| `src/dashboard.mjs` | Per-panel state, rendering pipeline, agent prompts.  |
| `src/server.mjs`    | Loopback HTTP server and Server-Sent Events.         |
| `src/store.mjs`     | Durable artifacts and instance pointers.             |
| `src/theme.mjs`     | Graph palette shared with the exported files.        |
| `ui/`               | Dashboard front-end.                                 |
