"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { useEffect, useMemo, useState } from "react";

import { GUEST_CART_STORAGE_KEY } from "@/lib/cart";
import { isLikelyNetworkError } from "@/lib/network";

type CartItem = {
  id: string;
  cart_id: string;
  product_id: string;
  quantity: number;
  price_cents_snapshot: number;
  created_at: string;
};

type CartItemsResponse = {
  data: CartItem[];
  notFound?: boolean;
};

type CheckoutSessionResponse = {
  data?: {
    session_id: string;
    url: string;
  };
  error?: string;
};

function formatPrice(priceCents: number): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
  }).format(priceCents / 100);
}

async function fetchCartItems(cartID: string): Promise<CartItemsResponse> {
  const response = await fetch(`/api/cart/${encodeURIComponent(cartID)}/items`, {
    cache: "no-store",
  });

  if (response.status === 404) {
    return { data: [], notFound: true };
  }

  if (!response.ok) {
    throw new Error("failed to load cart");
  }

  return (await response.json()) as CartItemsResponse;
}

async function updateCartItem(itemID: string, quantity: number) {
  const response = await fetch(`/api/cart/items/${encodeURIComponent(itemID)}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ quantity }),
  });

  const payload = (await response.json()) as { error?: string };
  if (!response.ok) {
    throw new Error(payload.error || "failed to update item");
  }
  return payload;
}

async function removeCartItem(itemID: string) {
  const response = await fetch(`/api/cart/items/${encodeURIComponent(itemID)}`, {
    method: "DELETE",
  });

  if (!response.ok && response.status !== 204) {
    throw new Error("failed to remove item");
  }
}

async function createCheckoutSession() {
  const origin = typeof window !== "undefined" ? window.location.origin : "";
  const response = await fetch("/api/checkout/session", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      success_url: `${origin}/checkout/success`,
      cancel_url: `${origin}/checkout/cancel`,
    }),
  });

  const payload = (await response.json()) as CheckoutSessionResponse;
  if (!response.ok) {
    throw new Error(payload.error || "failed to start checkout");
  }
  const checkoutURL = payload.data?.url;
  if (!checkoutURL) {
    throw new Error("missing checkout URL");
  }
  return checkoutURL;
}

export default function CartClient() {
  const queryClient = useQueryClient();

  const [cartID] = useState(() => {
    if (typeof window === "undefined") {
      return "";
    }
    return window.localStorage.getItem(GUEST_CART_STORAGE_KEY) ?? "";
  });
  const [message, setMessage] = useState("");
  const [isOffline, setIsOffline] = useState(() =>
    typeof navigator !== "undefined" ? !navigator.onLine : false,
  );

  const query = useQuery({
    queryKey: ["cart-items", cartID],
    queryFn: () => fetchCartItems(cartID),
    enabled: cartID !== "",
    retry: false,
  });

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    const handleOnline = () => setIsOffline(false);
    const handleOffline = () => setIsOffline(true);
    window.addEventListener("online", handleOnline);
    window.addEventListener("offline", handleOffline);
    return () => {
      window.removeEventListener("online", handleOnline);
      window.removeEventListener("offline", handleOffline);
    };
  }, []);

  useEffect(() => {
    if (!query.data?.notFound) {
      return;
    }
    if (typeof window !== "undefined") {
      window.localStorage.removeItem(GUEST_CART_STORAGE_KEY);
    }
  }, [query.data?.notFound]);

  const subtotal = useMemo(() => {
    return (query.data?.data ?? []).reduce((sum, item) => {
      return sum + item.price_cents_snapshot * item.quantity;
    }, 0);
  }, [query.data]);

  const updateMutation = useMutation({
    mutationFn: ({ itemID, quantity }: { itemID: string; quantity: number }) =>
      updateCartItem(itemID, quantity),
    onSuccess: async () => {
      setMessage("Cart updated.");
      await queryClient.invalidateQueries({ queryKey: ["cart-items", cartID] });
    },
    onError: (error) => {
      if (isLikelyNetworkError(error)) {
        setMessage("Network unavailable. Reconnect and try again.");
        return;
      }
      setMessage(error instanceof Error ? error.message : "Unable to update cart.");
    },
  });

  const removeMutation = useMutation({
    mutationFn: (itemID: string) => removeCartItem(itemID),
    onSuccess: async () => {
      setMessage("Item removed.");
      await queryClient.invalidateQueries({ queryKey: ["cart-items", cartID] });
    },
    onError: () => {
      if (isOffline) {
        setMessage("You are offline. Reconnect to remove items.");
        return;
      }
      setMessage("Unable to remove item.");
    },
  });

  const checkoutMutation = useMutation({
    mutationFn: createCheckoutSession,
    onSuccess: (checkoutURL) => {
      window.location.assign(checkoutURL);
    },
    onError: (error) => {
      if (isLikelyNetworkError(error)) {
        setMessage("You appear offline. Reconnect to continue checkout.");
        return;
      }
      const text = error instanceof Error ? error.message : "Unable to start checkout.";
      if (text.toLowerCase().includes("unauthorized")) {
        setMessage("Please login to continue to checkout.");
        return;
      }
      setMessage(text);
    },
  });

  const items = query.data?.data ?? [];
  const hasItems = items.length > 0;
  const isMutating = updateMutation.isPending || removeMutation.isPending;
  const userMessage = query.data?.notFound
    ? "Your previous cart was no longer available. Start a new cart."
    : message;
  const cartLoadErrorMessage = isOffline || isLikelyNetworkError(query.error)
    ? "You appear offline. Reconnect to load your cart."
    : "Failed to load cart items.";

  return (
    <main className="relative mx-auto w-full max-w-6xl px-6 pb-14 pt-8 sm:px-10 lg:px-12">
      <div className="mb-8 flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-3xl font-semibold text-white sm:text-4xl">Your Cart</h1>
        <Link
          href="/catalog"
          className="rounded-full border border-white/20 px-4 py-2 text-sm text-slate-200 transition hover:border-cyan-300/80 hover:text-white"
        >
          Continue shopping
        </Link>
      </div>

      <div className="mb-6 rounded-2xl border border-cyan-300/25 bg-cyan-400/10 p-4 text-sm text-cyan-100">
        Sign in to save your cart across devices and sessions.
      </div>

      {userMessage ? <p className="mb-4 text-sm text-slate-200">{userMessage}</p> : null}

      {!cartID || (!query.isLoading && !hasItems) ? (
        <section className="rounded-3xl border border-white/12 bg-white/5 p-8 text-center">
          <h2 className="text-xl font-semibold text-white">Your cart is empty</h2>
          <p className="mt-2 text-sm text-slate-300">
            Add products to start checkout. Guest carts are supported instantly.
          </p>
          <Link
            href="/catalog"
            className="mt-5 inline-flex rounded-full bg-cyan-300 px-6 py-3 text-sm font-semibold tracking-wide text-[#03151b] transition hover:bg-cyan-200"
          >
            Browse products
          </Link>
        </section>
      ) : null}

      {query.isLoading ? (
        <div className="space-y-4">
          <div className="h-24 animate-pulse rounded-2xl border border-white/10 bg-white/8" />
          <div className="h-24 animate-pulse rounded-2xl border border-white/10 bg-white/8" />
        </div>
      ) : null}

      {query.isError ? (
        <section className="rounded-2xl border border-red-300/30 bg-red-400/10 p-4 text-sm text-red-100">
          <p>{cartLoadErrorMessage}</p>
          <button
            type="button"
            onClick={() => query.refetch()}
            className="mt-3 rounded-full border border-red-200/30 px-3 py-1.5 text-xs font-semibold text-red-100 transition hover:border-red-100/70"
          >
            Retry
          </button>
        </section>
      ) : null}

      {hasItems ? (
        <section className="grid gap-6 lg:grid-cols-[1.4fr_0.6fr]">
          <div className="space-y-4">
            {items.map((item) => (
              <article
                key={item.id}
                className="rounded-2xl border border-white/12 bg-white/5 p-4"
              >
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <p className="text-xs tracking-[0.12em] text-slate-400 uppercase">
                      Product ID
                    </p>
                    <p className="text-sm font-medium text-slate-200">{item.product_id}</p>
                    <p className="mt-1 text-sm text-cyan-200">
                      {formatPrice(item.price_cents_snapshot)} each
                    </p>
                  </div>

                  <div className="flex items-center gap-2">
                    <button
                      type="button"
                      disabled={isMutating}
                      onClick={() =>
                        isMutating
                          ? undefined
                          : updateMutation.mutate({
                              itemID: item.id,
                              quantity: Math.max(1, item.quantity - 1),
                            })
                      }
                      className="h-9 w-9 rounded-full border border-white/20 text-slate-100 transition hover:border-cyan-300/80 disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      -
                    </button>
                    <span className="min-w-10 text-center text-sm font-semibold text-white">
                      {item.quantity}
                    </span>
                    <button
                      type="button"
                      disabled={isMutating}
                      onClick={() =>
                        isMutating
                          ? undefined
                          : updateMutation.mutate({
                              itemID: item.id,
                              quantity: item.quantity + 1,
                            })
                      }
                      className="h-9 w-9 rounded-full border border-white/20 text-slate-100 transition hover:border-cyan-300/80 disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      +
                    </button>
                    <button
                      type="button"
                      disabled={isMutating}
                      onClick={() => (isMutating ? undefined : removeMutation.mutate(item.id))}
                      className="ml-2 rounded-full border border-white/20 px-3 py-1.5 text-xs text-slate-200 transition hover:border-red-300/80 hover:text-red-100 disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      Remove
                    </button>
                  </div>
                </div>
              </article>
            ))}
          </div>

          <aside className="h-fit rounded-2xl border border-white/12 bg-white/5 p-5">
            <p className="text-xs tracking-[0.16em] text-slate-400 uppercase">Subtotal</p>
            <p className="mt-2 text-3xl font-semibold text-white">{formatPrice(subtotal)}</p>
            <p className="mt-2 text-xs text-slate-400">Taxes and shipping calculated at checkout.</p>
            <button
              type="button"
              disabled={checkoutMutation.isPending}
              onClick={() => checkoutMutation.mutate()}
              className="mt-5 w-full rounded-full bg-cyan-300 px-5 py-3 text-sm font-semibold tracking-wide text-[#03151b] transition hover:bg-cyan-200"
            >
              {checkoutMutation.isPending ? "Redirecting..." : "Proceed to checkout"}
            </button>
          </aside>
        </section>
      ) : null}
    </main>
  );
}
