"use client";

import { useMemo, useState, type FormEvent } from "react";
import { packIdFrom } from "@/lib/api";
import type { MyPack, PackSummary } from "@/lib/protocol";

/** Packs grouped by category, in a stable order. */
export function groupPacks(packs: PackSummary[]): [string, PackSummary[]][] {
  const order = ["Sports", "Science", "Technology", "History", "Geography", "Economics", "Entertainment"];
  const groups = new Map<string, PackSummary[]>();
  for (const p of packs) {
    const list = groups.get(p.category) ?? [];
    list.push(p);
    groups.set(p.category, list);
  }
  return [...groups.entries()].sort((a, b) => {
    const ia = order.indexOf(a[0]);
    const ib = order.indexOf(b[0]);
    return (ia < 0 ? 99 : ia) - (ib < 0 ? 99 : ib) || a[0].localeCompare(b[0]);
  });
}

/**
 * Searchable, grouped pack chooser for the host. "Surprise me" picks a
 * random pack other than the current one.
 */
export function PackPicker({
  packs,
  value,
  onChange,
  mine = [],
  onPickCustom,
}: {
  packs: PackSummary[];
  value: string;
  onChange: (id: string) => void;
  /** The host's own player-made packs. */
  mine?: MyPack[];
  /** Loads a player-made pack by id, then selects it. Resolves to an error message or null. */
  onPickCustom?: (id: string) => Promise<string | null>;
}) {
  const [query, setQuery] = useState("");
  const [link, setLink] = useState("");
  const [linkError, setLinkError] = useState<string | null>(null);
  const shownMine = useMemo(() => {
    const q = query.trim().toLowerCase();
    return mine.filter((p) => !p.hidden && (!q || p.title.toLowerCase().includes(q)));
  }, [mine, query]);

  async function useLink(e: FormEvent) {
    e.preventDefault();
    const id = packIdFrom(link);
    if (!id) {
      setLinkError("Paste a pack link like …/pack/?id=u-abc…");
      return;
    }
    setLinkError((await onPickCustom?.(id)) ?? null);
    setLink("");
  }
  const groups = useMemo(() => {
    const q = query.trim().toLowerCase();
    const shown = q
      ? packs.filter((p) => `${p.title} ${p.category} ${p.description} ${p.noun}`.toLowerCase().includes(q))
      : packs;
    return groupPacks(shown);
  }, [packs, query]);

  function surprise() {
    const others = packs.filter((p) => p.id !== value);
    const pick = others[Math.floor(Math.random() * others.length)];
    if (pick) onChange(pick.id);
  }

  return (
    <div>
      <div className="mt-2 flex gap-2">
        <input
          className="field"
          type="search"
          placeholder={`Search ${packs.length} packs`}
          aria-label="Search topic packs"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <button type="button" className="btn shrink-0" onClick={surprise}>
          Surprise me
        </button>
      </div>
      <div className="mt-3 max-h-80 overflow-y-auto rounded-[10px] border border-rule-soft">
        {shownMine.length > 0 && (
          <div role="group" aria-label="Your packs">
            <p className="sticky top-0 bg-sheet px-3 pb-1 pt-2 text-sm font-semibold text-ink-soft">Your packs</p>
            <ul>
              {shownMine.map((p) => {
                const active = p.id === value;
                return (
                  <li key={p.id}>
                    <button
                      type="button"
                      aria-pressed={active}
                      onClick={() => onPickCustom?.(p.id)}
                      className={`flex w-full items-baseline justify-between gap-3 px-3 py-2 text-left ${
                        active ? "bg-ink text-sheet" : "hover:bg-rule-soft"
                      }`}
                    >
                      <span className="font-semibold">{p.title}</span>
                      <span className={`shrink-0 text-sm ${active ? "opacity-80" : "text-ink-soft"}`}>v{p.version}</span>
                    </button>
                  </li>
                );
              })}
            </ul>
          </div>
        )}
        {groups.map(([category, list]) => (
          <div key={category} role="group" aria-label={category}>
            <p className="sticky top-0 bg-sheet px-3 pb-1 pt-2 text-sm font-semibold text-ink-soft">{category}</p>
            <ul>
              {list.map((p) => {
                const active = p.id === value;
                return (
                  <li key={p.id}>
                    <button
                      type="button"
                      aria-pressed={active}
                      onClick={() => onChange(p.id)}
                      className={`flex w-full items-baseline justify-between gap-3 px-3 py-2 text-left ${
                        active ? "bg-ink text-sheet" : "hover:bg-rule-soft"
                      }`}
                    >
                      <span>
                        <span className="font-semibold">{p.title}</span>
                        <span className={`block text-sm ${active ? "opacity-80" : "text-ink-soft"}`}>{p.description}</span>
                      </span>
                      <span className={`shrink-0 text-sm tabular ${active ? "opacity-80" : "text-ink-soft"}`}>{p.candidates}</span>
                    </button>
                  </li>
                );
              })}
            </ul>
          </div>
        ))}
        {groups.length === 0 && shownMine.length === 0 && (
          <p className="px-3 py-4 text-sm text-ink-soft">No pack matches “{query}”.</p>
        )}
      </div>
      {onPickCustom && (
        <form onSubmit={useLink} className="mt-3">
          <label htmlFor="pack-link" className="text-sm font-semibold">
            Someone shared a pack?
          </label>
          <div className="mt-1 flex gap-2">
            <input
              id="pack-link"
              className="field text-sm"
              placeholder="Paste the pack's link"
              value={link}
              onChange={(e) => setLink(e.target.value)}
            />
            <button className="btn shrink-0">Use it</button>
          </div>
          {linkError && (
            <p role="alert" className="mt-1 text-sm text-no">
              {linkError}
            </p>
          )}
        </form>
      )}
    </div>
  );
}
