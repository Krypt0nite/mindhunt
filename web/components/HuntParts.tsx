"use client";

import { useState } from "react";
import type { Move, PoolCandidate, PoolView } from "@/lib/protocol";
import { describeAnswer } from "./common";

export interface Flash {
  text: string;
  tone: "yes" | "no" | "unknown" | "wrong" | "correct";
  detail: string;
}

export function FlashLine({ flash, remaining }: { flash: Flash; remaining: number }) {
  const color = {
    yes: "border-yes",
    no: "border-no",
    unknown: "border-unk",
    wrong: "border-no",
    correct: "border-hi",
  }[flash.tone];
  const word = { yes: "Yes", no: "No", unknown: "Unknown", wrong: "Wrong", correct: "Correct" }[flash.tone];
  return (
    <div role="status" className={`panel border-l-[6px] ${color} p-4`}>
      <p className="text-sm text-ink-soft">{flash.text}</p>
      <p className="mt-0.5 font-[family-name:var(--font-display)] text-3xl font-extrabold">{word}</p>
      <p className="text-sm text-ink-soft">
        {flash.detail || `${remaining} ${remaining === 1 ? "candidate is" : "candidates are"} still possible.`}
      </p>
    </div>
  );
}


export function History({ log, opponent }: { log: Move[]; opponent: string }) {
  if (!log.length) return null;
  return (
    <div>
      <h2 className="text-lg font-bold">Your questions</h2>
      <ol className="mt-2 flex flex-col">
        {[...log].reverse().map((m) => (
          <li key={m.n} className="flex items-baseline gap-3 border-b border-rule-soft py-2">
            <span className="w-6 shrink-0 text-right text-sm text-ink-soft tabular">{m.n}</span>
            <span className="grow">
              {m.text}
              {m.manual && <span className="block text-xs text-ink-soft">Answered by {opponent}</span>}
            </span>
            <span
              className={`shrink-0 text-sm font-semibold ${
                m.answer === "yes" || m.answer === "correct" ? "text-yes" : m.answer === "no" || m.answer === "wrong" ? "text-no" : "text-ink-soft"
              }`}
            >
              {describeAnswer(m)}
            </span>
          </li>
        ))}
      </ol>
    </div>
  );
}


export function Board({
  pool,
  remaining,
  guessed,
  selected,
  onSelect,
}: {
  pool: PoolView;
  remaining: Set<string>;
  guessed: Set<string>;
  selected: string | null;
  onSelect: (id: string | null) => void;
}) {
  const [hideOut, setHideOut] = useState(false);
  const list = hideOut ? pool.candidates.filter((c) => remaining.has(c.id)) : pool.candidates;
  return (
    <div>
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h2 className="text-lg font-bold">
          <span className="rounded bg-hi-soft px-1.5 tabular">{remaining.size}</span> of {pool.candidates.length} still possible
        </h2>
        <label className="flex items-center gap-2 text-sm text-ink-soft">
          <input type="checkbox" checked={hideOut} onChange={(e) => setHideOut(e.target.checked)} className="h-4 w-4 accent-[var(--color-ink)]" />
          Hide ruled-out names
        </label>
      </div>
      <p className="mt-1 text-sm text-ink-soft">The board updates from your answers. Unknown answers never rule anyone out.</p>
      <ul className="mt-3 grid grid-cols-2 gap-1.5 sm:grid-cols-3" aria-label="Candidates">
        {list.map((c) => (
          <li key={c.id}>
            <button
              className="tile w-full"
              data-live={remaining.has(c.id)}
              aria-pressed={selected === c.id}
              style={selected === c.id ? { outline: "2px solid var(--color-ink)", outlineOffset: "1px" } : undefined}
              onClick={() => onSelect(selected === c.id ? null : c.id)}
            >
              <span className="tile-name">{c.name}</span>
              {guessed.has(c.id) && <span className="sr-only"> (already guessed)</span>}
              <span className="sr-only">{remaining.has(c.id) ? ", still possible" : ", ruled out"}</span>
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}


export function CandidateDetail({
  c,
  pool,
  live,
  canGuess,
  busy,
  cost,
  onGuess,
  onClose,
}: {
  c: PoolCandidate;
  pool: PoolView;
  live: boolean;
  canGuess: boolean;
  busy: boolean;
  cost: number;
  onGuess: () => void;
  onClose: () => void;
}) {
  return (
    <div className="panel sticky bottom-3 mt-4 p-4 shadow-[0_8px_0_-3px_var(--color-rule)] sm:p-5" role="dialog" aria-label={c.name}>
      <div className="flex items-start justify-between gap-3">
        <div>
          <h3 className="text-xl font-bold">{c.name}</h3>
          <p className={`text-sm ${live ? "text-ink" : "text-ink-soft"}`}>{live ? "Still possible" : "Ruled out by your answers"}</p>
        </div>
        <button className="btn btn-quiet" onClick={onClose}>
          Close
        </button>
      </div>
      <dl className="mt-3 grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
        {pool.attributes.map((a) =>
          c.attrs[a.key] ? (
            <div key={a.key} className="contents">
              <dt className="text-ink-soft">{a.label}</dt>
              <dd>{c.attrs[a.key]}</dd>
            </div>
          ) : null,
        )}
      </dl>
      {canGuess && (
        <button className="btn btn-ink mt-4 w-full sm:w-auto" disabled={busy} onClick={onGuess}>
          Guess {c.name}
          {!live && " (ruled out!)"}
        </button>
      )}
      {canGuess && <p className="mt-2 text-xs text-ink-soft">Correct ends your hunt. Wrong adds {cost} moves.</p>}
    </div>
  );
}
