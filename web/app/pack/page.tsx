"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useState, type FormEvent } from "react";
import { Header, NameForm } from "@/components/common";
import { ApiError, createRoom, deletePack, getPack, reportPack } from "@/lib/api";
import { useApp } from "@/lib/app";
import type { CustomPackView } from "@/lib/protocol";

export default function PackPage() {
  return (
    <Suspense fallback={<Header />}>
      <PackLanding />
    </Suspense>
  );
}

function PackLanding() {
  const params = useSearchParams();
  const id = params.get("id") ?? "";
  const justSaved = params.get("saved") === "1";
  const { guest, ready } = useApp();
  const router = useRouter();
  const [pack, setPack] = useState<CustomPackView | null>(null);
  const [missing, setMissing] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!ready) return;
    if (!id) {
      setMissing("This link has no pack in it.");
      return;
    }
    getPack(id, guest)
      .then(setPack)
      .catch((e) =>
        setMissing(
          e instanceof ApiError && e.status === 404
            ? "This pack doesn't exist, was deleted, or is hidden after reports."
            : "Couldn't load the pack. Check your connection and try again.",
        ),
      );
  }, [id, guest, ready]);

  async function play(practice: boolean) {
    if (!guest || !pack) return;
    setBusy(practice ? "practice" : "friend");
    setError(null);
    try {
      const r = await createRoom(guest, practice, pack.id);
      router.push(`/room/?code=${r.code}`);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Couldn't create a room.");
      setBusy(null);
    }
  }

  async function copy() {
    if (!pack) return;
    const url = `${window.location.origin}/pack/?id=${pack.id}`;
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      window.prompt("Copy this link", url);
    }
  }

  async function remove() {
    if (!guest || !pack || !window.confirm(`Delete “${pack.title}”? Its link stops working. This can't be undone.`)) return;
    setBusy("delete");
    try {
      await deletePack(guest, pack.id);
      router.push("/create/");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Couldn't delete the pack.");
      setBusy(null);
    }
  }

  return (
    <>
      <Header>
        <Link href="/" className="underline underline-offset-4">
          Home
        </Link>
      </Header>
      <main className="mx-auto w-full max-w-5xl px-4 pb-16 pt-4 sm:px-6 lg:pt-8">
        {missing ? (
          <div role="alert">
            <h1 className="text-3xl font-extrabold">Pack not found</h1>
            <p className="mt-2 text-ink-soft">{missing}</p>
            <div className="mt-4 flex flex-wrap gap-2">
              <Link href="/" className="btn btn-ink">
                Play a built-in pack
              </Link>
              <Link href="/create/" className="btn">
                Make your own
              </Link>
            </div>
          </div>
        ) : !pack ? (
          <p className="text-ink-soft">Loading…</p>
        ) : (
          <div className="grid gap-8 lg:grid-cols-[minmax(0,1fr)_22rem] lg:gap-12">
            <section className="min-w-0">
              {justSaved && pack.is_owner && (
                <p role="status" className="mb-4 rounded-[10px] bg-hi-soft px-4 py-3">
                  Saved. Share the link below; anyone who opens it can play.
                </p>
              )}
              {pack.hidden && (
                <p role="status" className="mb-4 rounded-[10px] border border-no px-4 py-3 text-no">
                  This pack is hidden because players reported it. Only you can see it until it is reviewed. Edit it, or
                  delete it.
                </p>
              )}
              <p className="text-sm font-semibold text-ink-soft">Player-made pack by {pack.author}</p>
              <h1 className="mt-1 text-4xl font-extrabold sm:text-5xl">{pack.title}</h1>
              {pack.description && <p className="mt-3 max-w-[60ch] text-lg text-ink-soft">{pack.description}</p>}
              <p className="mt-3 text-sm text-ink-soft tabular">
                {pack.candidates} {pack.noun}s · version {pack.version}
              </p>

              <h2 className="mt-8 text-lg font-bold">You can ask about</h2>
              <ul className="mt-2 flex flex-wrap gap-2">
                {[...pack.attributes, "First letters of the name"].map((a) => (
                  <li key={a} className="chip">
                    {a}
                  </li>
                ))}
              </ul>

              <h2 className="mt-8 text-lg font-bold">
                The {pack.noun}s <span className="font-normal text-ink-soft tabular">({pack.items.length})</span>
              </h2>
              <p className="mt-2 leading-relaxed text-ink-soft">{pack.items.join(" · ")}</p>
            </section>

            <aside className="flex flex-col gap-4">
              <div className="panel p-5">
                {!guest ? (
                  <>
                    <h2 className="text-xl font-bold">What should we call you?</h2>
                    <p className="mb-4 mt-1 text-sm text-ink-soft">No account needed.</p>
                    <NameForm submitLabel="Continue" />
                  </>
                ) : (
                  <>
                    <h2 className="text-xl font-bold">Play this pack</h2>
                    <div className="mt-4 flex flex-col gap-3">
                      <button className="btn btn-ink h-12" disabled={busy !== null || pack.hidden} onClick={() => play(false)}>
                        {busy === "friend" ? "Creating room…" : "Play with a friend"}
                      </button>
                      <button className="btn h-11" disabled={busy !== null || pack.hidden} onClick={() => play(true)}>
                        {busy === "practice" ? "Setting up…" : "Practice against the bot"}
                      </button>
                      <button className="btn btn-quiet" onClick={copy}>
                        {copied ? "Link copied" : "Copy the pack's link"}
                      </button>
                    </div>
                  </>
                )}
                {error && (
                  <p role="alert" className="mt-3 text-sm text-no">
                    {error}
                  </p>
                )}
              </div>

              {pack.is_owner ? (
                <div className="panel p-5">
                  <h2 className="font-bold">Your pack</h2>
                  {pack.reports ? (
                    <p className="mt-1 text-sm text-ink-soft">
                      Reported by {pack.reports} {pack.reports === 1 ? "player" : "players"}.
                    </p>
                  ) : null}
                  <div className="mt-3 flex flex-wrap gap-2">
                    <Link href={`/create/?id=${pack.id}`} className="btn">
                      Edit
                    </Link>
                    <button className="btn btn-quiet text-no" onClick={remove} disabled={busy !== null}>
                      Delete
                    </button>
                  </div>
                </div>
              ) : (
                guest && <ReportForm id={pack.id} />
              )}
            </aside>
          </div>
        )}
      </main>
    </>
  );
}

function ReportForm({ id }: { id: string }) {
  const { guest } = useApp();
  const [reason, setReason] = useState("");
  const [state, setState] = useState<"idle" | "sending" | "done" | "error">("idle");

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (!guest || !reason.trim()) return;
    setState("sending");
    try {
      await reportPack(guest, id, reason.trim());
      setState("done");
    } catch {
      setState("error");
    }
  }

  if (state === "done") {
    return <p className="text-sm text-ink-soft">Thanks. We&apos;ll review this pack.</p>;
  }
  return (
    <details className="text-sm">
      <summary className="cursor-pointer text-ink-soft">Report this pack</summary>
      <form onSubmit={submit} className="mt-3 flex flex-col gap-2">
        <label htmlFor="report-reason" className="font-semibold">
          What&apos;s wrong with it?
        </label>
        <textarea
          id="report-reason"
          className="field min-h-20 py-2"
          maxLength={300}
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder="Hurtful, private information, spam…"
        />
        <button className="btn self-start" disabled={!reason.trim() || state === "sending"}>
          {state === "sending" ? "Sending…" : "Send report"}
        </button>
        {state === "error" && <p className="text-no">Couldn&apos;t send the report. Try again.</p>}
      </form>
    </details>
  );
}
