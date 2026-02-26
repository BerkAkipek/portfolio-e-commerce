"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";

type Order = {
  id: string;
  user_id: string;
  status: string;
  currency: string;
  subtotal_cents: number;
  total_cents: number;
  created_at: string;
  updated_at: string;
};

type OrdersResponse = {
  data: Order[];
  limit: number;
  offset: number;
  error?: string;
};

function formatPrice(cents: number, currency: string): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: (currency || "USD").toUpperCase(),
  }).format(cents / 100);
}

function hasReceipt(status: string): boolean {
  const normalized = status.trim().toLowerCase();
  return normalized === "paid" || normalized === "succeeded" || normalized === "completed";
}

async function fetchOrders(): Promise<OrdersResponse> {
  const response = await fetch("/api/orders?limit=20&offset=0", {
    cache: "no-store",
  });

  if (response.status === 401) {
    return { data: [], limit: 20, offset: 0, error: "unauthorized" };
  }

  const payload = (await response.json()) as OrdersResponse;
  if (!response.ok) {
    throw new Error(payload.error || "failed to load orders");
  }
  return payload;
}

export default function OrdersClient() {
  const query = useQuery({
    queryKey: ["my-orders"],
    queryFn: fetchOrders,
  });

  const orders = query.data?.data ?? [];

  return (
    <main className="relative mx-auto w-full max-w-6xl px-6 pb-14 pt-8 sm:px-10 lg:px-12">
      <div className="mb-8 flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-3xl font-semibold text-white sm:text-4xl">My Orders</h1>
        <div className="flex gap-2">
          <Link
            href="/catalog"
            className="rounded-full border border-white/20 px-4 py-2 text-sm text-slate-200 transition hover:border-cyan-300/80 hover:text-white"
          >
            Catalog
          </Link>
          <Link
            href="/cart"
            className="rounded-full border border-white/20 px-4 py-2 text-sm text-slate-200 transition hover:border-cyan-300/80 hover:text-white"
          >
            Cart
          </Link>
        </div>
      </div>

      {query.isLoading ? (
        <div className="space-y-3">
          <div className="h-24 animate-pulse rounded-2xl border border-white/10 bg-white/8" />
          <div className="h-24 animate-pulse rounded-2xl border border-white/10 bg-white/8" />
        </div>
      ) : null}

      {query.data?.error === "unauthorized" ? (
        <section className="rounded-3xl border border-white/12 bg-white/5 p-8 text-center">
          <h2 className="text-xl font-semibold text-white">Login required</h2>
          <p className="mt-2 text-sm text-slate-300">
            Please sign in to view your order history.
          </p>
          <Link
            href="/auth"
            className="mt-5 inline-flex rounded-full bg-cyan-300 px-6 py-3 text-sm font-semibold tracking-wide text-[#03151b] transition hover:bg-cyan-200"
          >
            Go to login
          </Link>
        </section>
      ) : null}

      {query.isError ? (
        <div className="rounded-2xl border border-red-300/30 bg-red-400/10 p-4 text-sm text-red-100">
          Failed to load orders.
        </div>
      ) : null}

      {!query.isLoading && !query.isError && query.data?.error !== "unauthorized" && orders.length === 0 ? (
        <section className="rounded-3xl border border-white/12 bg-white/5 p-8 text-center">
          <h2 className="text-xl font-semibold text-white">No orders yet</h2>
          <p className="mt-2 text-sm text-slate-300">
            Complete checkout to see your order history here.
          </p>
        </section>
      ) : null}

      {orders.length > 0 ? (
        <section className="space-y-4">
          {orders.map((order) => (
            <article key={order.id} className="rounded-2xl border border-white/12 bg-white/5 p-4">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <p className="text-xs tracking-[0.12em] text-slate-400 uppercase">Order ID</p>
                  <p className="text-sm font-medium text-slate-200">
                    <Link href={`/orders/${order.id}`} className="transition hover:text-cyan-200">
                      {order.id}
                    </Link>
                  </p>
                  <p className="mt-1 text-xs text-slate-400">
                    {new Date(order.created_at).toLocaleString("en-US")}
                  </p>
                </div>
                <div className="text-right">
                  <p className="text-xs tracking-[0.12em] text-slate-400 uppercase">Status</p>
                  <p className="text-sm font-semibold text-cyan-200 uppercase">{order.status}</p>
                  <p className="mt-1 text-sm font-semibold text-white">
                    {formatPrice(order.total_cents, order.currency)}
                  </p>
                  {hasReceipt(order.status) ? (
                    <Link
                      href={`/orders/${order.id}/receipt`}
                      className="mt-2 inline-flex text-xs font-semibold text-cyan-200 transition hover:text-cyan-100"
                    >
                      View receipt
                    </Link>
                  ) : null}
                </div>
              </div>
            </article>
          ))}
        </section>
      ) : null}
    </main>
  );
}
