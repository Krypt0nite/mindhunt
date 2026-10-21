"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";
import { BallStrip, Header, NameForm, Portrait, Toasts, describeAnswer } from "@/components/common";
import { Board, CandidateDetail, FlashLine, History, type Flash } from "@/components/HuntParts";
import { QuestionBuilder } from "@/components/QuestionBuilder";
import { TextAsk } from "@/components/TextAsk";
import { ApiError, getDaily, matchDaily, playDaily } from "@/lib/api";
import { useApp } from "@/lib/app";
import type { DailyAction, DailyView, GameError, PoolView } from "@/lib/protocol";
import { useNow } from "@/lib/useRoom";

export default function DailyPage() {
  const { guest, ready, serverError } = useApp();
  return (
    <>
      <Header />
      <main className="mx-auto w-full max-w-6xl px-4 pb-24 sm:px-6">
        {!ready ? null : serverError ? (
          <p className="pt-10 text-ink-soft">{serverError}</p>
        ) : !guest ? (
          <section className="max-w-xl pt-10">
            <h1 className="text-4xl font-extrabold">Today&apos;s daily challenge</h1>
            <p className="mb-5 mt-2 text-ink-soft">Choose a name to play. Your streak is kept under this name on this device.</p>
            <NameForm submitLabel="Play today's challenge" />
          </section>
        ) : (
          <Daily />
        )}
      </main>
    </>
  );
}

