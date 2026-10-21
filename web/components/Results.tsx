"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useApp } from "@/lib/app";
import type { ResultPlayer, ResultView, StateView } from "@/lib/protocol";
import { saveRecent } from "@/lib/recent";
import type { Send } from "@/lib/useRoom";
import { BallStrip, describeAnswer, formatDuration, Portrait } from "./common";

function headline(r: ResultView, you: number): string {
  if (r.winner === -1) return r.reason === "both_gave_up" ? "Nobody found it" : "It's a draw";
  return r.winner === you ? "You won" : "You lost";
}

function explain(r: ResultView, you: number): string {
  const [a, b] = r.players;
  const me = r.players[you]!;
  const them = r.players[1 - you]!;
  const winner = r.winner >= 0 ? r.players[r.winner]! : null;
  const loser = r.winner >= 0 ? r.players[1 - r.winner]! : null;
  switch (r.reason) {
    case "fewer_moves":
      return `${who(winner!, you)} scored ${winner!.score}, ${who(loser!, you, true)} scored ${loser!.score}. Fewer moves wins.`;
    case "faster":
      return `Both scored ${a.score}. ${who(winner!, you)} took ${formatDuration(winner!.active_ms)}, ${who(loser!, you, true)} took ${formatDuration(loser!.active_ms)}, so the faster hunt wins.`;
    case "tie":
      return `Both scored ${a.score} and finished within a second of each other (${formatDuration(a.active_ms)} and ${formatDuration(b.active_ms)}).`;
    case "both_gave_up":
      return "You both gave up, so this round is a draw.";
    case "abandoned":
      return winner === me ? `${them.name} left the game, so the win is yours.` : `You were away too long, so ${them.name} claimed the win.`;
    case "contradictions": {
      if (!winner) return "You both gave answers that contradict the data, so neither of you gets the win.";
      const liar = loser!;
      return liar === me
        ? `${(me.answer_flags ?? []).length} of your answers to ${them.name} contradict the data, so the win goes to ${them.name}.`
        : `${(them.answer_flags ?? []).length} of ${them.name}'s answers contradict the data, so the win is yours.`;
    }
  }
}

function who(p: ResultPlayer, you: number, lower = false): string {
  if (p.seat === you) return lower ? "you" : "You";
  return p.name;
}

export function Results({ state, send }: { state: StateView; send: Send }) {
  const { appName } = useApp();
  const r = state.result!;
  const you = state.you.seat;
  const me = r.players[you]!;
  const them = r.players[1 - you]!;
  const outcome = r.winner === -1 ? "draw" : r.winner === you ? "won" : "lost";
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    saveRecent({
      room: state.room,
      round: state.round,
      pack: state.config.pack_title,
      opponent: them.name,
      outcome,
      moves: me.score,
      opponentMoves: them.score,
      practice: state.practice,
      at: r.ended_at,
    });
  }, [state.room, state.round, r.ended_at]); // eslint-disable-line react-hooks/exhaustive-deps

  const diff = { easy: "Easy", normal: "Normal", hard: "Hard" }[state.config.difficulty];
  const shareText = [
    `${appName}: ${state.config.pack_title} (${diff})`,
    `Me: ${me.score} moves ${me.grid}`,
    outcome === "won" ? `Won, ${me.score} to ${them.score}` : outcome === "lost" ? `Lost, ${me.score} to ${them.score}` : `Draw at ${me.score}`,
    state.practice ? "" : `${window.location.origin}/room/?code=${state.room}`,
  ]
    .filter(Boolean)
    .join("\n");

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

  return (
    <div className="pt-4 lg:pt-8">
      <section aria-live="polite">
        <h1
          className={`text-5xl font-extrabold sm:text-7xl ${outcome === "won" ? "" : outcome === "lost" ? "text-ink-soft" : ""}`}
        >
          {headline(r, you)}
        </h1>
        <p className="mt-3 max-w-[60ch] text-lg text-ink-soft">{explain(r, you)}</p>
      </section>

      <div className="mt-6 flex flex-wrap gap-3">
        {state.rematch_you ? (
          <button className="btn h-12" disabled>
            Waiting for {them.name}…
          </button>
        ) : (
          <button className="btn btn-ink h-12 px-6 text-lg" onClick={() => send("rematch")}>
            {state.rematch_opponent ? `${them.name} wants a rematch. Play again` : "Rematch"}
          </button>
        )}
        <button className="btn h-12" onClick={() => send("to_lobby")}>
          Change pack or settings
        </button>
        <button className="btn h-12" onClick={share}>
          {copied ? "Copied to clipboard" : "Share result"}
        </button>
        <Link href="/" className="btn btn-quiet h-12">
          Home
        </Link>
      </div>

      <div className="mt-10 grid gap-6 lg:grid-cols-2">
        <FactCard player={them} title={`${them.name} hid`} hunterName="You" hunter={me} />
        <FactCard player={me} title="You hid" hunterName={them.name} hunter={them} />
      </div>
    </div>
  );
}

