"use client";

import { useEffect, useRef, useState, type FormEvent } from "react";
import type { MatchView, Pending, Predicate } from "@/lib/protocol";
import type { Send } from "@/lib/useRoom";
import { useSpeech } from "@/lib/useSpeech";

/**
 * Type or say a question. Known questions are answered from the data; with
 * free text on, anything else goes to the opponent to answer by hand.
 * Nothing is sent without the player pressing a button.
 */
export function TextAsk({
  preview,
  submitText,
  cancelPending,
  freeText,
  opponent,
  pending,
  busy,
  onAskPredicate,
}: {
  /** Which question does this text mean? Costs nothing. */
  preview: (text: string) => Promise<MatchView | null>;
  /** Ask the text; resolves true when it was accepted. */
  submitText: (text: string) => Promise<boolean>;
  cancelPending?: () => void;
  freeText: boolean;
  opponent: string;
  pending?: Pending;
  busy: boolean;
  onAskPredicate: (p: Predicate) => void;
}) {
  const [text, setText] = useState("");
  const [match, setMatch] = useState<MatchView | null>(null);
  const [checking, setChecking] = useState(false);
  const [sending, setSending] = useState(false);
  const [heard, setHeard] = useState(false);
  const seq = useRef(0);
  const previewRef = useRef(preview); // callers may pass a new function each render
  previewRef.current = preview;
  const speech = useSpeech((t) => {
    setText(t);
    setHeard(true);
  });

  // Ask the server what the text means (debounced). Costs nothing.
  useEffect(() => {
    const t = text.trim();
    if (t.length < 3) {
      setMatch(null);
      return;
    }
    const mine = ++seq.current;
    setChecking(true);
    const h = setTimeout(async () => {
      const m = await previewRef.current(t);
      if (mine !== seq.current) return;
      setChecking(false);
      setMatch(m);
    }, 450);
    return () => clearTimeout(h);
  }, [text]);

  async function sendText(e?: FormEvent) {
    e?.preventDefault();
    const t = text.trim();
    if (t.length < 3) return;
    setSending(true);
    const ok = await submitText(t);
    setSending(false);
    if (ok) {
      setText("");
      setMatch(null);
      setHeard(false);
    }
  }

  function askSuggested() {
    if (!match?.predicate) return;
    onAskPredicate(match.predicate);
    setText("");
    setMatch(null);
    setHeard(false);
  }

  if (pending) {
    return (
      <div role="status">
        <p className="text-sm text-ink-soft">Waiting for {opponent} to answer</p>
        <p className="mt-1 text-lg font-semibold">“{pending.text}”</p>
        <p className="mt-1 text-sm text-ink-soft">Your move timer is paused. The move is counted when they answer.</p>
        <button className="btn mt-3" onClick={() => cancelPending?.()}>
          Cancel this question
        </button>
      </div>
    );
  }

  const t = text.trim();
  const ready = t.length >= 3 && !checking;
  const exact = ready && match?.exact && match.predicate;
  const suggestion = ready && match && !match.exact && match.predicate;

  return (
    <form onSubmit={(e) => (exact || (freeText && !suggestion) ? sendText(e) : e.preventDefault())}>
      <label htmlFor="text-question" className="font-semibold">
        Type{speech.supported ? " or say" : ""} a question
      </label>
      <div className="mt-2 flex gap-2">
        <input
          id="text-question"
          className="field"
          maxLength={140}
          autoComplete="off"
          placeholder="Did they ever play for India?"
          value={text}
          onChange={(e) => {
            setText(e.target.value);
            setHeard(false);
          }}
        />
        {speech.supported && (
          <button
            type="button"
            className={`btn shrink-0 px-3 ${speech.listening ? "btn-ink" : ""}`}
            aria-pressed={speech.listening}
            aria-label={speech.listening ? "Stop listening" : "Ask by voice"}
            title={speech.listening ? "Stop listening" : "Ask by voice"}
            onClick={() => (speech.listening ? speech.stop() : speech.start())}
          >
            <MicIcon />
            <span className="sr-only">{speech.listening ? "Listening" : "Voice"}</span>
          </button>
        )}
      </div>
      {speech.listening && (
        <p className="mt-2 text-sm font-semibold" role="status">
          Listening… speak your question, then check it before sending.
        </p>
      )}
      {speech.error && (
        <p className="mt-2 text-sm text-no" role="alert">
          {speech.error}
        </p>
      )}
      {heard && !speech.listening && t && (
        <p className="mt-2 text-sm text-ink-soft">Check the words above. Nothing is sent until you press a button.</p>
      )}

      <div className="mt-3 min-h-[3.25rem]" aria-live="polite">
        {checking && t.length >= 3 && <p className="text-sm text-ink-soft">Checking…</p>}
        {exact && (
          <div className="flex flex-wrap items-center gap-3">
            <button className="btn btn-ink" disabled={busy || sending}>
              Ask (1 move)
            </button>
            <span className="text-sm text-ink-soft">Answered from the data as “{match!.text}”</span>
          </div>
        )}
        {suggestion && (
          <div className="flex flex-col gap-2">
            <p className="text-sm">
              Did you mean <strong>“{match!.text}”</strong>? That one is answered from the data.
            </p>
            <div className="flex flex-wrap gap-2">
              <button type="button" className="btn btn-ink" disabled={busy} onClick={askSuggested}>
                Ask that (1 move)
              </button>
              {freeText && (
                <button type="button" className="btn" disabled={sending} onClick={() => sendText()}>
                  Send my words to {opponent}
                </button>
              )}
            </div>
          </div>
        )}
        {ready && !match?.predicate && (
          freeText ? (
            <div className="flex flex-wrap items-center gap-3">
              <button className="btn btn-ink" disabled={sending}>
                Send to {opponent}
              </button>
              <span className="text-sm text-ink-soft">The game can&apos;t answer this, so {opponent} will. It counts as a move once answered.</span>
            </div>
          ) : (
            <p className="text-sm text-ink-soft">
              The game can&apos;t answer that one. Try other words, or pick a question above. The host can turn on typed questions in the lobby.
            </p>
          )
        )}
      </div>
    </form>
  );
}

