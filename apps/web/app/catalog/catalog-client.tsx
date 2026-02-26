"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useMemo, useState } from "react";
import {
  buildCategoryOptions,
  filterAndSortProducts,
  getProductCategories,
  paginateProducts,
  PRODUCTS_PER_PAGE,
} from "./catalog-utils";

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

type ProductsResponse = {
  data: Product[];
  limit: number;
  offset: number;
};

function formatPrice(priceCents: number, currency: string): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: (currency || "USD").toUpperCase(),
  }).format(priceCents / 100);
}

async function fetchProducts(): Promise<ProductsResponse> {
  const response = await fetch("/api/products?limit=100&offset=0");
  if (!response.ok) {
    throw new Error("failed to load products");
  }
  return (await response.json()) as ProductsResponse;
}

function ProductSkeletonCard() {
  return (
    <div className="rounded-3xl border border-white/10 bg-white/5 p-4">
      <div className="h-40 animate-pulse rounded-2xl bg-white/10" />
      <div className="mt-4 h-3 w-20 animate-pulse rounded bg-white/10" />
      <div className="mt-3 h-4 w-36 animate-pulse rounded bg-white/10" />
      <div className="mt-5 h-8 w-full animate-pulse rounded-full bg-white/10" />
    </div>
  );
}

export default function CatalogClient() {
  const [search, setSearch] = useState("");
  const [selectedCategory, setSelectedCategory] = useState("All");
  const [sortBy, setSortBy] = useState("newest");
  const [page, setPage] = useState(1);

  const { data, isLoading, isError } = useQuery({
    queryKey: ["products-catalog"],
    queryFn: fetchProducts,
  });

  const categories = useMemo<string[]>(() => {
    return buildCategoryOptions(data?.data ?? []);
  }, [data]);

  const filteredAndSorted = useMemo<Product[]>(() => {
    return filterAndSortProducts(
      [...(data?.data ?? [])],
      selectedCategory,
      search,
      sortBy,
    );
  }, [data, search, selectedCategory, sortBy]);

  const pagination = useMemo(() => {
    return paginateProducts(filteredAndSorted, page, PRODUCTS_PER_PAGE) as {
      items: Product[];
      page: number;
      totalPages: number;
    };
  }, [filteredAndSorted, page]);
  const totalPages = pagination.totalPages;
  const currentPage = pagination.page;
  const paginatedProducts = pagination.items;

  const handleFilterChange = (category: string) => {
    setSelectedCategory(category);
    setPage(1);
  };

  const handleSearchChange = (value: string) => {
    setSearch(value);
    setPage(1);
  };

  const handleSortChange = (value: string) => {
    setSortBy(value);
    setPage(1);
  };

  return (
    <main className="relative mx-auto w-full max-w-6xl px-6 pb-14 pt-8 sm:px-10 lg:px-12">
      <div className="mb-10 flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="text-xs font-semibold tracking-[0.2em] text-cyan-200 uppercase">
            Catalog
          </p>
          <h1 className="mt-2 text-3xl font-semibold text-white sm:text-4xl">
            Browse Products
          </h1>
        </div>
        <Link
          href="/cart"
          className="rounded-full border border-white/20 px-4 py-2 text-sm text-slate-200 transition hover:border-cyan-300/80 hover:text-white"
        >
          Go to cart
        </Link>
      </div>

      <section className="mb-8 grid gap-4 rounded-3xl border border-white/10 bg-white/5 p-4 sm:grid-cols-2 lg:grid-cols-[2fr_1fr]">
        <input
          value={search}
          onChange={(event) => handleSearchChange(event.target.value)}
          placeholder="Search by product or category..."
          className="h-11 rounded-xl border border-white/15 bg-black/30 px-4 text-sm text-white outline-none placeholder:text-slate-400 focus:border-cyan-300/70"
        />
        <select
          value={sortBy}
          onChange={(event) => handleSortChange(event.target.value)}
          className="h-11 rounded-xl border border-white/15 bg-black/30 px-4 text-sm text-white outline-none focus:border-cyan-300/70"
        >
          <option value="newest">Sort: Newest</option>
          <option value="price-asc">Price: Low to High</option>
          <option value="price-desc">Price: High to Low</option>
          <option value="name-asc">Name: A to Z</option>
          <option value="name-desc">Name: Z to A</option>
        </select>
      </section>

      <section className="mb-8 flex flex-wrap gap-2">
        {categories.map((category) => (
          <button
            key={category}
            type="button"
            onClick={() => handleFilterChange(category)}
            className={`rounded-full border px-4 py-1.5 text-xs font-semibold tracking-wide uppercase transition ${
              selectedCategory === category
                ? "border-cyan-300/90 bg-cyan-300/20 text-cyan-100"
                : "border-white/15 text-slate-300 hover:border-white/30 hover:text-white"
            }`}
          >
            {category}
          </button>
        ))}
      </section>

      {isError ? (
        <div className="rounded-2xl border border-red-300/30 bg-red-400/10 p-4 text-sm text-red-100">
          Unable to load catalog right now. Please try again.
        </div>
      ) : null}

      {isLoading ? (
        <section className="grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
          {Array.from({ length: PRODUCTS_PER_PAGE }).map((_, index) => (
            <ProductSkeletonCard key={index} />
          ))}
        </section>
      ) : (
        <section className="grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
          {paginatedProducts.map((product) => (
            <article key={product.id} className="rounded-3xl border border-white/12 bg-white/5 p-4">
              <div className="mb-4 flex h-40 items-center justify-center rounded-2xl border border-white/10 bg-gradient-to-br from-cyan-400/20 to-emerald-400/10">
                <span className="text-xs font-medium tracking-[0.15em] text-slate-300 uppercase">
                  {getProductCategories(product)[0]}
                </span>
              </div>
              <p className="text-xs tracking-[0.14em] text-slate-400 uppercase">
                {getProductCategories(product).join(" · ")}
              </p>
              <h2 className="mt-1 line-clamp-2 min-h-12 text-base font-semibold text-white">
                <Link
                  href={`/products/${product.slug}`}
                  className="transition hover:text-cyan-200"
                >
                  {product.name}
                </Link>
              </h2>
              <p className="mt-2 line-clamp-2 text-sm text-slate-300">
                {product.description || "Premium product from our latest collection."}
              </p>
              <div className="mt-4 flex items-center justify-between">
                <p className="text-sm font-semibold text-cyan-200">
                  {formatPrice(product.price_cents, product.currency)}
                </p>
                <Link
                  href={`/products/${product.slug}`}
                  className="rounded-full border border-white/15 px-3 py-1.5 text-xs font-medium text-slate-200 transition hover:border-cyan-300/70 hover:text-white"
                >
                  View details
                </Link>
              </div>
            </article>
          ))}
        </section>
      )}

      <section className="mt-8 flex items-center justify-between">
        <p className="text-sm text-slate-300">
          Showing{" "}
          {filteredAndSorted.length === 0
            ? 0
            : (currentPage - 1) * PRODUCTS_PER_PAGE + 1}
          -{Math.min(currentPage * PRODUCTS_PER_PAGE, filteredAndSorted.length)} of{" "}
          {filteredAndSorted.length}
        </p>
        <div className="flex items-center gap-2">
          <button
            type="button"
            disabled={currentPage <= 1}
            onClick={() => setPage((prev) => Math.max(1, prev - 1))}
            className="rounded-full border border-white/20 px-4 py-2 text-sm text-slate-200 transition enabled:hover:border-cyan-300/80 enabled:hover:text-white disabled:cursor-not-allowed disabled:opacity-40"
          >
            Previous
          </button>
          <span className="text-sm text-slate-300">
            Page {currentPage} of {totalPages}
          </span>
          <button
            type="button"
            disabled={currentPage >= totalPages}
            onClick={() => setPage((prev) => Math.min(totalPages, prev + 1))}
            className="rounded-full border border-white/20 px-4 py-2 text-sm text-slate-200 transition enabled:hover:border-cyan-300/80 enabled:hover:text-white disabled:cursor-not-allowed disabled:opacity-40"
          >
            Next
          </button>
        </div>
      </section>
    </main>
  );
}
