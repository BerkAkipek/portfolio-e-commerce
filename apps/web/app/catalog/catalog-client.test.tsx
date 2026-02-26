import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import CatalogClient from "./catalog-client";

type Product = {
  id: string;
  name: string;
  slug: string;
  description: string;
  price_cents: number;
  currency: string;
  stock: number;
  categories?: string[];
};

const baseProducts: Product[] = [
  {
    id: "p1",
    name: "Aero Running Jacket",
    slug: "aero-running-jacket",
    description: "Light jacket for speed sessions.",
    price_cents: 12900,
    currency: "USD",
    stock: 12,
    categories: ["Outerwear", "Training"],
  },
  {
    id: "p2",
    name: "Urban Carry Pack",
    slug: "urban-carry-pack",
    description: "Commuter pack with laptop sleeve.",
    price_cents: 7400,
    currency: "USD",
    stock: 20,
    categories: ["Accessories", "Lifestyle"],
  },
  {
    id: "p3",
    name: "Flex Everyday Tee",
    slug: "flex-everyday-tee",
    description: "Soft daily tee.",
    price_cents: 3200,
    currency: "USD",
    stock: 30,
    categories: ["Apparel"],
  },
];

function renderCatalog() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <CatalogClient />
    </QueryClientProvider>,
  );
}

function mockProductsFetch(products: Product[]) {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        data: products,
        limit: 100,
        offset: 0,
      }),
    }),
  );
}

describe("CatalogClient", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders skeleton loading state while products are pending", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation(
        () =>
          new Promise(() => {
            return undefined;
          }),
      ),
    );

    const { container } = renderCatalog();

    await waitFor(() => {
      expect(container.querySelectorAll(".animate-pulse").length).toBeGreaterThan(0);
    });
  });

  it("searches by category text", async () => {
    mockProductsFetch(baseProducts);
    const user = userEvent.setup();
    renderCatalog();

    await screen.findByText("Aero Running Jacket");
    await user.type(
      screen.getByPlaceholderText("Search by product or category..."),
      "training",
    );

    expect(screen.getByText("Aero Running Jacket")).toBeInTheDocument();
    expect(screen.queryByText("Urban Carry Pack")).not.toBeInTheDocument();
  });

  it("filters by selected category chip", async () => {
    mockProductsFetch(baseProducts);
    const user = userEvent.setup();
    renderCatalog();

    await screen.findByText("Urban Carry Pack");
    await user.click(screen.getByRole("button", { name: "Accessories" }));

    expect(screen.getByText("Urban Carry Pack")).toBeInTheDocument();
    expect(screen.queryByText("Aero Running Jacket")).not.toBeInTheDocument();
  });

  it("sorts by price low to high", async () => {
    mockProductsFetch(baseProducts);
    const user = userEvent.setup();
    renderCatalog();

    await screen.findByText("Urban Carry Pack");
    await user.selectOptions(
      screen.getByRole("combobox"),
      screen.getByRole("option", { name: "Price: Low to High" }),
    );

    const titles = screen.getAllByRole("heading", { level: 2 }).map((node) => node.textContent);
    expect(titles[0]).toBe("Flex Everyday Tee");
  });

  it("paginates when results exceed one page", async () => {
    const products = Array.from({ length: 10 }).map((_, index) => ({
      id: `id-${index + 1}`,
      name: `Catalog Product ${index + 1}`,
      slug: `catalog-product-${index + 1}`,
      description: "Catalog item",
      price_cents: 1000 + index,
      currency: "USD",
      stock: 100,
      categories: ["Accessories"],
    }));

    mockProductsFetch(products);
    const user = userEvent.setup();
    renderCatalog();

    await screen.findByText("Catalog Product 1");
    expect(screen.getByText("Page 1 of 2")).toBeInTheDocument();
    expect(screen.queryByText("Catalog Product 9")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Next" }));
    expect(screen.getByText("Page 2 of 2")).toBeInTheDocument();
    expect(screen.getByText("Catalog Product 9")).toBeInTheDocument();
  });

  it("loads additional product pages beyond the first 100 results", async () => {
    const products = Array.from({ length: 120 }).map((_, index) => ({
      id: `bulk-${index + 1}`,
      name: `Bulk Product ${index + 1}`,
      slug: `bulk-product-${index + 1}`,
      description: "Bulk catalog item",
      price_cents: 1000 + index,
      currency: "USD",
      stock: 100,
      categories: ["Accessories"],
    }));

    const fetchSpy = vi.fn().mockImplementation((input: string | URL) => {
      const url = new URL(String(input), "http://localhost");
      const limit = Number.parseInt(url.searchParams.get("limit") ?? "100", 10);
      const offset = Number.parseInt(url.searchParams.get("offset") ?? "0", 10);
      const page = products.slice(offset, offset + limit);
      return Promise.resolve({
        ok: true,
        json: async () => ({
          data: page,
          limit,
          offset,
        }),
      });
    });
    vi.stubGlobal("fetch", fetchSpy);

    const user = userEvent.setup();
    renderCatalog();

    await screen.findByText("Bulk Product 1");
    await user.type(screen.getByPlaceholderText("Search by product or category..."), "Bulk Product 120");

    expect(await screen.findByText("Bulk Product 120")).toBeInTheDocument();
    expect(fetchSpy).toHaveBeenCalledTimes(2);
  });
});