function Daily() {
  const { guest, appName } = useApp();
  const [view, setView] = useState<DailyView | null>(null);
  const [pool, setPool] = useState<PoolView | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [flash, setFlash] = useState<Flash | null>(null);
  const [selected, setSelected] = useState<string | null>(null);
  const [tab, setTab] = useState<"ask" | "board">("ask");
  const [mode, setMode] = useState<"pick" | "type">("pick");
  const [confirmGiveUp, setConfirmGiveUp] = useState(false);
  const [errors, setErrors] = useState<{ id: number; error: GameError }[]>([]);

  const load = useCallback(async () => {
    if (!guest) return;
    try {
      const v = await getDaily(guest);
      setView(v);
      if (v.pool) setPool(v.pool);
    } catch (e) {
      setLoadError(e instanceof Error ? e.message : "Couldn't load today's challenge.");
    }
  }, [guest]);

  useEffect(() => {
    load();
  }, [load]);

  const pushError = (message: string, code = "error") => {
    const id = Date.now() + Math.random();
    setErrors((e) => [...e.slice(-2), { id, error: { code, message } }]);
    setTimeout(() => setErrors((e) => e.filter((x) => x.id !== id)), 6000);
  };

  async function play(action: DailyAction): Promise<boolean> {
    if (!guest) return false;
    setBusy(true);
    try {
      const v = await playDaily(guest, action);
      setView(v);
      const m = v.last;
      if (m) {
        if (m.kind === "guess") {
          const name = pool?.candidates.find((c) => c.id === m.guess_id)?.name ?? "that";
          setFlash(
            m.answer === "correct"
              ? { text: `It's ${name}!`, tone: "correct", detail: `Found in ${m.total} moves.` }
              : { text: `It's not ${name}.`, tone: "wrong", detail: `That cost ${m.cost} moves.` },
          );
        } else if (m.kind === "ask") {
          const tone = (m.answer ?? "unknown") as Flash["tone"];
          setFlash({ text: m.text, tone, detail: tone === "unknown" ? "No data for this one, so nothing is ruled out." : "" });
        }
      }
      setSelected(null);
      return true;
    } catch (e) {
      pushError(e instanceof ApiError ? e.message : "Couldn't save your move. Try again.");
      return false;
    } finally {
      setBusy(false);
    }
  }

  const remaining = useMemo(() => new Set(view?.remaining_ids ?? []), [view?.remaining_ids]);
  const byId = useMemo(() => new Map((pool?.candidates ?? []).map((c) => [c.id, c])), [pool]);

  if (loadError) {
    return (
      <section className="max-w-xl pt-10">
        <h1 className="text-3xl font-extrabold">Can&apos;t load today&apos;s challenge</h1>
        <p className="mt-2 text-ink-soft">{loadError}</p>
        <button className="btn btn-ink mt-6" onClick={load}>
          Try again
        </button>
      </section>
    );
  }
  if (!view || !pool) return <p className="pt-10 text-ink-soft">Loading today&apos;s challenge…</p>;

  const day = new Date(`${view.date}T00:00:00`).toLocaleDateString(undefined, { weekday: "long", day: "numeric", month: "long" });

  return (
    <div className="pt-4 lg:pt-8">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-4xl font-extrabold sm:text-5xl">Daily #{view.number}</h1>
          <p className="mt-1 text-ink-soft">
            {day}. Today&apos;s pack is {view.pack_title}: everyone is hunting the same {view.pack_noun || "secret"}, out of{" "}
            {pool.candidates.length}.
          </p>
        </div>
        {view.streak.current > 0 && (
          <p className="text-ink-soft">
            Streak <strong className="text-ink tabular">{view.streak.current}</strong> {view.streak.current === 1 ? "day" : "days"}
          </p>
        )}
      </div>

      {view.result ? (
        <DailyDone view={view} appName={appName} />
      ) : (
        <>
          <div className="panel mt-5 flex flex-wrap items-center gap-x-6 gap-y-3 p-4 sm:p-5">
            <div>
              <p className="text-sm text-ink-soft">Your moves</p>
              <p className="font-[family-name:var(--font-display)] text-5xl font-extrabold leading-none tabular" aria-live="polite">
                {view.moves}
              </p>
            </div>
            <div className="min-w-0 grow">
              <BallStrip log={view.log} empty="Ask your first question to start the count." />
            </div>
          </div>

          <div className="mt-4 flex gap-2 lg:hidden" role="tablist">
            <button role="tab" aria-selected={tab === "ask"} aria-pressed={tab === "ask"} className="chip grow justify-center" onClick={() => setTab("ask")}>
              Ask and guess
            </button>
            <button role="tab" aria-selected={tab === "board"} aria-pressed={tab === "board"} className="chip grow justify-center" onClick={() => setTab("board")}>
              Board, {remaining.size} left
            </button>
          </div>

          <div className="mt-4 grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.1fr)] lg:gap-8">
            <section className={`${tab === "ask" ? "" : "hidden"} flex flex-col gap-5 lg:flex`}>
              {flash && <FlashLine flash={flash} remaining={remaining.size} />}
              <div className="panel p-4 sm:p-5">
                <div className="mb-4 flex gap-2" role="tablist" aria-label="How to ask">
                  <button role="tab" aria-selected={mode === "pick"} aria-pressed={mode === "pick"} className="chip" onClick={() => setMode("pick")}>
                    Pick a question
                  </button>
                  <button role="tab" aria-selected={mode === "type"} aria-pressed={mode === "type"} className="chip" onClick={() => setMode("type")}>
                    Type or say
                  </button>
                </div>
                {mode === "pick" ? (
                  <QuestionBuilder attributes={pool.attributes} log={view.log} busy={busy} onAsk={(p) => play({ predicate: p })} />
                ) : (
                  <TextAsk
                    preview={(t) => matchDaily(t).catch(() => null)}
                    submitText={(t) => play({ text: t })}
                    freeText={false}
                    opponent=""
                    busy={busy}
                    onAskPredicate={(p) => play({ predicate: p })}
                  />
                )}
              </div>
              <div className="panel flex flex-wrap items-center justify-between gap-3 p-4 sm:p-5">
                <div>
                  <p className="font-semibold">Think you know?</p>
                  <p className="text-sm text-ink-soft">A wrong guess costs {view.wrong_guess_cost} moves. Pick a name on the board to guess.</p>
                </div>
                <button className="btn btn-ink lg:hidden" onClick={() => setTab("board")}>
                  Make a guess
                </button>
              </div>
              <History log={view.log} opponent="" />
              <div className="text-sm">
                {confirmGiveUp ? (
                  <span className="flex flex-wrap items-center gap-2">
                    <span className="text-ink-soft">Giving up ends today&apos;s streak and shows the answer.</span>
                    <button className="btn h-9 min-h-0 border-no text-no" onClick={() => play({ give_up: true })}>
                      Give up today
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
            </section>

            <section className={`${tab === "board" ? "" : "hidden"} lg:block`}>
              <Board
                pool={pool}
                remaining={remaining}
                guessed={new Set(view.log.filter((m) => m.kind === "guess").map((m) => m.guess_id!))}
                selected={selected}
                onSelect={setSelected}
              />
              {selected && byId.get(selected) && (
                <CandidateDetail
                  c={byId.get(selected)!}
                  pool={pool}
                  live={remaining.has(selected)}
                  canGuess={!view.log.some((m) => m.guess_id === selected)}
                  busy={busy}
                  cost={view.wrong_guess_cost}
                  onGuess={() => play({ candidate_id: selected })}
                  onClose={() => setSelected(null)}
                />
              )}
            </section>
          </div>
        </>
      )}
      <Toasts errors={errors} onDismiss={(id) => setErrors((e) => e.filter((x) => x.id !== id))} />
    </div>
  );
}

function DailyDone({ view, appName }: { view: DailyView; appName: string }) {
  const r = view.result!;
  const s = r.secret;
  const solved = view.status === "solved";
  const [copied, setCopied] = useState(false);
  const now = useNow(1000);
  const left = Math.max(0, Date.parse(view.next) - now);
  const hh = Math.floor(left / 3600000);
  const mm = Math.floor((left % 3600000) / 60000);
  const ss = Math.floor((left % 60000) / 1000);

  const shareText = [
    `${appName} Daily #${view.number}: ${solved ? `${r.score} moves` : "gave up"}`,
    r.grid,
    `${window.location.origin}/daily/`,
  ].join("\n");

  async function share() {
    if (navigator.share) {
      try {
        await navigator.share({ text: shareText });
        return;
      } catch {
        /* fall back to copy */
      }
    }
    try {
      await navigator.clipboard.writeText(shareText);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      window.prompt("Copy your result", shareText);
    }
  }

  const scores = Object.entries(r.stats.scores)
    .map(([k, n]) => [Number(k), n] as const)
    .sort((a, b) => a[0] - b[0]);
  const top = Math.max(1, ...scores.map(([, n]) => n));

  return (
    <div className="mt-6 grid gap-8 lg:grid-cols-2">
      <section>
        <h2 className="text-5xl font-extrabold sm:text-6xl">{solved ? `Solved in ${r.score}` : "Not today"}</h2>
        <p className="mt-3 text-lg text-ink-soft">
          {solved
            ? r.stats.solved > 1
              ? `You used fewer moves than ${r.better_than}% of the ${r.stats.solved - 1} other players who solved it.`
              : "You're the first to solve today's challenge."
            : `The answer was ${s.name}.`}
        </p>
        <p className="mt-2 text-ink-soft">
          {view.streak.current > 0
            ? `Streak: ${view.streak.current} ${view.streak.current === 1 ? "day" : "days"}, best ${view.streak.best}. `
            : view.streak.best > 0
              ? `Best streak: ${view.streak.best} ${view.streak.best === 1 ? "day" : "days"}. `
              : ""}
          {view.streak.played} played, {view.streak.solved} solved.
        </p>
        <div className="mt-5 flex flex-wrap gap-3">
          <button className="btn btn-ink h-12 px-6 text-lg" onClick={share}>
            {copied ? "Copied to clipboard" : "Share result"}
          </button>
          <Link href="/" className="btn h-12">
            Play a friend
          </Link>
        </div>
        <p className="mt-4 text-sm text-ink-soft tabular">
          Next challenge in {hh}h {String(mm).padStart(2, "0")}m {String(ss).padStart(2, "0")}s
        </p>

        {scores.length > 0 && (
          <figure className="mt-8">
            <figcaption className="font-semibold">How today&apos;s solvers did</figcaption>
            <ul className="mt-3 flex flex-col gap-1.5" aria-label="Number of players by moves used">
              {scores.map(([moves, n]) => {
                const mine = solved && moves === r.score;
                return (
                  <li key={moves} className="grid grid-cols-[5rem_1fr] items-center gap-3 text-sm">
                    <span className="text-right text-ink-soft tabular">{moves} {moves === 1 ? "move" : "moves"}</span>
                    <span className="flex items-center gap-2">
                      <span
                        className={`h-6 rounded-sm ${mine ? "bg-hi" : "bg-rule"}`}
                        style={{ width: `${Math.max(4, (n / top) * 100)}%` }}
                      />
                      <span className={`tabular ${mine ? "font-semibold" : "text-ink-soft"}`}>
                        {n}
                        {mine ? " (you)" : ""}
                      </span>
                    </span>
                  </li>
                );
              })}
            </ul>
            <p className="mt-2 text-sm text-ink-soft">
              {r.stats.finished} finished today, {r.stats.finished - r.stats.solved} gave up.
            </p>
          </figure>
        )}
      </section>

      <article className="panel h-fit overflow-hidden">
        <Portrait key={s.id} card={s}>
          <p className="text-sm text-ink-soft">Today&apos;s secret</p>
          <h3 className="text-3xl font-extrabold">{s.name}</h3>
        </Portrait>
        <dl className="mx-5 mt-3 grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 border-t border-rule-soft pt-3 text-sm">
          {s.attrs.map((a) => (
            <div key={a.label} className="contents">
              <dt className="text-ink-soft">{a.label}</dt>
              <dd>{a.value}</dd>
            </div>
          ))}
        </dl>
        {r.cracked_by && (
          <p className="mx-5 mt-4 rounded-lg bg-hi-soft px-3 py-2 text-sm">
            The question that cracked it: <strong>{r.cracked_by.text}</strong> It ruled out {r.cracked_by.removed}{" "}
            {r.cracked_by.removed === 1 ? "candidate" : "candidates"}.
          </p>
        )}
        <details className="mt-4 border-t border-rule-soft">
          <summary className="cursor-pointer px-5 py-3 font-semibold">Your questions</summary>
          <ol className="flex flex-col px-5 pb-5 text-sm">
            {view.log.map((m) => (
              <li key={m.n} className="flex gap-3 border-b border-rule-soft py-1.5">
                <span className="w-5 shrink-0 text-right text-ink-soft tabular">{m.n}</span>
                <span className="grow">{m.text}</span>
                <span className="shrink-0 font-semibold">{describeAnswer(m)}</span>
              </li>
            ))}
          </ol>
        </details>
      </article>
    </div>
  );
}
