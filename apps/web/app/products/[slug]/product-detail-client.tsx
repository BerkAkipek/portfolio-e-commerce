"use client";
/* eslint-disable @next/next/no-img-element */

import { useMutation, useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useMemo, useState } from "react";

import { GUEST_CART_STORAGE_KEY } from "@/lib/cart";

type Product = {
  id: string;
  name: string;
  slug: string;
  description: string;
  price_cents: number;
  currency: string;
  stock: number;
  categories?: string[];
  image_url?: string | null;
};

type ProductResponse = {
  data: Product;
};

function formatPrice(priceCents: number, currency: string): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: (currency || "USD").toUpperCase(),
  }).format(priceCents / 100);
}

function getStockLabel(stock: number): { text: string; className: string } {
  if (stock <= 0) {
    return { text: "Out of stock", className: "text-red-200" };
  }
  if (stock <= 10) {
    return { text: `Low stock (${stock} left)`, className: "text-amber-200" };
  }
  return { text: "In stock", className: "text-emerald-200" };
}

async function fetchProductBySlug(slug: string): Promise<ProductResponse> {
  const response = await fetch(`/api/products/${encodeURIComponent(slug)}`);
  if (!response.ok) {
    throw new Error("failed to load product");
  }
  return (await response.json()) as ProductResponse;
}

async function addToCart(productID: string, quantity: number) {
  const response = await fetch("/api/cart/items", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ product_id: productID, quantity }),
  });

  const payload = (await response.json()) as {
    data?: { cart_id?: string };
    error?: string;
  };
  if (!response.ok) {
    throw new Error(payload.error || "failed to add to cart");
  }

  return payload;
}

function buildImageGallery(product: Product) {
  if (!product.image_url) {
    return [];
  }
  return [
    product.image_url,
    `${product.image_url}?v=2`,
    `${product.image_url}?v=3`,
  ];
}

