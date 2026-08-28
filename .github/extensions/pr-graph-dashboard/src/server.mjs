// Loopback HTTP server backing one canvas instance.
//
// The host iframe has no privileged bridge, so the UI talks to the extension
// over plain HTTP on 127.0.0.1 and receives push updates via Server-Sent
// Events.

import { createServer } from "node:http";
import { readFile } from "node:fs/promises";
import { randomUUID } from "node:crypto";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { NODE_TYPES, RELATIONS, topNodes } from "./dot.mjs";
import { checkGraphviz } from "./graphviz.mjs";
import { completeArgs } from "./complete.mjs";
import { browse } from "./browse.mjs";
import { PROMPT_PRESETS } from "./dashboard.mjs";

const UI_DIR = path.join(path.dirname(fileURLToPath(import.meta.url)), "..", "ui");

const MIME = {
    ".html": "text/html; charset=utf-8",
    ".js": "text/javascript; charset=utf-8",
    ".css": "text/css; charset=utf-8",
    ".svg": "image/svg+xml; charset=utf-8",
};

function sendJson(res, status, body) {
    const payload = JSON.stringify(body);
    res.writeHead(status, { "Content-Type": "application/json; charset=utf-8", "Cache-Control": "no-store" });
    res.end(payload);
}

async function readBody(req, limit = 2_000_000) {
    const chunks = [];
    let size = 0;
    for await (const chunk of req) {
        size += chunk.length;
        if (size > limit) throw new Error("request body too large");
        chunks.push(chunk);
    }
    if (chunks.length === 0) return {};
    const text = Buffer.concat(chunks).toString("utf-8");
    if (!text.trim()) return {};
    return JSON.parse(text);
}

// Same-origin only: scripts/styles/fetch/SSE come from this loopback server,
// the favicon is an empty data: URL, and inline scripts/objects/frames are
// forbidden. This blocks script execution even if SVG sanitization ever regresses.
const CONTENT_SECURITY_POLICY = [
    "default-src 'none'",
    "script-src 'self'",
    "style-src 'self' 'unsafe-inline'",
    "img-src 'self' data:",
    "connect-src 'self'",
    "font-src 'self'",
    "base-uri 'none'",
    "form-action 'none'",
].join("; ");

async function serveStatic(res, name) {
    const file = path.join(UI_DIR, name);
    if (!file.startsWith(UI_DIR)) {
        res.writeHead(403).end("forbidden");
        return;
    }
    try {
        const contents = await readFile(file);
        res.writeHead(200, {
            "Content-Type": MIME[path.extname(file)] ?? "application/octet-stream",
            "Cache-Control": "no-store",
            "Referrer-Policy": "no-referrer",
            "Content-Security-Policy": CONTENT_SECURITY_POLICY,
        });
        res.end(contents);
    } catch {
        res.writeHead(404).end("not found");
    }
}

/**
 * Starts a loopback server for a dashboard instance.
 *
 * @param {import("./dashboard.mjs").Dashboard} dashboard
 * @returns {Promise<{server: import("node:http").Server, url: string}>}
 */
