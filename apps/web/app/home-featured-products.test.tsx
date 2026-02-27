import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import HomeFeaturedProducts from "./home-featured-products";
import { GUEST_CART_STORAGE_KEY } from "@/lib/cart";

function renderFeatured() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <HomeFeaturedProducts />
    </QueryClientProvider>,
  );
}

describe("HomeFeaturedProducts", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    window.localStorage.clear();
  });

  it("renders loading skeleton while featured products are pending", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation(
        () =>
          new Promise(() => {
            return undefined;
          }),
      ),
    );

    const { container } = renderFeatured();
    await waitFor(() => {
      expect(container.querySelectorAll(".animate-pulse").length).toBeGreaterThan(0);
    });
  });

  it("renders empty state when API returns no featured products", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({ data: [], limit: 4, offset: 0 }),
      }),
    );

    renderFeatured();

    expect(await screen.findByText("No featured products yet")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Browse catalog" })).toBeInTheDocument();
  });

  it("adds a featured product to cart", async () => {
    const fetchSpy = vi.fn().mockImplementation((input: string | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.startsWith("/api/products?")) {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            data: [
              {
                id: "p-1",
                name: "Aero Running Jacket",
                slug: "aero-running-jacket",
                description: "Lightweight weather-resistant jacket.",
                price_cents: 12900,
                currency: "USD",
                stock: 15,
                categories: ["Outerwear"],
              },
            ],
            limit: 4,
            offset: 0,
          }),
        });
      }
      if (url === "/api/cart/items" && init?.method === "POST") {
        return Promise.resolve({
          ok: true,
          json: async () => ({ data: { cart_id: "cart-123" } }),
        });
      }
      return Promise.resolve({
        ok: false,
        json: async () => ({ error: "not handled" }),
      });
    });
    vi.stubGlobal("fetch", fetchSpy);

    const user = userEvent.setup();
    renderFeatured();

    await screen.findByRole("heading", { name: "Aero Running Jacket" });
    await user.click(screen.getByRole("button", { name: "Add to cart" }));

    await screen.findByText("Added to cart.");
    const request = fetchSpy.mock.calls.find(([url]) => String(url) === "/api/cart/items");
    expect(request?.[1]?.body).toBe(JSON.stringify({ product_id: "p-1", quantity: 1 }));
    expect(window.localStorage.getItem(GUEST_CART_STORAGE_KEY)).toBe("cart-123");
  });
});