export default function ProductDetailClient({ slug }: { slug: string }) {
  const [quantity, setQuantity] = useState(1);
  const [message, setMessage] = useState("");

  const { data, isLoading, isError } = useQuery({
    queryKey: ["product-detail", slug],
    queryFn: () => fetchProductBySlug(slug),
  });

  const product = data?.data;
  const images = useMemo(() => (product ? buildImageGallery(product) : []), [product]);
  const [selectedImage, setSelectedImage] = useState(0);

  const mutation = useMutation({
    mutationFn: ({ productID, count }: { productID: string; count: number }) =>
      addToCart(productID, count),
    onSuccess: (payload) => {
      const returnedCartID = payload.data?.cart_id;
      if (returnedCartID) {
        window.localStorage.setItem(GUEST_CART_STORAGE_KEY, returnedCartID);
      }
      setMessage("Added to cart.");
    },
    onError: (error) => {
      const messageText =
        error instanceof Error ? error.message : "Unable to add item to cart.";
      setMessage(messageText);
    },
  });

  if (isLoading) {
    return (
      <main className="relative mx-auto w-full max-w-6xl px-6 pb-14 pt-8 sm:px-10 lg:px-12">
        <div className="grid gap-8 lg:grid-cols-2">
          <div className="h-[480px] animate-pulse rounded-3xl border border-white/10 bg-white/10" />
          <div className="space-y-4">
            <div className="h-4 w-24 animate-pulse rounded bg-white/10" />
            <div className="h-10 w-72 animate-pulse rounded bg-white/10" />
            <div className="h-6 w-32 animate-pulse rounded bg-white/10" />
            <div className="h-24 w-full animate-pulse rounded bg-white/10" />
          </div>
        </div>
      </main>
    );
  }

  if (isError || !product) {
    return (
      <main className="relative mx-auto w-full max-w-6xl px-6 pb-14 pt-8 sm:px-10 lg:px-12">
        <div className="rounded-2xl border border-red-300/30 bg-red-400/10 p-4 text-sm text-red-100">
          Product not found or unavailable.
        </div>
      </main>
    );
  }

  const stock = getStockLabel(product.stock);

  return (
    <main className="relative mx-auto w-full max-w-6xl px-6 pb-14 pt-8 sm:px-10 lg:px-12">
      <div className="mb-8 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Link
            href="/catalog"
            className="rounded-full border border-white/20 px-4 py-2 text-sm text-slate-200 transition hover:border-cyan-300/80 hover:text-white"
          >
            Back to catalog
          </Link>
          <Link
            href="/cart"
            className="rounded-full border border-white/20 px-4 py-2 text-sm text-slate-200 transition hover:border-cyan-300/80 hover:text-white"
          >
            Cart
          </Link>
        </div>
        <span className="text-xs tracking-[0.16em] text-slate-400 uppercase">
          {product.categories?.join(" · ") || "Uncategorized"}
        </span>
      </div>

      <section className="grid gap-8 lg:grid-cols-2">
        <div>
          <div className="flex h-[440px] items-center justify-center overflow-hidden rounded-3xl border border-white/12 bg-white/5">
            {images.length > 0 ? (
              <img
                src={images[Math.min(selectedImage, images.length - 1)]}
                alt={product.name}
                className="h-full w-full object-cover"
              />
            ) : (
              <div className="flex h-full w-full items-center justify-center bg-gradient-to-br from-cyan-400/20 to-emerald-400/10 text-sm tracking-[0.2em] text-slate-300 uppercase">
                No image
              </div>
            )}
          </div>

          {images.length > 1 ? (
            <div className="mt-4 grid grid-cols-3 gap-3">
              {images.map((url, idx) => (
                <button
                  key={url}
                  type="button"
                  onClick={() => setSelectedImage(idx)}
                  className={`h-24 overflow-hidden rounded-2xl border transition ${
                    selectedImage === idx
                      ? "border-cyan-300/80"
                      : "border-white/10 hover:border-white/30"
                  }`}
                >
                  <img src={url} alt={`${product.name} ${idx + 1}`} className="h-full w-full object-cover" />
                </button>
              ))}
            </div>
          ) : null}
        </div>

        <div className="rounded-3xl border border-white/12 bg-white/5 p-6">
          <h1 className="text-3xl font-semibold text-white sm:text-4xl">{product.name}</h1>
          <p className="mt-3 text-2xl font-semibold text-cyan-200">
            {formatPrice(product.price_cents, product.currency)}
          </p>
          <p className={`mt-2 text-sm font-medium ${stock.className}`}>{stock.text}</p>

          <p className="mt-6 text-sm leading-7 text-slate-300">
            {product.description || "No description provided for this product yet."}
          </p>

          <div className="mt-8 flex flex-wrap items-center gap-3">
            <label className="text-xs tracking-[0.12em] text-slate-400 uppercase" htmlFor="quantity">
              Quantity
            </label>
            <input
              id="quantity"
              type="number"
              min={1}
              value={quantity}
              onChange={(event) => {
                const parsed = Number.parseInt(event.target.value, 10);
                setQuantity(Number.isNaN(parsed) || parsed <= 0 ? 1 : parsed);
              }}
              className="h-10 w-24 rounded-xl border border-white/15 bg-black/30 px-3 text-sm text-white outline-none focus:border-cyan-300/70"
            />
            <button
              type="button"
              disabled={product.stock <= 0 || mutation.isPending}
              onClick={() => {
                setMessage("");
                mutation.mutate({ productID: product.id, count: quantity });
              }}
              className="rounded-full bg-cyan-300 px-6 py-3 text-sm font-semibold tracking-wide text-[#03151b] transition hover:bg-cyan-200 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {mutation.isPending ? "Adding..." : "Add to cart"}
            </button>
          </div>

          {message ? <p className="mt-4 text-sm text-slate-200">{message}</p> : null}
        </div>
      </section>
    </main>
  );
}
