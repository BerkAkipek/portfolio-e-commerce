import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import OrdersClient from "./orders-client";

function renderOrders() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <OrdersClient />
    </QueryClientProvider>,
  );
}

describe("OrdersClient", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("shows login required state when unauthenticated", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        status: 401,
        ok: false,
        json: async () => ({ error: "unauthorized" }),
      }),
    );

    renderOrders();
    expect(await screen.findByText("Login required")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Go to login" })).toHaveAttribute("href", "/auth");
  });

  it("renders orders list", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        status: 200,
        ok: true,
        json: async () => ({
          data: [
            {
              id: "order-1",
              user_id: "user-1",
              status: "paid",
              currency: "USD",
              subtotal_cents: 1000,
              total_cents: 1200,
              created_at: "2026-02-01T00:00:00Z",
              updated_at: "2026-02-01T00:00:00Z",
            },
          ],
          limit: 20,
          offset: 0,
        }),
      }),
    );

    renderOrders();
    expect(await screen.findByText("order-1")).toBeInTheDocument();
    expect(screen.getByText("$12.00")).toBeInTheDocument();
    expect(screen.getByText("paid")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "order-1" })).toHaveAttribute("href", "/orders/order-1");
    expect(screen.getByRole("link", { name: "View receipt" })).toHaveAttribute(
      "href",
      "/orders/order-1/receipt",
    );
  });
});
