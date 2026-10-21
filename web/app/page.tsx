"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState, type FormEvent } from "react";
import { NameForm, Wordmark } from "@/components/common";
import { groupPacks } from "@/components/PackPicker";
import { createRoom, getDaily, listPacks, myPacks } from "@/lib/api";
import { useApp } from "@/lib/app";
import type { DailyView, MyPack, PackSummary } from "@/lib/protocol";
import { loadRecent, winStreak, type RecentResult } from "@/lib/recent";

const SAMPLE: { q: string; a: "yes" | "no" | "unknown" }[] = [
  { q: "Have they represented India?", a: "yes" },
  { q: "Are they a spin bowler?", a: "no" },
  { q: "Did they debut before the 2000s?", a: "yes" },
  { q: "Have they won the 50-over World Cup?", a: "yes" },
];

export default function Home() {
  const { guest, ready, serverError, config } = useApp();
  const router = useRouter();
  const [busy, setBusy] = useState<"friend" | "practice" | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [code, setCode] = useState("");
  const [packs, setPacks] = useState<PackSummary[]>([]);
  const [recent, setRecent] = useState<RecentResult[]>([]);
  const [daily, setDaily] = useState<DailyView | null>(null);
  const [mine, setMine] = useState<MyPack[]>([]);

  useEffect(() => {
    listPacks().then(setPacks).catch(() => {});
    setRecent(loadRecent());
  }, []);

  useEffect(() => {
    if (guest) getDaily(guest).then(setDaily).catch(() => setDaily(null));
    if (guest) myPacks(guest).then(setMine).catch(() => setMine([]));
  }, [guest]);

  async function open(practice: boolean) {
    if (!guest) return;
    setBusy(practice ? "practice" : "friend");
    setError(null);
    try {
      const r = await createRoom(guest, practice);
      router.push(`/room/?code=${r.code}`);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Couldn't create a room.");
      setBusy(null);
    }
  }

  function join(e: FormEvent) {
    e.preventDefault();
    const c = code.trim().toUpperCase().replace(/[^A-Z0-9]/g, "");
    if (c.length !== 6) {
      setError("Room codes are 6 letters and numbers, like K7QM2X.");
      return;
    }
    router.push(`/room/?code=${c}`);
  }

  const streak = winStreak(recent);
  const wrongCost = config?.wrong_guess_cost ?? 3;

  return (
    <>
      <main className="mx-auto grid w-full max-w-6xl gap-10 px-4 pb-16 pt-8 sm:px-6 lg:grid-cols-[1.15fr_1fr] lg:gap-14 lg:pt-12">
        <section>
          <Wordmark size="lg" />
          <p className="mt-5 max-w-[34ch] text-xl leading-snug text-ink-soft sm:text-2xl">
            You each pick a secret. Ask yes-or-no questions. Whoever finds the other&apos;s secret in fewer moves wins.
          </p>

          <figure className="panel mt-8 max-w-lg p-4 sm:p-5" aria-label="Example of a hunt">
            <ol className="flex flex-col gap-2.5">
              {SAMPLE.map((s, i) => (
                <li key={s.q} className="flex items-center gap-3">
                  <span className={`ball ball-${s.a}`}>{s.a === "yes" ? "Y" : "N"}</span>
                  <span className="text-[0.98rem]">{s.q}</span>
                  <span className="ml-auto text-sm text-ink-soft tabular">{i + 1}</span>
                </li>
              ))}
              <li className="flex items-center gap-3 border-t border-rule-soft pt-2.5">
                <span className="ball ball-correct">✓</span>
                <span className="font-semibold">Is it Kapil Dev?</span>
                <span className="ml-auto text-sm font-semibold tabular">5</span>
              </li>
            </ol>
            <figcaption className="mt-3 text-sm text-ink-soft">
              Solved in 5 moves. A wrong guess costs {wrongCost}, so ask before you guess.
            </figcaption>
          </figure>
        </section>

        <section className="flex flex-col gap-6">
          <div className="panel p-5 sm:p-6">
            {!ready ? (
              <p className="text-ink-soft">Connecting to the game server…</p>
            ) : serverError ? (
              <div role="alert">
                <h2 className="text-xl font-bold">Can&apos;t reach the game server</h2>
                <p className="mt-2 text-ink-soft">{serverError}</p>
                <button className="btn mt-4" onClick={() => window.location.reload()}>
                  Try again
                </button>
              </div>
            ) : !guest ? (
              <>
                <h2 className="text-2xl font-bold">What should we call you?</h2>
                <p className="mb-4 mt-1 text-ink-soft">No account needed. Your opponent sees this name.</p>
                <NameForm submitLabel="Continue" />
              </>
            ) : (
              <>
                <h2 className="text-2xl font-bold">Play</h2>
                <div className="mt-4 flex flex-col gap-3">
                  <button className="btn btn-ink h-14 text-lg" onClick={() => open(false)} disabled={busy !== null}>
                    {busy === "friend" ? "Creating room…" : "Play with a friend"}
                  </button>
                  <button className="btn h-12" onClick={() => open(true)} disabled={busy !== null}>
                    {busy === "practice" ? "Setting up…" : "Practice against the bot"}
                  </button>
                </div>
                <form onSubmit={join} className="mt-5 border-t border-rule-soft pt-5">
                  <label htmlFor="room-code" className="font-semibold">
                    Have a room code?
                  </label>
                  <div className="mt-2 flex gap-2">
                    <input
                      id="room-code"
                      className="field uppercase tracking-[0.2em] tabular"
                      placeholder="K7QM2X"
                      maxLength={6}
                      autoCapitalize="characters"
                      autoComplete="off"
                      value={code}
                      onChange={(e) => setCode(e.target.value)}
                    />
                    <button className="btn">Join</button>
                  </div>
                </form>
                {error && (
                  <p role="alert" className="mt-3 text-sm text-no">
                    {error}
                  </p>
                )}
                <details className="mt-5 text-sm text-ink-soft">
                  <summary className="cursor-pointer">Change your name ({guest.display_name})</summary>
                  <div className="mt-3">
                    <NameForm submitLabel="Save name" initial={guest.display_name} />
                  </div>
                </details>
              </>
            )}
          </div>

          {guest && daily && (
            <Link href="/daily/" className="panel group block p-5 hover:border-ink sm:p-6">
              <span className="flex items-baseline justify-between gap-3">
                <span className="text-2xl font-bold">Daily #{daily.number}</span>
                {daily.streak.current > 0 && (
                  <span className="text-sm text-ink-soft">
                    Streak <strong className="text-ink tabular">{daily.streak.current}</strong>
                  </span>
                )}
              </span>
              <span className="mt-1 block text-ink-soft">
                {daily.status === "solved"
                  ? `Solved in ${daily.result?.score ?? daily.moves} moves. A new one starts at midnight.`
                  : daily.status === "gave_up"
                    ? "You gave up today. A new one starts at midnight."
                    : daily.moves > 0
                      ? `In progress: ${daily.moves} ${daily.moves === 1 ? "move" : "moves"} so far. Pick up where you left off.`
                      : "One secret, the same for everyone today. Solo, no time limit."}
              </span>
              <span className="mt-3 inline-flex font-semibold underline underline-offset-4 group-hover:no-underline">
                {daily.status === "hunting" ? (daily.moves > 0 ? "Continue today's challenge" : "Play today's challenge") : "See today's result"}
              </span>
            </Link>
          )}

          {guest && (
            <div className="panel p-5 sm:p-6">
              <h2 className="text-xl font-bold">Make your own pack</h2>
              <p className="mt-1 text-ink-soft">
                Your colleagues, your cricket club, your favourite films: list them, add a few facts, share the link.
              </p>
              <Link href="/create/" className="btn mt-4">
                Start a pack
              </Link>
              {mine.length > 0 && (
                <ul className="mt-4 flex flex-col gap-1.5 border-t border-rule-soft pt-3 text-sm" aria-label="Your packs">
                  {mine.slice(0, 5).map((p) => (
                    <li key={p.id} className="flex items-baseline justify-between gap-3">
                      <Link href={`/pack/?id=${p.id}`} className="font-semibold underline-offset-4 hover:underline">
                        {p.title}
                      </Link>
                      <span className="text-ink-soft">{p.hidden ? "Hidden after reports" : `v${p.version}`}</span>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          )}

          {packs.length > 0 && (
            <div>
              <h2 className="text-lg font-bold">Topic packs</h2>
              <p className="mt-1 text-sm text-ink-soft">The host picks one in the room, or hits Surprise me.</p>
              <dl className="mt-3 flex flex-col gap-3">
                {groupPacks(packs).map(([category, list]) => (
                  <div key={category} className="grid grid-cols-[6.5rem_1fr] gap-3 border-b border-rule-soft pb-3">
                    <dt className="font-semibold">{category}</dt>
                    <dd className="flex flex-col gap-1">
                      {list.map((p) => (
                        <span key={p.id} className="flex items-baseline justify-between gap-3">
                          <span>{p.title}</span>
                          <span className="shrink-0 text-sm text-ink-soft tabular">{p.candidates} to find</span>
                        </span>
                      ))}
                    </dd>
                  </div>
                ))}
              </dl>
            </div>
          )}

          {recent.length > 0 && (
            <div>
              <h2 className="text-lg font-bold">
                Your recent games{streak >= 2 && <span className="font-normal text-ink-soft">, {streak} wins in a row</span>}
              </h2>
              <ul className="mt-2 flex flex-col gap-1.5 text-sm">
                {recent.slice(0, 5).map((r) => (
                  <li key={`${r.room}-${r.round}-${r.at}`} className="flex justify-between gap-3">
                    <span>
                      <span
                        className={
                          r.outcome === "won" ? "font-semibold text-yes" : r.outcome === "lost" ? "text-no" : "text-ink-soft"
                        }
                      >
                        {r.outcome === "won" ? "Won" : r.outcome === "lost" ? "Lost" : "Draw"}
                      </span>{" "}
                      vs {r.opponent}
                    </span>
                    <span className="text-ink-soft tabular">
                      {r.moves} to {r.opponentMoves} moves
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </section>
      </main>
    </>
  );
}
