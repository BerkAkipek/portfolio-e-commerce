"use client";

import Link from "next/link";

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <html lang="en" className="dark">
      <body className="bg-[#070b14] text-slate-100 antialiased">
        <main className="relative mx-auto flex min-h-screen w-full max-w-3xl flex-col items-center justify-center px-6 py-16 text-center sm:px-10">
          <p className="text-xs font-semibold tracking-[0.2em] text-cyan-200 uppercase">
            Application error
          </p>
          <h1 className="mt-3 text-3xl font-semibold text-white sm:text-4xl">
            NovaCart encountered a critical error.
          </h1>
          <p className="mt-3 max-w-xl text-sm text-slate-300 sm:text-base">
            Please refresh or retry. If this keeps happening, use the catalog link to continue.
          </p>
          <div className="mt-6 flex flex-wrap items-center justify-center gap-3">
            <button
              type="button"
              onClick={reset}
              className="rounded-full bg-cyan-300 px-6 py-3 text-sm font-semibold tracking-wide text-[#03151b] transition hover:bg-cyan-200"
            >
              Retry app
            </button>
            <Link
              href="/catalog"
              className="rounded-full border border-white/25 px-6 py-3 text-sm font-semibold tracking-wide text-slate-100 transition hover:border-cyan-300/80 hover:text-white"
            >
              Open catalog
            </Link>
          </div>
          {process.env.NODE_ENV !== "production" ? (
            <p className="mt-5 max-w-xl text-xs text-red-200/90">{error.message}</p>
          ) : null}
        </main>
      </body>
    </html>
  );
}
