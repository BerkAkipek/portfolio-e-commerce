import Link from "next/link";

export default function NotFound() {
  return (
    <main className="relative mx-auto flex min-h-[60vh] w-full max-w-3xl flex-col items-center justify-center px-6 py-16 text-center sm:px-10">
      <p className="text-xs font-semibold tracking-[0.2em] text-cyan-200 uppercase">
        404
      </p>
      <h1 className="mt-3 text-3xl font-semibold text-white sm:text-4xl">
        We couldn&apos;t find that page.
      </h1>
      <p className="mt-3 max-w-xl text-sm text-slate-300 sm:text-base">
        The link may be outdated, or the page may have moved.
      </p>
      <div className="mt-6 flex flex-wrap items-center justify-center gap-3">
        <Link
          href="/catalog"
          className="rounded-full bg-cyan-300 px-6 py-3 text-sm font-semibold tracking-wide text-[#03151b] transition hover:bg-cyan-200"
        >
          Browse catalog
        </Link>
        <Link
          href="/"
          className="rounded-full border border-white/25 px-6 py-3 text-sm font-semibold tracking-wide text-slate-100 transition hover:border-cyan-300/80 hover:text-white"
        >
          Back home
        </Link>
      </div>
    </main>
  );
}
