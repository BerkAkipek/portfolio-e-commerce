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

type OrderItem = {
  id: string;
  order_id: string;
  product_id: string;
  product_name: string;
  price_cents_snapshot: number;
  quantity: number;
  created_at: string;
};

type OrderDetailResponse = {
  data?: {
    order: Order;
    items: OrderItem[];
  };
  error?: string;
};

function formatPrice(cents: number, currency: string): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: (currency || "USD").toUpperCase(),
  }).format(cents / 100);
}

async function fetchOrderDetail(orderID: string): Promise<OrderDetailResponse> {
  const response = await fetch(`/api/orders/${encodeURIComponent(orderID)}`, {
    cache: "no-store",
  });

  const payload = (await response.json()) as OrderDetailResponse;
  if (response.status === 401) {
    return { error: "unauthorized" };
  }
  if (response.status === 404) {
    return { error: "not_found" };
  }
  if (!response.ok) {
    throw new Error(payload.error || "failed to load order");
  }
  return payload;
}

export default function OrderDetailClient({ orderID }: { orderID: string }) {
  const query = useQuery({
    queryKey: ["order-detail", orderID],
    queryFn: () => fetchOrderDetail(orderID),
  });

  const order = query.data?.data?.order;
  const items = query.data?.data?.items ?? [];

  return (
    <main className="relative mx-auto w-full max-w-6xl px-6 pb-14 pt-8 sm:px-10 lg:px-12">
      <div className="mb-8 flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-3xl font-semibold text-white sm:text-4xl">Order Details</h1>
        <Link
          href="/orders"
          className="rounded-full border border-white/20 px-4 py-2 text-sm text-slate-200 transition hover:border-cyan-300/80 hover:text-white"
        >
          Back to My Orders
        </Link>
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
            Please sign in to view this order.
          </p>
          <Link
            href="/auth"
            className="mt-5 inline-flex rounded-full bg-cyan-300 px-6 py-3 text-sm font-semibold tracking-wide text-[#03151b] transition hover:bg-cyan-200"
          >
            Go to login
          </Link>
        </section>
      ) : null}

      {query.data?.error === "not_found" ? (
        <section className="rounded-3xl border border-white/12 bg-white/5 p-8 text-center">
          <h2 className="text-xl font-semibold text-white">Order not found</h2>
          <p className="mt-2 text-sm text-slate-300">
            This order does not exist or you do not have access to it.
          </p>
        </section>
      ) : null}

      {query.isError ? (
        <div className="rounded-2xl border border-red-300/30 bg-red-400/10 p-4 text-sm text-red-100">
          Failed to load order details.
        </div>
      ) : null}

      {order && query.data?.error !== "unauthorized" && query.data?.error !== "not_found" ? (
        <section className="space-y-5">
          <article className="rounded-2xl border border-white/12 bg-white/5 p-4">
            <p className="text-xs tracking-[0.12em] text-slate-400 uppercase">Order</p>
            <p className="mt-1 text-sm font-medium text-slate-200">{order.id}</p>
            <div className="mt-4 flex flex-wrap gap-6 text-sm">
              <div>
                <p className="text-xs tracking-[0.12em] text-slate-400 uppercase">Status</p>
                <p className="mt-1 font-semibold text-cyan-200 uppercase">{order.status}</p>
              </div>
              <div>
                <p className="text-xs tracking-[0.12em] text-slate-400 uppercase">Placed</p>
                <p className="mt-1 text-slate-200">{new Date(order.created_at).toLocaleString("en-US")}</p>
              </div>
              <div>
                <p className="text-xs tracking-[0.12em] text-slate-400 uppercase">Total</p>
                <p className="mt-1 font-semibold text-white">
                  {formatPrice(order.total_cents, order.currency)}
                </p>
              </div>
            </div>
          </article>

          <section className="space-y-3">
            <h2 className="text-xl font-semibold text-white">Items</h2>
            {items.length === 0 ? (
              <div className="rounded-2xl border border-white/12 bg-white/5 p-4 text-sm text-slate-300">
                No line items recorded for this order.
              </div>
            ) : (
              items.map((item) => (
                <article key={item.id} className="rounded-2xl border border-white/12 bg-white/5 p-4">
                  <div className="flex flex-wrap items-center justify-between gap-3">
                    <div>
                      <p className="text-sm font-semibold text-white">{item.product_name}</p>
                      <p className="mt-1 text-xs text-slate-400">
                        Product ID: {item.product_id}
                      </p>
                    </div>
                    <div className="text-right">
                      <p className="text-sm text-slate-200">Qty: {item.quantity}</p>
                      <p className="mt-1 text-sm font-semibold text-cyan-200">
                        {formatPrice(item.price_cents_snapshot * item.quantity, order.currency)}
                      </p>
                    </div>
                  </div>
                </article>
              ))
            )}
          </section>
        </section>
      ) : null}
    </main>
  );
}
