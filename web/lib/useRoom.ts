"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { wsURL } from "./api";
import {
  CloseCodes,
  type ClientMsgType,
  type GameError,
  type PoolView,
  type ServerMsg,
  type StateView,
} from "./protocol";

export type Send = (type: ClientMsgType, data?: unknown) => Promise<ServerMsg | null>;

export type ConnStatus ="connecting" | "open" | "reconnecting" | "closed";

export interface RoomConn {
  status: ConnStatus;
  /** Why the connection ended for good (room full, not found, …). */
  closedReason: string | null;
  state: StateView | null;
  pool: PoolView | null;
  /** Milliseconds to add to Date.now() to get server time. */
  clockSkew: number;
  errors: { id: number; error: GameError }[];
  /** Latest answer or guess result for this player (changes identity on each). */
  lastResult: { seq: number; msg: ServerMsg } | null;
  dismissError: (id: number) => void;
  send: (type: ClientMsgType, data?: unknown) => Promise<ServerMsg | null>;
}

const FATAL: Record<number, string> = {
  [CloseCodes.roomFull]: "This room already has two players.",
  [CloseCodes.roomNotFound]: "This room doesn't exist or has closed.",
  [CloseCodes.unauthorized]: "Your guest session expired. Go home to start a new one.",
  [CloseCodes.replaced]: "This room is open in another tab or window.",
};

/**
 * Connects to a room and keeps the latest snapshot. The server sends a full
 * per-player state after every change, so the client never patches state:
 * it just renders the newest snapshot.
 */
export function useRoom(code: string, token: string | undefined): RoomConn {
  const [status, setStatus] = useState<ConnStatus>("connecting");
  const [closedReason, setClosedReason] = useState<string | null>(null);
  const [state, setState] = useState<StateView | null>(null);
  const [pool, setPool] = useState<PoolView | null>(null);
  const [clockSkew, setClockSkew] = useState(0);
  const [errors, setErrors] = useState<{ id: number; error: GameError }[]>([]);
  const [lastResult, setLastResult] = useState<{ seq: number; msg: ServerMsg } | null>(null);

  const wsRef = useRef<WebSocket | null>(null);
  const cseqRef = useRef(0);
  const lastSeqRef = useRef(0);
  const pending = useRef(new Map<number, (m: ServerMsg | null) => void>());
  const errId = useRef(0);

  const pushError = useCallback((error: GameError) => {
    const id = ++errId.current;
    setErrors((e) => [...e.slice(-2), { id, error }]);
    setTimeout(() => setErrors((e) => e.filter((x) => x.id !== id)), 6000);
  }, []);

  useEffect(() => {
    if (!token || !code) return;
    let stopped = false;
    let attempt = 0;
    let timer: ReturnType<typeof setTimeout> | undefined;

    const connect = () => {
      const ws = new WebSocket(wsURL(code, token));
      wsRef.current = ws;
      cseqRef.current = 0; // cseq is per connection
      lastSeqRef.current = 0;

      ws.onopen = () => {
        attempt = 0;
        setStatus("open");
      };
      ws.onmessage = (ev) => {
        let msg: ServerMsg;
        try {
          msg = JSON.parse(ev.data as string);
        } catch {
          return;
        }
        if (msg.seq <= lastSeqRef.current) return;
        lastSeqRef.current = msg.seq;
        switch (msg.type) {
          case "state": {
            const s = msg.data as StateView;
            setState(s);
            setClockSkew(Date.parse(s.server_time) - Date.now());
            break;
          }
          case "pool":
            setPool(msg.data as PoolView);
            break;
          case "error":
            pushError(msg.data as GameError);
            break;
          case "answer":
          case "guess_result":
            setLastResult({ seq: msg.seq, msg });
            break;
          case "kicked":
            setClosedReason(FATAL[CloseCodes.replaced] ?? null);
            break;
        }
        if (msg.reply_to) {
          const resolve = pending.current.get(msg.reply_to);
          if (resolve) {
            pending.current.delete(msg.reply_to);
            resolve(msg);
          }
        }
      };
      ws.onclose = (ev) => {
        if (wsRef.current === ws) wsRef.current = null;
        pending.current.forEach((r) => r(null));
        pending.current.clear();
        if (stopped) return;
        const fatal = FATAL[ev.code];
        if (fatal) {
          setClosedReason(fatal);
          setStatus("closed");
          return;
        }
        setStatus("reconnecting");
        const delay = Math.min(5000, 500 * 2 ** attempt++);
        timer = setTimeout(connect, delay);
      };
    };

    setStatus("connecting");
    setClosedReason(null);
    connect();
    return () => {
      stopped = true;
      clearTimeout(timer);
      wsRef.current?.close(1000, "leaving");
      wsRef.current = null;
    };
  }, [code, token, pushError]);

  const send = useCallback((type: ClientMsgType, data?: unknown) => {
    const ws = wsRef.current;
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      pushError({ code: "offline", message: "Not connected. Reconnecting…" });
      return Promise.resolve(null);
    }
    const cseq = ++cseqRef.current;
    ws.send(JSON.stringify({ type, cseq, data: data ?? {} }));
    return new Promise<ServerMsg | null>((resolve) => {
      pending.current.set(cseq, resolve);
      // Most messages have no direct reply (only a new state); release the
      // promise after a short wait so callers never hang.
      setTimeout(() => {
        if (pending.current.delete(cseq)) resolve(null);
      }, 4000);
    });
  }, [pushError]);

  const dismissError = useCallback((id: number) => setErrors((e) => e.filter((x) => x.id !== id)), []);

  return { status, closedReason, state, pool, clockSkew, errors, dismissError, send, lastResult };
}

/** Re-renders every `ms` so countdowns stay current. */
export function useNow(ms = 1000): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const t = setInterval(() => setNow(Date.now()), ms);
    return () => clearInterval(t);
  }, [ms]);
  return now;
}
