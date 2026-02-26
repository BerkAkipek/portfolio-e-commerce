import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import OrderDetailClient from "./order-detail-client";

function renderOrderDetail(orderID = "order-1") {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <OrderDetailClient orderID={orderID} />
    </QueryClientProvider>,
  );
}

describe("OrderDetailClient", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("shows login state when unauthorized", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        status: 401,
        ok: false,
        json: async () => ({ error: "unauthorized" }),
      }),
    );

    renderOrderDetail();
    expect(await screen.findByText("Login required")).toBeInTheDocument();
  });

  it("renders order details and line items", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        status: 200,
        ok: true,
        json: async () => ({
          data: {
            order: {
              id: "order-1",
              user_id: "user-1",
              status: "paid",
              currency: "USD",
              subtotal_cents: 1000,
              total_cents: 1200,
              created_at: "2026-02-01T00:00:00Z",
              updated_at: "2026-02-01T00:00:00Z",
            },
            items: [
              {
                id: "item-1",
                order_id: "order-1",
                product_id: "prod-1",
                product_name: "Pro Tee",
                price_cents_snapshot: 600,
                quantity: 2,
                created_at: "2026-02-01T00:00:00Z",
              },
            ],
          },
        }),
      }),
    );

    renderOrderDetail();

    expect(await screen.findByText("order-1")).toBeInTheDocument();
    expect(screen.getByText("Pro Tee")).toBeInTheDocument();
    expect(screen.getByText("Qty: 2")).toBeInTheDocument();
    expect(screen.getAllByText("$12.00").length).toBeGreaterThan(0);
  });
});
