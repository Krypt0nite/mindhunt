"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { ApiError, getPack, listPacks, myPacks } from "@/lib/api";
import { useApp } from "@/lib/app";
import { PackPicker } from "./PackPicker";
import type { ClientMsgType, Config, MyPack, PackSummary, PlayerView, ServerMsg, StateView } from "@/lib/protocol";

type Send = (type: ClientMsgType, data?: unknown) => Promise<ServerMsg | null>;

const DIFFICULTIES: { id: Config["difficulty"]; label: string }[] = [
  { id: "easy", label: "Easy" },
  { id: "normal", label: "Normal" },
  { id: "hard", label: "Hard" },
];
const POOL: Record<string, number> = { easy: 24, normal: 48 };
const TIMERS = [0, 30, 60, 90, 120];

export function Lobby({ state, send }: { state: StateView; send: Send }) {
  const router = useRouter();
  const { guest } = useApp();
  const [packs, setPacks] = useState<PackSummary[]>([]);
  const [mine, setMine] = useState<MyPack[]>([]);
  const [copied, setCopied] = useState(false);
  const { you, opponent, config, practice } = state;
  const isHost = you.host;

  useEffect(() => {
    listPacks().then(setPacks).catch(() => {});
  }, []);
  useEffect(() => {
    if (guest && isHost) myPacks(guest).then(setMine).catch(() => {});
  }, [guest, isHost]);

  /** The server only plays player-made packs it has loaded, so load first. */
  async function pickCustom(id: string): Promise<string | null> {
    try {
      const p = await getPack(id, guest);
      if (p.hidden) return "That pack is hidden after reports.";
      await setConfig({ pack_id: id });
      return null;
    } catch (e) {
      return e instanceof ApiError && e.status === 404 ? "No pack at that link." : "Couldn't load that pack.";
    }
  }

  const inviteURL = typeof window === "undefined" ? "" : `${window.location.origin}/room/?code=${state.room}`;
  const pack = packs.find((p) => p.id === config.pack_id);

  function setConfig(patch: Partial<Config>) {
    return send("set_config", {
      pack_id: config.pack_id,
      difficulty: config.difficulty,
      timer_s: config.timer_s,
      free_text: config.free_text,
      ...patch,
    });
  }

  async function copy() {
    try {
      await navigator.clipboard.writeText(inviteURL);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      window.prompt("Copy this invite link", inviteURL);
    }
  }

  async function share() {
    try {
      await navigator.share({ title: "Join my game", text: `Join my game in room ${state.room}`, url: inviteURL });
    } catch {
      /* cancelled */
    }
  }

  async function leave() {
    await send("leave");
    router.push("/");
  }

  const poolSize = (d: string) => {
    const total = pack?.candidates ?? config.pool_size;
    return Math.min(POOL[d] ?? total, total);
  };

  let startHint = "";
  if (!opponent) startHint = "Waiting for a friend to join.";
  else if (!opponent.ready) startHint = `Waiting for ${opponent.name} to get ready.`;

  return (
    <div className="grid gap-8 pt-4 lg:grid-cols-[1fr_1fr] lg:gap-12 lg:pt-8">
      <section>
        <h1 className="text-4xl font-extrabold sm:text-5xl">
          {practice ? "Practice round" : (
            <>
              Room <span className="tabular tracking-wider">{state.room}</span>
            </>
          )}
        </h1>

        {!practice && (
          <div className="mt-5">
            <p className="text-ink-soft">Send this link to the friend you want to play.</p>
            <div className="mt-2 flex flex-col gap-2 sm:flex-row">
              <input className="field text-sm" readOnly value={inviteURL} aria-label="Invite link" onFocus={(e) => e.target.select()} />
              <div className="flex gap-2">
                <button className="btn btn-ink grow" onClick={copy}>
                  {copied ? "Copied" : "Copy link"}
                </button>
                {typeof navigator !== "undefined" && "share" in navigator && (
                  <button className="btn grow" onClick={share}>
                    Share
                  </button>
                )}
              </div>
            </div>
          </div>
        )}

        <ul className="mt-8 flex flex-col gap-3" aria-label="Players">
          <SeatRow p={you} you />
          {opponent ? <SeatRow p={opponent} /> : (
            <li className="flex min-h-16 items-center rounded-[var(--radius-panel)] border-2 border-dashed border-rule px-4 text-ink-soft">
              Open seat
            </li>
          )}
        </ul>

        <div className="mt-6 flex flex-wrap items-center gap-3">
          {isHost ? (
            <button className="btn btn-ink h-14 px-8 text-lg" disabled={!opponent?.ready} onClick={() => send("start")}>
              Start the round
            </button>
          ) : (
            <button
              className={`btn h-14 px-8 text-lg ${you.ready ? "" : "btn-ink"}`}
              aria-pressed={you.ready}
              onClick={() => send("ready", { ready: !you.ready })}
            >
              {you.ready ? "Not ready yet" : "I'm ready"}
            </button>
          )}
          <button className="btn btn-quiet" onClick={leave}>
            Leave room
          </button>
        </div>
        {isHost && startHint && <p className="mt-3 text-ink-soft">{startHint}</p>}
        {!isHost && you.ready && <p className="mt-3 text-ink-soft">Waiting for {opponent?.name ?? "the host"} to start.</p>}
      </section>

      <section className="panel h-fit p-5 sm:p-6">
        <h2 className="text-2xl font-bold">Round settings</h2>
        {!isHost && <p className="mt-1 text-ink-soft">The host chooses these.</p>}

        <fieldset className="mt-5" disabled={!isHost}>
          <legend className="font-semibold">Topic pack</legend>
          <p className="mt-1">
            {config.pack_title}
            {pack && <span className="text-ink-soft"> in {pack.category}</span>}
            {config.pack_author && <span className="text-ink-soft"> · player-made by {config.pack_author}</span>}
          </p>
          {isHost && packs.length > 1 && (
            <PackPicker
              packs={packs}
              value={config.pack_id}
              onChange={(id) => setConfig({ pack_id: id })}
              mine={mine}
              onPickCustom={pickCustom}
            />
          )}
          {config.pack_draft && (
            <p className="mt-1 text-sm text-ink-soft">
              This pack&apos;s data is still being checked. Spot a wrong answer? Note the question and tell us.
            </p>
          )}
        </fieldset>

        <fieldset className="mt-5" disabled={!isHost}>
          <legend className="font-semibold">Difficulty</legend>
          <div className="mt-2 flex flex-wrap gap-2">
            {DIFFICULTIES.map((d) => (
              <button
                key={d.id}
                type="button"
                className="chip disabled:opacity-60"
                aria-pressed={config.difficulty === d.id}
                onClick={() => setConfig({ difficulty: d.id })}
              >
                {d.label}
                <span className="ml-1.5 opacity-70 tabular">{poolSize(d.id)}</span>
              </button>
            ))}
          </div>
          <p className="mt-1.5 text-sm text-ink-soft">The number is how many candidates are in play.</p>
        </fieldset>

        <fieldset className="mt-5" disabled={!isHost}>
          <legend className="font-semibold">Move timer</legend>
          <div className="mt-2 flex flex-wrap gap-2">
            {TIMERS.map((t) => (
              <button
                key={t}
                type="button"
                className="chip disabled:opacity-60"
                aria-pressed={config.timer_s === t}
                onClick={() => setConfig({ timer_s: t })}
              >
                {t === 0 ? "Off" : `${t}s`}
              </button>
            ))}
          </div>
          <p className="mt-1.5 text-sm text-ink-soft">
            {config.timer_s === 0
              ? "No time pressure. Take as long as you like."
              : `If you take longer than ${config.timer_s} seconds on a move, it costs you a move.`}
          </p>
        </fieldset>

        {!practice && (
          <fieldset className="mt-5" disabled={!isHost}>
            <legend className="font-semibold">Typed questions</legend>
            <div className="mt-2 flex flex-wrap gap-2">
              {[false, true].map((on) => (
                <button
                  key={String(on)}
                  type="button"
                  className="chip disabled:opacity-60"
                  aria-pressed={config.free_text === on}
                  onClick={() => setConfig({ free_text: on })}
                >
                  {on ? "Allowed" : "Only answerable ones"}
                </button>
              ))}
            </div>
            <p className="mt-1.5 text-sm text-ink-soft">
              {config.free_text
                ? "Questions the game can't answer go to the other player, who answers Yes, No or Not sure (3 times at most). Answers are checked after the round; two wrong ones lose the win."
                : "Typed or spoken questions must match something the game can answer from its data."}
            </p>
          </fieldset>
        )}

        <div className="mt-6 border-t border-rule-soft pt-4 text-sm text-ink-soft">
          <h3 className="mb-1 font-semibold text-ink">How a round works</h3>
          <ul className="list-disc space-y-1 pl-5">
            <li>You both secretly pick one candidate.</li>
            <li>You hunt at the same time. Each question costs 1 move.</li>
            <li>A wrong guess costs {config.wrong_guess_cost} moves. A correct guess ends your hunt.</li>
            <li>Fewer moves wins. On a tie, the faster player wins.</li>
          </ul>
        </div>
      </section>
    </div>
  );
}

function SeatRow({ p, you = false }: { p: PlayerView; you?: boolean }) {
  return (
    <li className="panel flex min-h-16 items-center justify-between gap-3 px-4 py-3">
      <div className="flex items-center gap-3">
        <span
          className={`h-2.5 w-2.5 shrink-0 rounded-full ${p.connected ? "bg-yes" : "bg-unk"}`}
          aria-label={p.connected ? "Online" : "Offline"}
        />
        <div>
          <p className="font-semibold">
            {p.name}
            {you && <span className="font-normal text-ink-soft"> (you)</span>}
          </p>
          <p className="text-sm text-ink-soft">
            {p.bot ? "Computer opponent" : p.host ? "Host" : p.connected ? "Guest" : "Away, can rejoin with the same link"}
          </p>
        </div>
      </div>
      {!p.host && (
        <span className={`text-sm font-semibold ${p.ready ? "text-yes" : "text-ink-soft"}`}>{p.ready ? "Ready" : "Not ready"}</span>
      )}
    </li>
  );
}
