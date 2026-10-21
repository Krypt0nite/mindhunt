"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { Suspense } from "react";
import { Header, NameForm, Toasts } from "@/components/common";
import { Hunt } from "@/components/Hunt";
import { Lobby } from "@/components/Lobby";
import { Results } from "@/components/Results";
import { SecretPicker } from "@/components/SecretPicker";
import { useApp } from "@/lib/app";
import { useRoom } from "@/lib/useRoom";

export default function RoomPage() {
  return (
    <Suspense fallback={<Header />}>
      <RoomGate />
    </Suspense>
  );
}

function RoomGate() {
  const params = useSearchParams();
  const code = (params.get("code") ?? "").toUpperCase();
  const { guest, ready, serverError } = useApp();

  if (!code) return <Notice title="No room code" body="Open the invite link again, or enter the code on the home page." />;
  if (!ready) return <Header />;
  if (serverError) return <Notice title="Can't reach the game server" body={serverError} />;
  if (!guest) {
    return (
      <>
        <Header />
        <main className="mx-auto max-w-xl px-4 pt-10 sm:px-6">
          <h1 className="text-3xl font-extrabold">You&apos;re invited to room {code}</h1>
          <p className="mb-5 mt-2 text-ink-soft">Choose a name to join. No account needed.</p>
          <NameForm submitLabel="Join room" />
        </main>
      </>
    );
  }
  return <RoomScreen code={code} token={guest.token} />;
}

function RoomScreen({ code, token }: { code: string; token?: string }) {
  const room = useRoom(code, token);
  const { state, pool, status, closedReason } = room;

  if (closedReason) return <Notice title="Can't open this room" body={closedReason} />;

  const conn =
    status === "reconnecting" ? (
      <span role="status" className="rounded-full bg-hi-soft px-3 py-1 text-ink">
        Reconnecting…
      </span>
    ) : null;

  if (!state) {
    return (
      <>
        <Header>{conn}</Header>
        <main className="mx-auto max-w-6xl px-4 pt-10 text-ink-soft sm:px-6">Joining room {code}…</main>
      </>
    );
  }

  const needsPool = state.phase === "picking" || state.phase === "hunting";
  return (
    <>
      <Header>{conn}</Header>
      <main className="mx-auto w-full max-w-6xl px-4 pb-24 sm:px-6">
        {state.phase === "lobby" && <Lobby state={state} send={room.send} />}
        {needsPool && !pool && <p className="pt-10 text-ink-soft">Loading the candidates…</p>}
        {state.phase === "picking" && pool && <SecretPicker state={state} pool={pool} send={room.send} />}
        {state.phase === "hunting" && pool && (
          <Hunt state={state} pool={pool} send={room.send} clockSkew={room.clockSkew} lastResult={room.lastResult} />
        )}
        {state.phase === "results" && state.result && <Results state={state} send={room.send} />}
        {state.phase === "void" && (
          <section className="mx-auto max-w-xl pt-10">
            <h1 className="text-3xl font-extrabold">Round cancelled</h1>
            <p className="mt-2 text-ink-soft">Both players were away for more than 10 minutes, so this round doesn&apos;t count.</p>
            <button className="btn btn-ink mt-6" onClick={() => room.send("to_lobby")}>
              Back to the lobby
            </button>
          </section>
        )}
      </main>
      <Toasts errors={room.errors} onDismiss={room.dismissError} />
    </>
  );
}

function Notice({ title, body }: { title: string; body: string }) {
  return (
    <>
      <Header />
      <main className="mx-auto max-w-xl px-4 pt-10 sm:px-6">
        <h1 className="text-3xl font-extrabold">{title}</h1>
        <p className="mt-2 text-ink-soft">{body}</p>
        <Link href="/" className="btn btn-ink mt-6">
          Go to the home page
        </Link>
      </main>
    </>
  );
}
