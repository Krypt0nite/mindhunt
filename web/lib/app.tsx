"use client";

import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react";
import { createGuest, publicConfig, renameGuest, storedGuest, verifyGuest } from "./api";
import type { Guest, PublicConfig } from "./protocol";

interface AppCtx {
  appName: string;
  config: PublicConfig | null;
  guest: Guest | null;
  ready: boolean;
  serverError: string | null;
  signIn: (name: string) => Promise<Guest>;
  rename: (name: string) => Promise<void>;
}

const Ctx = createContext<AppCtx | null>(null);

const FALLBACK_NAME = process.env.NEXT_PUBLIC_APP_NAME ?? "MindHunt";

export function AppProvider({ children }: { children: ReactNode }) {
  const [config, setConfig] = useState<PublicConfig | null>(null);
  const [guest, setGuest] = useState<Guest | null>(null);
  const [ready, setReady] = useState(false);
  const [serverError, setServerError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [cfg, g] = await Promise.all([
          publicConfig(),
          (async () => {
            const s = storedGuest();
            return s ? verifyGuest(s) : null;
          })(),
        ]);
        if (cancelled) return;
        setConfig(cfg);
        setGuest(g);
      } catch (e) {
        if (!cancelled) setServerError(e instanceof Error ? e.message : "Can't reach the game server.");
      } finally {
        if (!cancelled) setReady(true);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const appName = config?.app_name ?? FALLBACK_NAME;
  useEffect(() => {
    document.title = appName;
  }, [appName]);

  const signIn = useCallback(async (name: string) => {
    const g = await createGuest(name);
    setGuest(g);
    return g;
  }, []);

  const rename = useCallback(
    async (name: string) => {
      if (!guest) return;
      setGuest(await renameGuest(guest, name));
    },
    [guest],
  );

  return (
    <Ctx.Provider value={{ appName, config, guest, ready, serverError, signIn, rename }}>{children}</Ctx.Provider>
  );
}

export function useApp(): AppCtx {
  const c = useContext(Ctx);
  if (!c) throw new Error("useApp outside AppProvider");
  return c;
}
