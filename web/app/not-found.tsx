import Link from "next/link";

export default function NotFound() {
  return (
    <main className="mx-auto max-w-xl px-4 pt-16 sm:px-6">
      <h1 className="text-3xl font-extrabold">Page not found</h1>
      <p className="mt-2 text-ink-soft">That address doesn&apos;t lead anywhere. Start a game from the home page.</p>
      <Link href="/" className="btn btn-ink mt-6">
        Go to the home page
      </Link>
    </main>
  );
}
