"use client";

import { useMemo, useState } from "react";
import type { PoolView, StateView } from "@/lib/protocol";
import type { Send } from "@/lib/useRoom";

export function SecretPicker({ state, pool, send }: { state: StateView; pool: PoolView; send: Send }) {
  const [query, setQuery] = useState("");
  const [choice, setChoice] = useState<string | null>(null);
  const mine = state.hunt?.your_secret_id;
  const opp = state.opponent;
  const summaryAttrs = pool.attributes.filter((a) => a.type !== "string").slice(0, 3);

  const shown = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return pool.candidates;
    return pool.candidates.filter(
      (c) => c.name.toLowerCase().includes(q) || Object.values(c.attrs).some((v) => v.toLowerCase().includes(q)),
    );
  }, [pool.candidates, query]);

  const selected = choice ?? mine ?? null;
  const selectedName = pool.candidates.find((c) => c.id === selected)?.name;
  const mineName = pool.candidates.find((c) => c.id === mine)?.name;

  function lock(id: string) {
    send("pick_secret", { candidate_id: id });
    setChoice(null);
  }

  function random() {
    const c = pool.candidates[Math.floor(Math.random() * pool.candidates.length)];
    if (c) setChoice(c.id);
  }

  return (
    <div className="pt-4 lg:pt-8">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="text-4xl font-extrabold">Pick your secret</h1>
          <p className="mt-1 max-w-[60ch] text-ink-soft">
            {opp?.name ?? "Your opponent"} will try to find it. Only you can see your pick. You can change it until both of you have
            locked in.
          </p>
        </div>
        <p className="text-sm text-ink-soft" role="status">
          {opp?.bot
            ? "The bot has picked."
            : opp?.status === "picked"
              ? `${opp.name} has picked.`
              : `${opp?.name ?? "Opponent"} is choosing…`}
        </p>
      </div>

      <div className="sticky top-0 z-10 -mx-4 mt-5 border-b border-rule bg-paper/95 px-4 py-3 backdrop-blur sm:-mx-6 sm:px-6">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
          <input
            className="field sm:max-w-sm"
            type="search"
            placeholder={`Search ${pool.candidates.length} candidates`}
            aria-label="Search candidates"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          <button className="btn" onClick={random}>
            Choose at random
          </button>
          <div className="flex items-center gap-3 sm:ml-auto">
            {mine && !choice ? (
              <p>
                Locked in: <strong>{mineName}</strong>
              </p>
            ) : (
              <button className="btn btn-ink" disabled={!selected || selected === mine} onClick={() => selected && lock(selected)}>
                {selectedName ? `Lock in ${selectedName}` : "Select a candidate"}
              </button>
            )}
          </div>
        </div>
      </div>

      <ul className="mt-4 grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3" aria-label="Candidates">
        {shown.map((c) => {
          const active = c.id === selected;
          return (
            <li key={c.id}>
              <button
                className={`w-full rounded-[var(--radius-tile)] border px-3 py-2.5 text-left ${
                  active ? "border-ink bg-ink text-sheet" : "border-rule-soft bg-sheet hover:border-rule"
                }`}
                aria-pressed={active}
                onClick={() => setChoice(c.id)}
              >
                <span className="block font-semibold">
                  {c.name}
                  {c.id === mine && <span className="font-normal opacity-80"> (your secret)</span>}
                </span>
                <span className={`block text-sm ${active ? "opacity-80" : "text-ink-soft"}`}>
                  {summaryAttrs.map((a) => c.attrs[a.key]).filter(Boolean).join(", ")}
                </span>
              </button>
            </li>
          );
        })}
      </ul>
      {shown.length === 0 && <p className="mt-6 text-ink-soft">No candidate matches “{query}”. Try a shorter search.</p>}
    </div>
  );
}
