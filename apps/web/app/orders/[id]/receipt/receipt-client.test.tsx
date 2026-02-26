import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import ReceiptClient from "./receipt-client";

function renderReceipt(orderID = "order-1") {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <ReceiptClient orderID={orderID} />
    </QueryClientProvider>,
  );
}

describe("ReceiptClient", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders paid receipt with line items", async () => {
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

    renderReceipt();
    expect(await screen.findByText("Order #order-1")).toBeInTheDocument();
    expect(screen.getByText("Pro Tee")).toBeInTheDocument();
    expect(screen.getByText("Subtotal")).toBeInTheDocument();
    expect(screen.getByText("Total")).toBeInTheDocument();
  });

  it("shows unavailable state for missing order", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        status: 404,
        ok: false,
        json: async () => ({ error: "order not found" }),
      }),
    );

    renderReceipt();
    expect(await screen.findByText("Receipt unavailable")).toBeInTheDocument();
  });
});