function MicIcon() {
  return (
    <svg viewBox="0 0 24 24" width="20" height="20" aria-hidden fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
      <rect x="9" y="3" width="6" height="11" rx="3" />
      <path d="M5 11a7 7 0 0 0 14 0M12 18v3" />
    </svg>
  );
}

/** The opponent's typed question, which this player must answer by hand. */
export function IncomingQuestion({
  incoming,
  opponent,
  unknownsLeft,
  send,
}: {
  incoming: Pending;
  opponent: string;
  unknownsLeft: number;
  send: Send;
}) {
  const [busy, setBusy] = useState(false);
  async function answer(a: "yes" | "no" | "unknown") {
    setBusy(true);
    await send("answer_text", { id: incoming.id, answer: a });
    setBusy(false);
  }
  return (
    <section className="panel border-l-[6px] border-l-hi p-4 sm:p-5" aria-label={`${opponent} asks you a question`}>
      <p className="text-sm text-ink-soft">{opponent} asks about your secret</p>
      <p className="mt-1 font-[family-name:var(--font-display)] text-2xl font-bold">“{incoming.text}”</p>
      <div className="mt-3 flex flex-wrap gap-2">
        <button className="btn btn-ink min-w-20" disabled={busy} onClick={() => answer("yes")}>
          Yes
        </button>
        <button className="btn btn-ink min-w-20" disabled={busy} onClick={() => answer("no")}>
          No
        </button>
        <button className="btn" disabled={busy || unknownsLeft === 0} onClick={() => answer("unknown")}>
          Not sure ({unknownsLeft} left)
        </button>
      </div>
      <p className="mt-2 text-sm text-ink-soft">
        Answer honestly. Answers are checked against the data after the round, and two wrong ones cost you the win.
      </p>
    </section>
  );
}
