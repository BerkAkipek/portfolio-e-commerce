import Link from "next/link";

export default function CheckoutCancelPage() {
  return (
    <div className="relative min-h-screen overflow-hidden bg-[#070b14] text-slate-100">
      <div className="pointer-events-none absolute inset-0">
        <div className="absolute left-1/2 top-[-16rem] h-[36rem] w-[36rem] -translate-x-1/2 rounded-full bg-cyan-500/15 blur-3xl" />
        <div className="absolute bottom-[-12rem] right-[-8rem] h-[28rem] w-[28rem] rounded-full bg-emerald-500/10 blur-3xl" />
      </div>
      <main className="relative mx-auto flex min-h-[70vh] w-full max-w-6xl items-center justify-center px-6 sm:px-10 lg:px-12">
        <section className="w-full max-w-xl rounded-3xl border border-amber-300/30 bg-amber-400/10 p-8 text-center">
          <p className="text-xs font-semibold tracking-[0.18em] text-amber-100 uppercase">Checkout Canceled</p>
          <h1 className="mt-3 text-3xl font-semibold text-white">Your cart is still waiting</h1>
          <p className="mt-3 text-sm text-slate-200">
            No charge was made. You can return to cart and try again when ready.
          </p>
          <div className="mt-6 flex flex-wrap justify-center gap-3">
            <Link
              href="/cart"
              className="rounded-full bg-cyan-300 px-5 py-2.5 text-sm font-semibold text-[#03151b] transition hover:bg-cyan-200"
            >
              Back to Cart
            </Link>
            <Link
              href="/catalog"
              className="rounded-full border border-white/20 px-5 py-2.5 text-sm font-semibold text-slate-100 transition hover:border-cyan-300/80"
            >
              Continue Shopping
            </Link>
          </div>
        </section>
      </main>
    </div>
  );
}