/**
 * `player` is whose secret is shown; `hunter` is the one who hunted it, whose
 * moves and questions are listed underneath.
 */
function FactCard({
  player,
  hunter,
  title,
  hunterName,
}: {
  player: ResultPlayer;
  hunter: ResultPlayer;
  title: string;
  hunterName: string;
}) {
  const s = player.secret;
  const verdict =
    hunter.status === "solved"
      ? `${hunterName} found it in ${hunter.moves} ${hunter.moves === 1 ? "move" : "moves"}, ${formatDuration(hunter.active_ms)} of hunting.`
      : hunter.status === "gave_up"
        ? `${hunterName} gave up after ${hunter.moves} moves.`
        : `${hunterName} didn't finish (${hunter.moves} moves).`;

  return (
    <article className="panel overflow-hidden">
      <Portrait key={s.id} card={s}>
        <p className="text-sm text-ink-soft">{title}</p>
        <h2 className="text-3xl font-extrabold">{s.name}</h2>
        <p className="mt-1 text-sm">{verdict}</p>
      </Portrait>

      <dl className="mx-5 mt-3 grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 border-t border-rule-soft pt-3 text-sm">
        {s.attrs.map((a) => (
          <div key={a.label} className="contents">
            <dt className="text-ink-soft">{a.label}</dt>
            <dd>{a.value}</dd>
          </div>
        ))}
      </dl>
      {s.facts && s.facts.length > 0 && (
        <ul className="mx-5 mt-3 list-disc space-y-1 pl-5 text-sm">
          {s.facts.slice(0, 3).map((f) => (
            <li key={f}>{f}</li>
          ))}
        </ul>
      )}

      {player.answer_flags && player.answer_flags.length > 0 && (
        <div className="mx-5 mt-4 rounded-lg border border-no px-3 py-2 text-sm">
          <p className="font-semibold">
            {title.startsWith("You") ? "Your answers that contradict the data" : `${player.name}'s answers that contradict the data`}
          </p>
          <ul className="mt-1 space-y-1">
            {player.answer_flags.map((f) => (
              <li key={f.n}>
                “{f.text}” was answered {f.answered === "yes" ? "Yes" : "No"}, but for {s.name} the data says{" "}
                {f.data === "yes" ? "Yes" : "No"} (read as “{f.read_as}”).
              </li>
            ))}
          </ul>
        </div>
      )}

      {hunter.cracked_by && (
        <p className="mx-5 mt-4 rounded-lg bg-hi-soft px-3 py-2 text-sm">
          The question that cracked it: <strong>{hunter.cracked_by.text}</strong> It ruled out {hunter.cracked_by.removed}{" "}
          {hunter.cracked_by.removed === 1 ? "candidate" : "candidates"}.
        </p>
      )}

      <details className="mt-4 border-t border-rule-soft">
        <summary className="cursor-pointer px-5 py-3 font-semibold">{hunterName === "You" ? "Your" : `${hunterName}'s`} questions</summary>
        <div className="px-5 pb-5">
          <BallStrip log={hunter.log} />
          <ol className="mt-3 flex flex-col text-sm">
            {hunter.log.map((m) => (
              <li key={m.n} className="flex gap-3 border-b border-rule-soft py-1.5">
                <span className="w-5 shrink-0 text-right text-ink-soft tabular">{m.n}</span>
                <span className="grow">{m.text}</span>
                <span className="shrink-0 font-semibold">{describeAnswer(m)}</span>
              </li>
            ))}
          </ol>
        </div>
      </details>
    </article>
  );
}
