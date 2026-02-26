import Link from "next/link";

const featuredProducts = [
  {
    name: "Aero Running Jacket",
    category: "Outerwear",
    price: "$129",
    accent: "from-cyan-500/25 to-cyan-300/5",
  },
  {
    name: "Core Training Set",
    category: "Performance",
    price: "$89",
    accent: "from-emerald-500/25 to-emerald-300/5",
  },
  {
    name: "Urban Carry Pack",
    category: "Accessories",
    price: "$74",
    accent: "from-sky-500/25 to-sky-300/5",
  },
  {
    name: "Flex Everyday Tee",
    category: "Essentials",
    price: "$32",
    accent: "from-teal-500/25 to-teal-300/5",
  },
];

const collections = ["New Arrivals", "Best Sellers", "Men", "Women", "Accessories"];

const perks = [
  "Free shipping over $80",
  "30-day easy returns",
  "Secure Stripe checkout",
];

export default function Home() {
  return (
    <div className="relative min-h-screen overflow-hidden bg-[#070b14] text-slate-100">
      <div className="pointer-events-none absolute inset-0">
        <div className="absolute left-1/2 top-[-16rem] h-[36rem] w-[36rem] -translate-x-1/2 rounded-full bg-cyan-500/15 blur-3xl" />
        <div className="absolute bottom-[-12rem] right-[-8rem] h-[28rem] w-[28rem] rounded-full bg-emerald-500/10 blur-3xl" />
        <div className="absolute left-[-8rem] top-1/3 h-[22rem] w-[22rem] rounded-full bg-sky-500/10 blur-3xl" />
      </div>

      <main className="relative mx-auto flex w-full max-w-6xl flex-col px-6 pb-14 pt-8 sm:px-10 lg:px-12">
        <header className="mb-16 flex flex-wrap items-center justify-between gap-4 rounded-full border border-white/15 bg-white/5 px-5 py-3 backdrop-blur">
          <p className="text-sm font-medium tracking-[0.24em] text-slate-300 uppercase">NovaCart</p>
          <nav className="flex flex-wrap items-center gap-2 text-xs sm:text-sm">
            {collections.map((item) => (
              <Link
                key={item}
                href="/catalog"
                className="rounded-full border border-white/10 px-3 py-1.5 text-slate-300 transition hover:border-cyan-300/60 hover:text-white"
              >
                {item}
              </Link>
            ))}
            <Link
              href="/cart"
              className="rounded-full border border-cyan-300/35 px-3 py-1.5 text-slate-200 transition hover:border-cyan-300/70 hover:text-white"
            >
              Cart
            </Link>
            <Link
              href="/auth"
              className="rounded-full border border-white/10 px-3 py-1.5 text-slate-300 transition hover:border-cyan-300/60 hover:text-white"
            >
              Account
            </Link>
          </nav>
        </header>

        <section className="grid items-end gap-10 lg:grid-cols-[1.2fr_0.8fr]">
          <div className="animate-fade-in-up" style={{ animationDelay: "80ms" }}>
            <p className="mb-5 inline-flex rounded-full border border-emerald-300/40 bg-emerald-400/10 px-3 py-1 text-xs font-semibold tracking-[0.18em] text-emerald-100 uppercase">
              Spring Drop 2026
            </p>
            <h1 className="max-w-3xl text-4xl leading-tight font-semibold text-white sm:text-5xl lg:text-6xl">
              Premium essentials built for movement, styled for every day.
            </h1>
            <p className="mt-6 max-w-2xl text-base leading-7 text-slate-300 sm:text-lg">
              Discover performance-ready layers, versatile basics, and everyday gear
              crafted for comfort, quality, and clean design.
            </p>
            <div className="mt-9 flex flex-wrap gap-4">
              <Link
                href="/catalog"
                className="rounded-full bg-cyan-300 px-6 py-3 text-sm font-semibold tracking-wide text-[#03151b] transition hover:bg-cyan-200"
              >
                Shop New Arrivals
              </Link>
              <Link
                href="/catalog"
                className="rounded-full border border-slate-500/60 px-6 py-3 text-sm font-semibold tracking-wide text-slate-100 transition hover:border-slate-300 hover:bg-white/10"
              >
                Browse Best Sellers
              </Link>
            </div>
          </div>

          <aside
            className="animate-fade-in-up rounded-3xl border border-white/15 bg-white/5 p-6 backdrop-blur"
            style={{ animationDelay: "220ms" }}
          >
            <p className="text-xs font-semibold tracking-[0.18em] text-slate-300 uppercase">Why shop with us</p>
            <div className="mt-4 space-y-3">
              {perks.map((perk) => (
                <div key={perk} className="rounded-2xl border border-white/10 bg-black/20 px-4 py-3 text-sm text-slate-200">
                  {perk}
                </div>
              ))}
            </div>
            <Link
              href="/api/products?limit=50&offset=0"
              className="mt-5 inline-flex rounded-full border border-cyan-300/50 px-4 py-2 text-xs font-semibold tracking-wide text-cyan-100 transition hover:border-cyan-200 hover:bg-cyan-400/10"
            >
              View Products API
            </Link>
          </aside>
        </section>

        <section className="mt-20">
          <div className="mb-5 flex items-end justify-between gap-4">
            <h2 className="text-2xl font-semibold text-white sm:text-3xl">Featured products</h2>
            <Link href="/catalog" className="text-sm font-medium text-cyan-200 transition hover:text-cyan-100">
              View all products
            </Link>
          </div>
          <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
            {featuredProducts.map((product, idx) => (
              <article
                key={product.name}
                className="animate-fade-in-up rounded-3xl border border-white/12 bg-white/5 p-4"
                style={{ animationDelay: `${280 + idx * 100}ms` }}
              >
                <div
                  className={`mb-4 h-40 rounded-2xl border border-white/10 bg-gradient-to-br ${product.accent}`}
                />
                <p className="text-xs tracking-[0.14em] text-slate-400 uppercase">{product.category}</p>
                <h3 className="mt-1 text-base font-semibold text-white">{product.name}</h3>
                <div className="mt-4 flex items-center justify-between">
                  <p className="text-sm font-semibold text-cyan-200">{product.price}</p>
                  <button className="rounded-full border border-white/15 px-3 py-1.5 text-xs font-medium text-slate-200 transition hover:border-cyan-300/70 hover:text-white">
                    Add to cart
                  </button>
                </div>
              </article>
            ))}
          </div>
        </section>
      </main>
    </div>
  );
}
