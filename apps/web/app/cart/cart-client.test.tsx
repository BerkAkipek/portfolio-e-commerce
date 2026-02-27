import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { GUEST_CART_STORAGE_KEY } from "@/lib/cart";

import CartClient from "./cart-client";

function renderCart() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <CartClient />
    </QueryClientProvider>,
  );
}

describe("CartClient", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    window.localStorage.clear();
  });

  it("shows empty state and sign-in messaging without stored cart", async () => {
    renderCart();

    expect(await screen.findByText("Your cart is empty")).toBeInTheDocument();
    expect(
      screen.getByText("Sign in to save your cart across devices and sessions."),
    ).toBeInTheDocument();
  });

  it("renders items and subtotal from cart API", async () => {
    window.localStorage.setItem(GUEST_CART_STORAGE_KEY, "cart-1");
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({
          data: [
            {
              id: "item-1",
              cart_id: "cart-1",
              product_id: "prod-1",
              quantity: 2,
              price_cents_snapshot: 1500,
              created_at: "2026-01-01T00:00:00Z",
            },
            {
              id: "item-2",
              cart_id: "cart-1",
              product_id: "prod-2",
              quantity: 1,
              price_cents_snapshot: 700,
              created_at: "2026-01-01T00:00:00Z",
            },
          ],
        }),
      }),
    );

    renderCart();
    expect(await screen.findByText("prod-1")).toBeInTheDocument();
    expect(screen.getByText("$37.00")).toBeInTheDocument();
  });

  it("clears stale guest cart id when cart is not found", async () => {
    window.localStorage.setItem(GUEST_CART_STORAGE_KEY, "missing-cart");
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 404,
      }),
    );

    renderCart();

    expect(
      await screen.findByText("Your previous cart was no longer available. Start a new cart."),
    ).toBeInTheDocument();
    expect(window.localStorage.getItem(GUEST_CART_STORAGE_KEY)).toBeNull();
    expect(screen.getByText("Your cart is empty")).toBeInTheDocument();
  });

  it("updates quantity and shows confirmation message", async () => {
    window.localStorage.setItem(GUEST_CART_STORAGE_KEY, "cart-2");
    const fetchSpy = vi.fn().mockImplementation((input: string | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.includes("/api/cart/cart-2/items")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => ({
            data: [
              {
                id: "item-9",
                cart_id: "cart-2",
                product_id: "prod-9",
                quantity: 1,
                price_cents_snapshot: 990,
                created_at: "2026-01-01T00:00:00Z",
              },
            ],
          }),
        });
      }
      if (url.includes("/api/cart/items/item-9") && init?.method === "PATCH") {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => ({ data: {} }),
        });
      }
      return Promise.resolve({
        ok: false,
        status: 500,
        json: async () => ({ error: "unexpected request" }),
      });
    });
    vi.stubGlobal("fetch", fetchSpy);

    const user = userEvent.setup();
    renderCart();
    await screen.findByText("prod-9");

    await user.click(screen.getByRole("button", { name: "+" }));
    await screen.findByText("Cart updated.");

    const patchCall = fetchSpy.mock.calls.find(
      ([url, init]) =>
        String(url).includes("/api/cart/items/item-9") &&
        (init as RequestInit | undefined)?.method === "PATCH",
    );
    expect(patchCall).toBeTruthy();
  });

  it("prevents duplicate quantity update requests while a mutation is pending", async () => {
    window.localStorage.setItem(GUEST_CART_STORAGE_KEY, "cart-4");

    let resolvePatch: (() => void) | null = null;
    const patchPromise = new Promise<void>((resolve) => {
      resolvePatch = resolve;
    });

    const fetchSpy = vi.fn().mockImplementation((input: string | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.includes("/api/cart/cart-4/items")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => ({
            data: [
              {
                id: "item-4",
                cart_id: "cart-4",
                product_id: "prod-4",
                quantity: 1,
                price_cents_snapshot: 1000,
                created_at: "2026-01-01T00:00:00Z",
              },
            ],
          }),
        });
      }
      if (url.includes("/api/cart/items/item-4") && init?.method === "PATCH") {
        return patchPromise.then(() => ({
          ok: true,
          status: 200,
          json: async () => ({ data: {} }),
        }));
      }
      return Promise.resolve({
        ok: false,
        status: 500,
        json: async () => ({ error: "unexpected request" }),
      });
    });
    vi.stubGlobal("fetch", fetchSpy);

    const user = userEvent.setup();
    renderCart();
    const plusButton = await screen.findByRole("button", { name: "+" });

    await user.click(plusButton);
    await waitFor(() => {
      expect(plusButton).toBeDisabled();
    });
    await user.click(plusButton);

    const patchCalls = fetchSpy.mock.calls.filter(
      ([url, init]) =>
        String(url).includes("/api/cart/items/item-4") &&
        (init as RequestInit | undefined)?.method === "PATCH",
    );
    expect(patchCalls).toHaveLength(1);

    if (resolvePatch) {
      resolvePatch();
    }
    await screen.findByText("Cart updated.");
  });

  it("removes item and shows confirmation message", async () => {
    window.localStorage.setItem(GUEST_CART_STORAGE_KEY, "cart-3");
    let getCount = 0;
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation((input: string | URL, init?: RequestInit) => {
        const url = String(input);
        if (url.includes("/api/cart/cart-3/items")) {
          getCount += 1;
          if (getCount === 1) {
            return Promise.resolve({
              ok: true,
              status: 200,
              json: async () => ({
                data: [
                  {
                    id: "item-3",
                    cart_id: "cart-3",
                    product_id: "prod-3",
                    quantity: 1,
                    price_cents_snapshot: 2500,
                    created_at: "2026-01-01T00:00:00Z",
                  },
                ],
              }),
            });
          }
          return Promise.resolve({
            ok: true,
            status: 200,
            json: async () => ({ data: [] }),
          });
        }
        if (url.includes("/api/cart/items/item-3") && init?.method === "DELETE") {
          return Promise.resolve({
            ok: true,
            status: 204,
            text: async () => "",
          });
        }
        return Promise.resolve({
          ok: false,
          status: 500,
          json: async () => ({ error: "unexpected request" }),
        });
      }),
    );

    const user = userEvent.setup();
    renderCart();
    await screen.findByText("prod-3");

    await user.click(screen.getByRole("button", { name: "Remove" }));
    await screen.findByText("Item removed.");
    await waitFor(() => {
      expect(screen.getByText("Your cart is empty")).toBeInTheDocument();
    });
  });

  it("starts checkout and calls checkout session endpoint", async () => {
    window.localStorage.setItem(GUEST_CART_STORAGE_KEY, "cart-5");
    const fetchSpy = vi.fn().mockImplementation((input: string | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.includes("/api/cart/cart-5/items")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => ({
            data: [
              {
                id: "item-5",
                cart_id: "cart-5",
                product_id: "prod-5",
                quantity: 1,
                price_cents_snapshot: 2000,
                created_at: "2026-01-01T00:00:00Z",
              },
            ],
          }),
        });
      }
      if (url === "/api/checkout/session" && init?.method === "POST") {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => ({
            data: { session_id: "cs_1", url: "https://checkout.stripe.com/c/pay/cs_1" },
          }),
        });
      }
      return Promise.resolve({
        ok: false,
        status: 500,
        json: async () => ({ error: "unexpected request" }),
      });
    });
    vi.stubGlobal("fetch", fetchSpy);
    const user = userEvent.setup();
    renderCart();
    await screen.findByText("prod-5");

    await user.click(screen.getByRole("button", { name: "Proceed to checkout" }));
    await waitFor(() => {
      expect(fetchSpy).toHaveBeenCalledWith(
        "/api/checkout/session",
        expect.objectContaining({
          method: "POST",
        }),
      );
    });
  });

  it("shows offline-friendly cart load error and retry action", async () => {
    window.localStorage.setItem(GUEST_CART_STORAGE_KEY, "cart-network");
    vi.stubGlobal(
      "fetch",
      vi.fn().mockRejectedValue(new TypeError("Failed to fetch")),
    );

    renderCart();

    expect(
      await screen.findByText("You appear offline. Reconnect to load your cart."),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument();
  });
});
