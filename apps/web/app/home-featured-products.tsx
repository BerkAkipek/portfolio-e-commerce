"use client";

import { useMutation, useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useState } from "react";

import { GUEST_CART_STORAGE_KEY } from "@/lib/cart";
import { addToCart, fetchProducts } from "@/lib/storefront-api";

function formatPrice(priceCents: number, currency: string): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: (currency || "USD").toUpperCase(),
  }).format(priceCents / 100);
}

function FeaturedSkeletonCard({ delayMs }: { delayMs: number }) {
  return (
    <article
      className="animate-fade-in-up rounded-3xl border border-white/12 bg-white/5 p-4"
      style={{ animationDelay: `${delayMs}ms` }}
    >
      <div className="mb-4 h-40 animate-pulse rounded-2xl border border-white/10 bg-white/10" />
      <div className="h-3 w-20 animate-pulse rounded bg-white/10" />
      <div className="mt-2 h-5 w-36 animate-pulse rounded bg-white/10" />
      <div className="mt-5 h-9 w-full animate-pulse rounded-full bg-white/10" />
    </article>
  );
}

export default function HomeFeaturedProducts() {
  const [message, setMessage] = useState("");

  const featuredQuery = useQuery({
    queryKey: ["home-featured-products"],
    queryFn: () => fetchProducts(4, 0),
  });

  const addMutation = useMutation({
    mutationFn: ({ productID }: { productID: string }) => addToCart(productID, 1),
    onSuccess: (payload) => {
      const returnedCartID = payload.data?.cart_id;
      if (returnedCartID) {
        window.localStorage.setItem(GUEST_CART_STORAGE_KEY, returnedCartID);
      }
      setMessage("Added to cart.");
    },
    onError: (error) => {
      const text = error instanceof Error ? error.message : "Unable to add item to cart.";
      setMessage(text);
    },
  });

  const products = featuredQuery.data?.data ?? [];

  return (
    <>
      {message ? <p className="mb-4 text-sm text-slate-200">{message}</p> : null}

      {featuredQuery.isLoading ? (
        <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, idx) => (
            <FeaturedSkeletonCard key={idx} delayMs={280 + idx * 100} />
          ))}
        </div>
      ) : null}

      {featuredQuery.isError ? (
        <div className="rounded-2xl border border-red-300/30 bg-red-400/10 p-4 text-sm text-red-100">
          Unable to load featured products right now.
        </div>
      ) : null}

      {!featuredQuery.isLoading && !featuredQuery.isError && products.length === 0 ? (
        <section className="rounded-3xl border border-white/12 bg-white/5 p-8 text-center">
          <h3 className="text-xl font-semibold text-white">No featured products yet</h3>
          <p className="mt-2 text-sm text-slate-300">
            New arrivals will appear here once products are available.
          </p>
          <Link
            href="/catalog"
            className="mt-5 inline-flex rounded-full bg-cyan-300 px-6 py-3 text-sm font-semibold tracking-wide text-[#03151b] transition hover:bg-cyan-200"
          >
            Browse catalog
          </Link>
        </section>
      ) : null}

      {!featuredQuery.isLoading && !featuredQuery.isError && products.length > 0 ? (
        <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
          {products.map((product, idx) => (
            <article
              key={product.id}
              className="animate-fade-in-up rounded-3xl border border-white/12 bg-white/5 p-4"
              style={{ animationDelay: `${280 + idx * 100}ms` }}
            >
              <div className="mb-4 flex h-40 items-center justify-center rounded-2xl border border-white/10 bg-gradient-to-br from-cyan-500/25 to-cyan-300/5">
                <span className="text-xs tracking-[0.14em] text-slate-300 uppercase">
                  {product.categories?.[0] || "Featured"}
                </span>
              </div>
              <p className="text-xs tracking-[0.14em] text-slate-400 uppercase">
                {product.categories?.join(" · ") || "Uncategorized"}
              </p>
              <h3 className="mt-1 line-clamp-2 min-h-12 text-base font-semibold text-white">
                <Link href={`/products/${product.slug}`} className="transition hover:text-cyan-200">
                  {product.name}
                </Link>
              </h3>
              <div className="mt-4 flex items-center justify-between gap-2">
                <p className="text-sm font-semibold text-cyan-200">
                  {formatPrice(product.price_cents, product.currency)}
                </p>
                <button
                  type="button"
                  disabled={addMutation.isPending || product.stock <= 0}
                  onClick={() => {
                    setMessage("");
                    addMutation.mutate({ productID: product.id });
                  }}
                  className="rounded-full border border-white/15 px-3 py-1.5 text-xs font-medium text-slate-200 transition hover:border-cyan-300/70 hover:text-white disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {product.stock <= 0
                    ? "Out of stock"
                    : addMutation.isPending
                      ? "Adding..."
                      : "Add to cart"}
                </button>
              </div>
            </article>
          ))}
        </div>
      ) : null}
    </>
  );
}