export async function startServer(dashboard) {
    // Unguessable per-instance token. Every API/SSE request must carry it, so a
    // local process or web page that merely scans the loopback port cannot read
    // files or trigger writes/generation against this panel.
    const token = randomUUID();
    const clients = new Set();
    const keepAlives = new WeakMap();
    // Idempotent teardown: stop the keepalive, forget the client and destroy the
    // socket. Safe to call repeatedly (close/error can both fire).
    const dropClient = (client) => {
        const timer = keepAlives.get(client);
        if (timer) {
            clearInterval(timer);
            keepAlives.delete(client);
        }
        clients.delete(client);
        client.destroy();
    };
    // A single guarded writer for every SSE frame. A write after the peer went
    // away can throw or the stream can already be ended; either way we drop the
    // client instead of letting the error crash the server process.
    const send = (client, chunk) => {
        if (client.writableEnded || client.destroyed) {
            dropClient(client);
            return;
        }
        try {
            client.write(chunk);
        } catch {
            dropClient(client);
        }
    };
    const broadcast = () => {
        const payload = `data: ${JSON.stringify({ stateRev: dashboard.stateRev, renderRev: dashboard.renderRev })}\n\n`;
        for (const client of [...clients]) send(client, payload);
    };
    dashboard.on("changed", broadcast);

    const routes = {
        "GET /api/state": async (_req, res) => {
            const graphviz = await checkGraphviz();
            sendJson(res, 200, {
                ...dashboard.snapshot(),
                nodeTypes: NODE_TYPES,
                relations: RELATIONS,
                presets: PROMPT_PRESETS.map(({ id, label, needsSelection }) => ({ id, label, needsSelection })),
                graphviz,
            });
        },
        "GET /api/svg": async (_req, res) => {
            if (!dashboard.svg) {
                res.writeHead(404, { "Content-Type": "text/plain; charset=utf-8" });
                res.end(dashboard.renderError || "no rendered graph");
                return;
            }
            res.writeHead(200, { "Content-Type": "image/svg+xml; charset=utf-8", "Cache-Control": "no-store" });
            res.end(dashboard.svg);
        },
        "GET /api/browse": async (_req, res, url) => {
            const extensions = (url.searchParams.get("ext") ?? "")
                .split(",")
                .map((value) => value.trim().toLowerCase())
                .filter(Boolean);
            sendJson(
                res,
                200,
                await browse({
                    dir: url.searchParams.get("dir"),
                    workspacePath: dashboard.workspacePath,
                    sourcePath: dashboard.sourcePath,
                    extensions,
                }),
            );
        },
        "GET /api/nodes": async (req, res, url) => {
            const limit = Math.min(500, Math.max(1, Number(url.searchParams.get("limit")) || 50));
            const nodeType = url.searchParams.get("type") || undefined;
            const query = (url.searchParams.get("q") || "").toLowerCase();
            const graph = dashboard.filteredGraph();
            const ranked = topNodes(graph, { limit: 5000, nodeType });
            const matched = query
                ? ranked.filter((node) => node.name.toLowerCase().includes(query) || node.id.toLowerCase().includes(query))
                : ranked;
            sendJson(res, 200, { nodes: matched.slice(0, limit), total: matched.length });
        },
        "GET /api/events": async (_req, res) => {
            res.writeHead(200, {
                "Content-Type": "text/event-stream; charset=utf-8",
                "Cache-Control": "no-cache",
                Connection: "keep-alive",
            });
            clients.add(res);
            res.on("close", () => dropClient(res));
            res.on("error", () => dropClient(res));
            keepAlives.set(res, setInterval(() => send(res, ": ping\n\n"), 20_000));
            send(res, "retry: 2000\n\n");
        },
        "POST /api/load": async (req, res) => {
            const body = await readBody(req);
            if (body.dot) {
                dashboard.setDot(String(body.dot), { sourceLabel: body.label ? String(body.label) : "inline DOT" });
                sendJson(res, 200, { ok: true, path: null });
                return;
            }
            const file = await dashboard.loadFile(body.path);
            sendJson(res, 200, { ok: true, path: file });
        },
        "POST /api/reload": async (_req, res) => {
            if (!dashboard.sourcePath) throw new Error("no file is currently loaded");
            const file = await dashboard.loadFile(dashboard.sourcePath);
            sendJson(res, 200, { ok: true, path: file });
        },
        "POST /api/generate": async (req, res) => {
            const body = await readBody(req);
            const result = await dashboard.generate(body.args ?? "");
            sendJson(res, 200, { ok: true, ...result });
        },
        "POST /api/filters": async (req, res) => {
            const body = await readBody(req);
            const filters = body.reset ? dashboard.resetFilters() : dashboard.setFilters(body);
            sendJson(res, 200, { ok: true, filters });
        },
        "POST /api/view": async (req, res) => {
            const body = await readBody(req);
            sendJson(res, 200, { ok: true, view: dashboard.setView(body) });
        },
        "POST /api/select": async (req, res) => {
            const body = await readBody(req);
            sendJson(res, 200, { ok: true, selection: dashboard.select(body.nodeId ?? null) });
        },
        "POST /api/ask": async (req, res) => {
            const body = await readBody(req);
            let prompt = String(body.prompt ?? "");
            if (body.presetId) {
                const preset = PROMPT_PRESETS.find((candidate) => candidate.id === body.presetId);
                if (!preset) throw new Error(`unknown preset "${body.presetId}"`);
                prompt = [preset.prompt, prompt].filter(Boolean).join("\n\n");
            }
            sendJson(res, 200, { ok: true, ...(await dashboard.ask(prompt)) });
        },
        "GET /api/complete": async (_req, res, url) => {
            const line = url.searchParams.get("line") ?? "";
            sendJson(res, 200, await completeArgs(line, { cwd: dashboard.workspacePath }));
        },
        "POST /api/generate/ask": async (req, res) => {
            const body = await readBody(req);
            sendJson(res, 200, { ok: true, ...(await dashboard.askForArgs(body.prompt)) });
        },
        "POST /api/export": async (req, res) => {
            const body = await readBody(req);
            const file =
                body.kind === "dot"
                    ? await dashboard.exportDot(body.path, { filtered: body.filtered !== false })
                    : await dashboard.exportSvg(body.path);
            sendJson(res, 200, { ok: true, path: file });
        },
    };

    const isLoopbackHost = (host) => {
        if (!host) return false;
        const name = host.replace(/:\d+$/, "");
        return name === "127.0.0.1" || name === "localhost" || name === "[::1]";
    };

    const server = createServer((req, res) => {
        const url = new URL(req.url ?? "/", "http://127.0.0.1");
        // Reject foreign Host headers (blunts DNS-rebinding) and require the
        // per-instance token on every API/SSE route. Static UI assets are
        // harmless and load before the token is known, so they stay open.
        if (!isLoopbackHost(req.headers.host)) {
            res.writeHead(403).end("forbidden");
            return;
        }
        if (url.pathname.startsWith("/api/") && url.searchParams.get("t") !== token) {
            res.writeHead(403).end("forbidden");
            return;
        }
        const key = `${req.method} ${url.pathname}`;
        const handler = routes[key];
        if (handler) {
            Promise.resolve(handler(req, res, url)).catch((error) => {
                if (res.headersSent) {
                    res.end();
                    return;
                }
                sendJson(res, 400, { ok: false, error: error instanceof Error ? error.message : String(error) });
            });
            return;
        }
        if (req.method !== "GET") {
            res.writeHead(405).end("method not allowed");
            return;
        }
        const name = url.pathname === "/" ? "index.html" : path.basename(url.pathname);
        void serveStatic(res, name);
    });

    await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
    const address = server.address();
    const port = typeof address === "object" && address ? address.port : 0;

    const close = async () => {
        dashboard.off("changed", broadcast);
        for (const client of [...clients]) dropClient(client);
        clients.clear();
        await new Promise((resolve) => server.close(() => resolve()));
    };

    return { server, url: `http://127.0.0.1:${port}/#t=${token}`, close };
}
