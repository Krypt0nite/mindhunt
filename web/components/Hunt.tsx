"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import type { MatchView, Move, PoolView, Predicate, ServerMsg, StateView } from "@/lib/protocol";
import { useNow, type Send } from "@/lib/useRoom";
import { BallStrip } from "./common";
import { Board, CandidateDetail, FlashLine, History, type Flash } from "./HuntParts";
import { QuestionBuilder } from "./QuestionBuilder";
import { IncomingQuestion, TextAsk } from "./TextAsk";


export function Hunt({
  state,
  pool,
  send,
  clockSkew,
  lastResult,
}: {
  state: StateView;
  pool: PoolView;
  send: Send;
  clockSkew: number;
  lastResult: { seq: number; msg: ServerMsg } | null;
}) {
  const hunt = state.hunt!;
  const opp = state.opponent;
  const done = hunt.your_status !== "hunting";
  const [busy, setBusy] = useState(false);
  const [flash, setFlash] = useState<Flash | null>(null);
  const [selected, setSelected] = useState<string | null>(null);
  const [tab, setTab] = useState<"ask" | "board">("ask");
  const [confirmGiveUp, setConfirmGiveUp] = useState(false);
  const [mode, setMode] = useState<"pick" | "type">("pick");

  const remaining = useMemo(() => new Set(hunt.remaining_ids), [hunt.remaining_ids]);
  const byId = useMemo(() => new Map(pool.candidates.map((c) => [c.id, c])), [pool.candidates]);

  async function ask(p: Predicate) {
    setBusy(true);
    await send("ask", { predicate: p });
    setBusy(false);
  }

  async function guess(id: string) {
    setBusy(true);
    await send("guess", { candidate_id: id });
    setBusy(false);
    setSelected(null);
  }

  // Every answer and guess result (including manual answers that arrive
  // later) shows up here.
  const mountSeq = useRef(lastResult?.seq ?? 0); // ignore results from an earlier round
  useEffect(() => {
    if (!lastResult || lastResult.seq <= mountSeq.current) return;
    const m = lastResult.msg.data as Move;
    if (lastResult.msg.type === "guess_result") {
      const name = byId.get(m.guess_id ?? "")?.name ?? "that";
      setFlash(
        m.answer === "correct"
          ? { text: `It's ${name}!`, tone: "correct", detail: `Found in ${m.total} moves.` }
          : { text: `It's not ${name}.`, tone: "wrong", detail: `That cost ${m.cost} moves.` },
      );
      return;
    }
    const tone = (m.answer ?? "unknown") as Flash["tone"];
    setFlash({
      text: m.text,
      tone,
      detail: m.manual
        ? `${opp?.name ?? "Your opponent"} answered this one. The board only changes for questions the game answers.`
        : tone === "unknown"
          ? "No data for this one, so nothing is ruled out."
          : "",
    });
  }, [lastResult]); // eslint-disable-line react-hooks/exhaustive-deps

  // Newly answered move: keep the flash for a few seconds.
  useEffect(() => {
    if (!flash) return;
    const t = setTimeout(() => setFlash(null), 7000);
    return () => clearTimeout(t);
  }, [flash]);

  return (
    <div className="pt-3 lg:pt-6">
      <StatusBar state={state} send={send} clockSkew={clockSkew} />
      {hunt.incoming && (
        <div className="mt-4">
          <IncomingQuestion incoming={hunt.incoming} opponent={opp?.name ?? "Your opponent"} unknownsLeft={hunt.unknowns_left} send={send} />
        </div>
      )}

      <div className="mt-4 flex gap-2 lg:hidden" role="tablist">
        <button role="tab" aria-selected={tab === "ask"} className="chip grow justify-center" aria-pressed={tab === "ask"} onClick={() => setTab("ask")}>
          Ask and guess
        </button>
        <button role="tab" aria-selected={tab === "board"} className="chip grow justify-center" aria-pressed={tab === "board"} onClick={() => setTab("board")}>
          Board, {remaining.size} left
        </button>
      </div>

      <div className="mt-4 grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.1fr)] lg:gap-8">
        <section className={`${tab === "ask" ? "" : "hidden"} flex flex-col gap-5 lg:flex`}>
          {flash && <FlashLine flash={flash} remaining={remaining.size} />}
          {done ? (
            <div className="panel p-5">
              <h2 className="text-2xl font-bold">
                {hunt.your_status === "solved" ? `You found it in ${hunt.your_moves} moves` : "You gave up"}
              </h2>
              <p className="mt-1 text-ink-soft">
                {opp?.status === "hunting"
                  ? `${opp.name} is still hunting. Results appear when they finish.`
                  : "Working out the result…"}
              </p>
            </div>
          ) : (
            <>
              <div className="panel p-4 sm:p-5">
                <div className="mb-4 flex gap-2" role="tablist" aria-label="How to ask">
                  <button role="tab" aria-selected={mode === "pick"} aria-pressed={mode === "pick"} className="chip" onClick={() => setMode("pick")}>
                    Pick a question
                  </button>
                  <button role="tab" aria-selected={mode === "type"} aria-pressed={mode === "type"} className="chip" onClick={() => setMode("type")}>
                    {hunt.pending ? "Waiting for answer" : "Type or say"}
                  </button>
                </div>
                {mode === "pick" ? (
                  <QuestionBuilder attributes={pool.attributes} log={hunt.your_log} busy={busy} onAsk={ask} />
                ) : (
                  <TextAsk
                    preview={async (text) => {
                      const r = await send("match_text", { text });
                      return r?.type === "match" ? (r.data as MatchView) : null;
                    }}
                    submitText={async (text) => {
                      const r = await send("ask_text", { text });
                      return !!r && r.type !== "error";
                    }}
                    cancelPending={() => send("cancel_text")}
                    freeText={state.config.free_text}
                    opponent={opp?.name ?? "your opponent"}
                    pending={hunt.pending}
                    busy={busy}
                    onAskPredicate={ask}
                  />
                )}
              </div>
              <div className="panel flex flex-wrap items-center justify-between gap-3 p-4 sm:p-5">
                <div>
                  <p className="font-semibold">Think you know?</p>
                  <p className="text-sm text-ink-soft">
                    A wrong guess costs {state.config.wrong_guess_cost} moves. Pick a name on the board to guess.
                  </p>
                </div>
                <button className="btn btn-ink lg:hidden" onClick={() => setTab("board")}>
                  Make a guess
                </button>
              </div>
            </>
          )}
          <History log={hunt.your_log} opponent={opp?.name ?? "Opponent"} />
          {!done && (
            <div className="text-sm">
              {confirmGiveUp ? (
                <span className="flex flex-wrap items-center gap-2">
                  <span className="text-ink-soft">Giving up scores {Math.max(pool.candidates.length, hunt.your_moves) + 10} moves.</span>
                  <button className="btn h-9 min-h-0 border-no text-no" onClick={() => send("give_up")}>
                    Give up this round
                  </button>
                  <button className="btn btn-quiet" onClick={() => setConfirmGiveUp(false)}>
                    Keep hunting
                  </button>
                </span>
              ) : (
                <button className="btn btn-quiet text-ink-soft" onClick={() => setConfirmGiveUp(true)}>
                  Give up
                </button>
              )}
            </div>
          )}
        </section>

        <section className={`${tab === "board" ? "" : "hidden"} lg:block`}>
          <Board
            pool={pool}
            remaining={remaining}
            guessed={new Set(hunt.your_log.filter((m) => m.kind === "guess").map((m) => m.guess_id!))}
            selected={selected}
            onSelect={setSelected}
          />
          {selected && byId.get(selected) && (
            <CandidateDetail
              c={byId.get(selected)!}
              pool={pool}
              live={remaining.has(selected)}
              canGuess={!done && !hunt.your_log.some((m) => m.guess_id === selected)}
              busy={busy}
              cost={state.config.wrong_guess_cost}
              onGuess={() => guess(selected)}
              onClose={() => setSelected(null)}
            />
          )}
        </section>
      </div>
    </div>
  );
}

