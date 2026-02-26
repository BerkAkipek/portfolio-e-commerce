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

function hasReceipt(status: string): boolean {
  const normalized = status.trim().toLowerCase();
  return normalized === "paid" || normalized === "succeeded" || normalized === "completed";
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
    throw new Error(payload.error || "failed to load receipt");
  }
  return payload;
}

export default function ReceiptClient({ orderID }: { orderID: string }) {
  const query = useQuery({
    queryKey: ["order-receipt", orderID],
    queryFn: () => fetchOrderDetail(orderID),
  });

  const order = query.data?.data?.order;
  const items = query.data?.data?.items ?? [];

  return (
    <main className="relative mx-auto w-full max-w-3xl px-6 pb-14 pt-8 sm:px-10">
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3 print:hidden">
        <h1 className="text-3xl font-semibold text-white">Receipt</h1>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={() => window.print()}
            className="rounded-full bg-cyan-300 px-4 py-2 text-sm font-semibold text-[#03151b] transition hover:bg-cyan-200"
          >
            Print
          </button>
          <Link
            href={`/orders/${orderID}`}
            className="rounded-full border border-white/20 px-4 py-2 text-sm text-slate-200 transition hover:border-cyan-300/80 hover:text-white"
          >
            Order details
          </Link>
        </div>
      </div>

      {query.isLoading ? (
        <div className="h-32 animate-pulse rounded-2xl border border-white/10 bg-white/8" />
      ) : null}

      {query.data?.error === "unauthorized" ? (
        <section className="rounded-3xl border border-white/12 bg-white/5 p-8 text-center">
          <h2 className="text-xl font-semibold text-white">Login required</h2>
          <p className="mt-2 text-sm text-slate-300">Please sign in to access this receipt.</p>
        </section>
      ) : null}

      {query.data?.error === "not_found" ? (
        <section className="rounded-3xl border border-white/12 bg-white/5 p-8 text-center">
          <h2 className="text-xl font-semibold text-white">Receipt unavailable</h2>
          <p className="mt-2 text-sm text-slate-300">
            The order could not be found or is not accessible.
          </p>
        </section>
      ) : null}

      {order && query.data?.error !== "unauthorized" && query.data?.error !== "not_found" ? (
        <section className="rounded-3xl border border-white/12 bg-white/5 p-6 print:border-slate-200 print:bg-white print:text-black">
          <div className="mb-4 border-b border-white/10 pb-4 print:border-slate-200">
            <p className="text-xs tracking-[0.14em] text-slate-400 uppercase print:text-slate-500">
              Receipt
            </p>
            <p className="mt-1 text-sm font-medium">Order #{order.id}</p>
            <p className="mt-1 text-sm text-slate-300 print:text-slate-700">
              {new Date(order.created_at).toLocaleString("en-US")}
            </p>
            {!hasReceipt(order.status) ? (
              <p className="mt-2 text-xs text-amber-200 print:text-amber-700">
                Payment not finalized yet. Receipt is not final.
              </p>
            ) : null}
          </div>

          <div className="space-y-3">
            {items.map((item) => (
              <div key={item.id} className="flex items-center justify-between border-b border-white/8 pb-3 print:border-slate-100">
                <div>
                  <p className="text-sm font-semibold">{item.product_name}</p>
                  <p className="text-xs text-slate-400 print:text-slate-600">Qty {item.quantity}</p>
                </div>
                <p className="text-sm font-semibold">
                  {formatPrice(item.price_cents_snapshot * item.quantity, order.currency)}
                </p>
              </div>
            ))}
          </div>

          <div className="mt-4 border-t border-white/10 pt-4 print:border-slate-200">
            <div className="flex items-center justify-between text-sm text-slate-300 print:text-slate-700">
              <span>Subtotal</span>
              <span>{formatPrice(order.subtotal_cents, order.currency)}</span>
            </div>
            <div className="mt-2 flex items-center justify-between text-base font-semibold">
              <span>Total</span>
              <span>{formatPrice(order.total_cents, order.currency)}</span>
            </div>
          </div>
        </section>
      ) : null}
    </main>
  );
}
