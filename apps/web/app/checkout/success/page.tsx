import Link from "next/link";

export default function CheckoutSuccessPage() {
  return (
    <div className="relative min-h-screen overflow-hidden bg-[#070b14] text-slate-100">
      <div className="pointer-events-none absolute inset-0">
        <div className="absolute left-1/2 top-[-16rem] h-[36rem] w-[36rem] -translate-x-1/2 rounded-full bg-cyan-500/15 blur-3xl" />
        <div className="absolute bottom-[-12rem] right-[-8rem] h-[28rem] w-[28rem] rounded-full bg-emerald-500/10 blur-3xl" />
      </div>
      <main className="relative mx-auto flex min-h-[70vh] w-full max-w-6xl items-center justify-center px-6 sm:px-10 lg:px-12">
        <section className="w-full max-w-xl rounded-3xl border border-emerald-300/30 bg-emerald-400/10 p-8 text-center">
          <p className="text-xs font-semibold tracking-[0.18em] text-emerald-100 uppercase">Payment Confirmed</p>
          <h1 className="mt-3 text-3xl font-semibold text-white">Thanks for your order</h1>
          <p className="mt-3 text-sm text-slate-200">
            Your payment was completed in Stripe Checkout. Orders are now processing.
          </p>
          <div className="mt-6 flex flex-wrap justify-center gap-3">
            <Link
              href="/orders"
              className="rounded-full bg-cyan-300 px-5 py-2.5 text-sm font-semibold text-[#03151b] transition hover:bg-cyan-200"
            >
              View My Orders
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
