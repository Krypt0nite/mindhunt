// MindHunt on Cloudflare (free plan).
//
// Static files (the exported Next.js client and /assets/ images) are served
// by Workers Static Assets. /api/* and /ws reach this Worker, which forwards
// them to a single Durable Object, "hub". The hub runs the Go game server
// compiled to WebAssembly (server/cmd/mindhunt-wasm): rooms live in its
// memory, guests, daily plays and player-made packs in its SQLite storage.
//
// One hub keeps every room in one place, as the Go server expects, and a
// single always-on object can never exceed the free plan's daily duration.

import { DurableObject } from "cloudflare:workers";
import "../build/wasm_exec.js";
import wasmModule from "../build/mindhunt.wasm";
import shippedAssets from "../build/assets.json";

export default {
  async fetch(request, env) {
    const url = new URL(request.url);
    if (url.pathname.startsWith("/api/") || url.pathname === "/ws") {
      return env.HUB.get(env.HUB.idFromName("hub")).fetch(request);
    }
    return env.ASSETS.fetch(request);
  },
};

const PING = JSON.stringify({ type: "ping", seq: 0 }); // seq 0: the client ignores it
const NULL_BODY = new Set([101, 204, 205, 304]);

function jsonError(status, code, message) {
  return Response.json({ code, message }, { status, headers: { "Cache-Control": "no-store" } });
}

export class Hub extends DurableObject {
  constructor(ctx, env) {
    super(ctx, env);
    this.sockets = new Map(); // id -> server-side WebSocket
    this.nextId = 1;
    this.api = null;
  }

  // start boots the Go server once per object instance.
  async start() {
    if (this.api) return this.api;
    const sql = this.ctx.storage.sql;
    globalThis.mindhuntHost = {
      sql: (query, argsJSON) => {
        try {
          const cur = sql.exec(query, ...JSON.parse(argsJSON));
          const rows = cur.raw().toArray();
          return JSON.stringify({ rows, written: cur.rowsWritten });
        } catch (e) {
          return JSON.stringify({ error: String((e && e.message) || e) });
        }
      },
      send: (id, text) => {
        try {
          this.sockets.get(id)?.send(text);
        } catch {}
      },
      close: (id, code, reason) => {
        try {
          this.sockets.get(id)?.close(code, reason);
        } catch {}
      },
    };

    const go = new globalThis.Go();
    const env = {};
    for (const [k, v] of Object.entries(this.env)) if (typeof v === "string") env[k] = v;
    env.MINDHUNT_ASSETS = shippedAssets.join(",");
    go.env = env;
    const instance = await WebAssembly.instantiate(wasmModule, go.importObject);
    globalThis.mindhunt = undefined;
    go.run(instance); // runs main() until it blocks; the promise settles only on exit
    if (!globalThis.mindhunt) throw new Error(globalThis.mindhuntError || "game server did not start");
    this.api = globalThis.mindhunt;
    return this.api;
  }

  async fetch(request) {
    let api;
    try {
      api = await this.start();
    } catch (e) {
      console.error("boot failed", e);
      return jsonError(503, "unavailable", "game server is starting; try again");
    }
    const url = new URL(request.url);
    if (url.pathname === "/ws") return this.socket(request, url, api);

    const headers = {};
    for (const [k, v] of request.headers) headers[k] = v;
    let body = "";
    if (request.method !== "GET" && request.method !== "HEAD") {
      body = await request.text();
      if (body.length > 300_000) return jsonError(413, "too_large", "request too large");
    }
    const ip = request.headers.get("cf-connecting-ip") || "0.0.0.0";
    const res = api.http(request.method, request.url, JSON.stringify(headers), body, ip);
    const out = new Headers();
    for (const [k, vs] of Object.entries(JSON.parse(res.headers))) for (const v of vs) out.append(k, v);
    return new Response(NULL_BODY.has(res.status) ? null : res.body, { status: res.status, headers: out });
  }

  originAllowed(origin, url) {
    let host;
    try {
      host = new URL(origin).host;
    } catch {
      return false;
    }
    if (host === url.host) return true;
    const extra = (this.env.ALLOWED_ORIGINS || "").split(",").map((s) => s.trim()).filter(Boolean);
    return extra.includes(host);
  }

  socket(request, url, api) {
    if (request.headers.get("Upgrade")?.toLowerCase() !== "websocket") {
      return jsonError(426, "upgrade_required", "expected a WebSocket");
    }
    const origin = request.headers.get("Origin");
    if (origin && !this.originAllowed(origin, url)) return jsonError(403, "forbidden", "origin not allowed");

    const [client, server] = Object.values(new WebSocketPair());
    server.accept();
    const id = this.nextId++;
    this.sockets.set(id, server);

    const refused = api.wsOpen(id, url.searchParams.get("token") || "", url.searchParams.get("room") || "");
    if (refused) {
      this.sockets.delete(id);
      server.close(refused.code, refused.reason);
      return new Response(null, { status: 101, webSocket: client });
    }

    const ping = setInterval(() => {
      try {
        server.send(PING);
      } catch {}
    }, 30_000);
    const done = () => {
      clearInterval(ping);
      if (this.sockets.delete(id)) api.wsClose(id);
    };
    server.addEventListener("message", (e) => {
      if (typeof e.data === "string") api.wsMessage(id, e.data);
    });
    server.addEventListener("close", () => {
      done();
      try {
        server.close(1000, "bye");
      } catch {}
    });
    server.addEventListener("error", done);
    return new Response(null, { status: 101, webSocket: client });
  }
}
