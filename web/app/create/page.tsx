"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { Suspense, useEffect, useState } from "react";
import { Header, NameForm } from "@/components/common";
import { PackBuilder } from "@/components/PackBuilder";
import { ApiError, getPack } from "@/lib/api";
import { useApp } from "@/lib/app";
import type { PackDraft } from "@/lib/protocol";

export default function CreatePage() {
  return (
    <Suspense fallback={<Header />}>
      <Create />
    </Suspense>
  );
}

function Create() {
  const id = useSearchParams().get("id") ?? "";
  const { guest, ready } = useApp();
  const [draft, setDraft] = useState<PackDraft | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!id || !guest) return;
    getPack(id, guest)
      .then((v) => {
        if (!v.is_owner || !v.edit_draft) setError("Only the person who made this pack can edit it.");
        else setDraft(v.edit_draft);
      })
      .catch((e) => setError(e instanceof ApiError && e.status === 404 ? "This pack doesn't exist any more." : "Couldn't load the pack."));
  }, [id, guest]);

  return (
    <>
      <Header>
        <Link href="/" className="underline underline-offset-4">
          Home
        </Link>
      </Header>
      <main className="mx-auto w-full max-w-6xl px-4 pt-4 sm:px-6 lg:pt-8">
        {!ready ? null : !guest ? (
          <div className="max-w-xl">
            <h1 className="text-4xl font-extrabold">Make a pack</h1>
            <p className="mb-5 mt-2 text-ink-soft">Choose a name first. It&apos;s shown as the pack&apos;s author.</p>
            <NameForm submitLabel="Continue" />
          </div>
        ) : error ? (
          <div role="alert">
            <h1 className="text-3xl font-extrabold">Can&apos;t edit this pack</h1>
            <p className="mt-2 text-ink-soft">{error}</p>
            <Link href="/create/" className="btn mt-4">
              Make a new pack
            </Link>
          </div>
        ) : id && !draft ? (
          <p className="text-ink-soft">Loading the pack…</p>
        ) : (
          <PackBuilder key={id || "new"} initial={draft ?? undefined} packId={id || undefined} />
        )}
      </main>
    </>
  );
}
