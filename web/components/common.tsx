"use client";

import Link from "next/link";
import { useState, type FormEvent, type ReactNode } from "react";
import { useApp } from "@/lib/app";
import type { CandidateCard, GameError, Move } from "@/lib/protocol";

export function Wordmark({ size = "md" }: { size?: "md" | "lg" }) {
  const { appName } = useApp();
  return (
    <Link
      href="/"
      className={`inline-flex items-center gap-2 font-[family-name:var(--font-display)] font-extrabold tracking-tight ${
        size === "lg" ? "text-5xl sm:text-6xl" : "text-xl"
      }`}
    >
      <svg viewBox="0 0 64 64" aria-hidden className={size === "lg" ? "h-12 w-12 sm:h-14 sm:w-14" : "h-7 w-7"}>
        <rect width="64" height="64" rx="14" fill="var(--color-ink)" />
        <circle cx="27" cy="27" r="13" fill="none" stroke="var(--color-hi)" strokeWidth="6" />
        <path d="M37 37l13 13" stroke="var(--color-hi)" strokeWidth="7" strokeLinecap="round" />
      </svg>
      {appName}
    </Link>
  );
}

export function Header({ children }: { children?: ReactNode }) {
  const { guest } = useApp();
  return (
    <header className="mx-auto flex w-full max-w-6xl items-center justify-between gap-4 px-4 py-3 sm:px-6">
      <Wordmark />
      <div className="flex items-center gap-3 text-sm text-ink-soft">
        {children}
        {guest && <span className="hidden sm:inline">Playing as {guest.display_name}</span>}
      </div>
    </header>
  );
}

export function NameForm({
  submitLabel,
  onDone,
  initial = "",
}: {
  submitLabel: string;
  onDone?: () => void;
  initial?: string;
}) {
  const { signIn, guest, rename } = useApp();
  const [name, setName] = useState(initial);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function submit(e: FormEvent) {
    e.preventDefault();
    const n = name.trim();
    if (!n) {
      setError("Enter a name so your opponent knows who you are.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      if (guest) await rename(n);
      else await signIn(n);
      onDone?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Couldn't save your name.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form onSubmit={submit} className="flex flex-col gap-2 sm:flex-row">
      <label className="sr-only" htmlFor="display-name">
        Your name
      </label>
      <input
        id="display-name"
        className="field sm:max-w-xs"
        placeholder="Your name"
        maxLength={24}
        autoComplete="nickname"
        value={name}
        onChange={(e) => setName(e.target.value)}
      />
      <button className="btn btn-ink" disabled={busy}>
        {busy ? "Saving…" : submitLabel}
      </button>
      {error && (
        <p role="alert" className="text-sm text-no sm:self-center">
          {error}
        </p>
      )}
    </form>
  );
}

export function Toasts({ errors, onDismiss }: { errors: { id: number; error: GameError }[]; onDismiss: (id: number) => void }) {
  if (!errors.length) return null;
  return (
    <div className="pointer-events-none fixed inset-x-0 bottom-4 z-50 flex flex-col items-center gap-2 px-4" aria-live="assertive">
      {errors.map(({ id, error }) => (
        <button
          key={id}
          onClick={() => onDismiss(id)}
          className="pointer-events-auto max-w-md rounded-xl border border-no bg-sheet px-4 py-3 text-left text-sm shadow-[0_6px_0_-2px_var(--color-rule)]"
        >
          {error.message}
        </button>
      ))}
    </div>
  );
}

const ballLabel: Record<string, string> = {
  yes: "Y",
  no: "N",
  unknown: "?",
  wrong: "×",
  correct: "✓",
  timeout: "T",
  give_up: "—",
};

/** A move as a scorebook ball. */
export function Ball({ move }: { move: Move }) {
  const key = move.kind === "ask" || move.kind === "guess" ? (move.answer ?? "unknown") : move.kind;
  return (
    <span className={`ball ball-${key}`} title={`${move.text} — ${describeAnswer(move)}`}>
      {ballLabel[key] ?? "·"}
    </span>
  );
}

export function describeAnswer(m: Move): string {
  switch (m.kind) {
    case "ask":
      return m.answer === "yes" ? "Yes" : m.answer === "no" ? "No" : "Unknown";
    case "guess":
      return m.answer === "correct" ? "Correct" : `Wrong (+${m.cost} moves)`;
    case "timeout":
      return "+1 move";
    default:
      return "Gave up";
  }
}

export function BallStrip({ log, empty }: { log: Move[]; empty?: string }) {
  if (!log.length) return <p className="text-sm text-ink-soft">{empty ?? "No moves yet."}</p>;
  return (
    <ol className="flex flex-wrap gap-1.5" aria-label="Moves so far">
      {log.map((m) => (
        <li key={m.n}>
          <Ball move={m} />
        </li>
      ))}
    </ol>
  );
}

export function Monogram({ name, className = "" }: { name: string; className?: string }) {
  const initials = name
    .split(/[\s-]+/)
    .filter(Boolean)
    .map((w) => w[0])
    .join("")
    .slice(0, 2)
    .toUpperCase();
  return (
    <div
      aria-hidden
      className={`grid place-items-center rounded-full bg-ink font-[family-name:var(--font-display)] font-extrabold text-hi ${className}`}
    >
      {initials}
    </div>
  );
}

export function formatDuration(ms: number): string {
  const s = Math.round(ms / 1000);
  if (s < 60) return `${s}s`;
  return `${Math.floor(s / 60)}m ${String(s % 60).padStart(2, "0")}s`;
}

/**
 * The revealed candidate's picture with its credit, or initials when there is
 * no licensed image or the file fails to load. Only rendered on result
 * screens: the server sends image data nowhere else.
 */
export function Portrait({ card, children }: { card: CandidateCard; children?: ReactNode }) {
  const [broken, setBroken] = useState(false);
  const img = broken ? undefined : card.image;
  return (
    <>
      <div className="flex gap-4 p-5">
        {img ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={img.url}
            alt={`Photo of ${card.name}`}
            width={112}
            height={112}
            loading="lazy"
            onError={() => setBroken(true)}
            className="h-28 w-28 shrink-0 rounded-xl object-cover object-top"
            data-portrait
          />
        ) : (
          <Monogram name={card.name} className="h-28 w-28 shrink-0 text-4xl" />
        )}
        <div className="min-w-0">{children}</div>
      </div>
      {img && (
        <p className="px-5 text-xs leading-relaxed text-ink-soft break-words" data-credit>
          Photo: {img.credit} ·{" "}
          {img.license_url ? (
            <a href={img.license_url} className="underline" target="_blank" rel="noreferrer license">
              {img.license}
            </a>
          ) : (
            img.license
          )}{" "}
          ·{" "}
          <a href={img.source_url} className="underline" target="_blank" rel="noreferrer">
            Source
          </a>
        </p>
      )}
    </>
  );
}