function StatusBar({ state, send, clockSkew }: { state: StateView; send: Send; clockSkew: number }) {
  const hunt = state.hunt!;
  const opp = state.opponent;
  const now = useNow(500) + clockSkew;
  const deadline = hunt.deadline ? Date.parse(hunt.deadline) : null;
  const secsLeft = deadline ? Math.max(0, Math.ceil((deadline - now) / 1000)) : null;
  const claimAt = opp?.claimable_at ? Date.parse(opp.claimable_at) : null;
  const claimIn = claimAt ? Math.max(0, Math.ceil((claimAt - now) / 1000)) : null;

  let oppText = "Still hunting";
  if (opp?.status === "solved") oppText = "Found your secret";
  if (opp?.status === "gave_up") oppText = "Gave up";
  if (opp && !opp.connected) oppText = "Away";

  return (
    <div className="panel grid grid-cols-[1fr_auto] items-center gap-x-4 gap-y-3 p-4 sm:grid-cols-[auto_1fr_auto] sm:p-5">
      <div>
        <p className="text-sm text-ink-soft">Your moves</p>
        <p className="font-[family-name:var(--font-display)] text-5xl font-extrabold leading-none tabular" aria-live="polite">
          {hunt.your_moves}
        </p>
      </div>
      <div className="order-3 col-span-2 sm:order-none sm:col-span-1">
        <BallStrip log={hunt.your_log} empty="Ask your first question to start the count." />
        {secsLeft !== null && (
          <p className={`mt-2 text-sm tabular ${secsLeft <= 10 ? "font-semibold text-no" : "text-ink-soft"}`}>
            {secsLeft}s left for this move
          </p>
        )}
      </div>
      <div className="text-right">
        <p className="text-sm text-ink-soft">{opp?.name ?? "Opponent"}</p>
        <p className={`font-semibold ${opp?.status === "solved" ? "text-no" : ""}`}>{oppText}</p>
        {claimIn !== null && (
          claimIn > 0 ? (
            <p className="text-sm text-ink-soft tabular">You can claim the win in {claimIn}s</p>
          ) : (
            hunt.your_status !== "gave_up" && (
              <button className="btn btn-ink mt-1 h-9 min-h-0" onClick={() => send("claim_win")}>
                Claim the win
              </button>
            )
          )
        )}
      </div>
    </div>
  );
}

