import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import ProductDetailClient from "./product-detail-client";

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

function renderProductPage(slug = "aero-running-jacket") {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <ProductDetailClient slug={slug} />
    </QueryClientProvider>,
  );
}

function mockProductFetch(product: Product) {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockImplementation((input: string | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.startsWith("/api/products/")) {
        return Promise.resolve({
          ok: true,
          json: async () => ({ data: product }),
        });
      }
      if (url === "/api/cart/items" && init?.method === "POST") {
        return Promise.resolve({
          ok: true,
          json: async () => ({ data: { id: "cart-item-id" } }),
        });
      }
      return Promise.resolve({
        ok: false,
        json: async () => ({ error: "not handled" }),
      });
    }),
  );
}

describe("ProductDetailClient", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders loading skeleton while product is pending", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation(
        () =>
          new Promise(() => {
            return undefined;
          }),
      ),
    );

    const { container } = renderProductPage();
    await waitFor(() => {
      expect(container.querySelectorAll(".animate-pulse").length).toBeGreaterThan(0);
    });
  });

  it("renders product details and low-stock status", async () => {
    mockProductFetch({
      id: "p1",
      name: "Aero Running Jacket",
      slug: "aero-running-jacket",
      description: "Lightweight weather-resistant jacket.",
      price_cents: 12900,
      currency: "USD",
      stock: 5,
      categories: ["Outerwear", "Training"],
      image_url: "https://example.com/jacket.jpg",
    });

    renderProductPage();

    expect(await screen.findByRole("heading", { name: "Aero Running Jacket" })).toBeInTheDocument();
    expect(screen.getByText("$129.00")).toBeInTheDocument();
    expect(screen.getByText("Low stock (5 left)")).toBeInTheDocument();
    expect(screen.getByText("Outerwear · Training")).toBeInTheDocument();
  });

  it("disables add-to-cart when stock is zero", async () => {
    mockProductFetch({
      id: "p2",
      name: "Sold Out Item",
      slug: "sold-out-item",
      description: "Unavailable right now.",
      price_cents: 4500,
      currency: "USD",
      stock: 0,
      categories: ["Accessories"],
      image_url: null,
    });

    renderProductPage("sold-out-item");
    await screen.findByRole("heading", { name: "Sold Out Item" });

    const addButton = screen.getByRole("button", { name: "Add to cart" });
    expect(addButton).toBeDisabled();
    expect(screen.getByText("Out of stock")).toBeInTheDocument();
  });

  it("adds product to cart successfully", async () => {
    const fetchSpy = vi.fn().mockImplementation((input: string | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.startsWith("/api/products/")) {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            data: {
              id: "p3",
              name: "Flex Everyday Tee",
              slug: "flex-everyday-tee",
              description: "Soft everyday tee.",
              price_cents: 3200,
              currency: "USD",
              stock: 20,
              categories: ["Apparel"],
              image_url: null,
            },
          }),
        });
      }
      if (url === "/api/cart/items" && init?.method === "POST") {
        return Promise.resolve({
          ok: true,
          json: async () => ({ data: { id: "created" } }),
        });
      }
      return Promise.resolve({
        ok: false,
        json: async () => ({ error: "not handled" }),
      });
    });
    vi.stubGlobal("fetch", fetchSpy);

    const user = userEvent.setup();
    renderProductPage("flex-everyday-tee");
    await screen.findByRole("heading", { name: "Flex Everyday Tee" });

    const quantityInput = screen.getByLabelText("Quantity");
    fireEvent.change(quantityInput, { target: { value: "3" } });
    await user.click(screen.getByRole("button", { name: "Add to cart" }));

    await screen.findByText("Added to cart.");
    const cartRequest = fetchSpy.mock.calls.find(
      ([url]) => String(url) === "/api/cart/items",
    );
    expect(cartRequest).toBeTruthy();
    expect(cartRequest?.[1]?.body).toBe(
      JSON.stringify({ product_id: "p3", quantity: 3 }),
    );
  });

  it("shows add-to-cart API error message", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation((input: string | URL, init?: RequestInit) => {
        const url = String(input);
        if (url.startsWith("/api/products/")) {
          return Promise.resolve({
            ok: true,
            json: async () => ({
              data: {
                id: "p4",
                name: "Core Training Set",
                slug: "core-training-set",
                description: "Training essentials.",
                price_cents: 8900,
                currency: "USD",
                stock: 7,
                categories: ["Training"],
                image_url: null,
              },
            }),
          });
        }
        if (url === "/api/cart/items" && init?.method === "POST") {
          return Promise.resolve({
            ok: false,
            json: async () => ({ error: "cart service unavailable" }),
          });
        }
        return Promise.resolve({
          ok: false,
          json: async () => ({ error: "not handled" }),
        });
      }),
    );

    const user = userEvent.setup();
    renderProductPage("core-training-set");
    await screen.findByRole("heading", { name: "Core Training Set" });

    await user.click(screen.getByRole("button", { name: "Add to cart" }));
    await screen.findByText("cart service unavailable");
  });
});
