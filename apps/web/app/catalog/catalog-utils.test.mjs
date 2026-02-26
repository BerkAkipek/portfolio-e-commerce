import { describe, expect, it } from "vitest";

import {
  buildCategoryOptions,
  filterAndSortProducts,
  getProductCategories,
  paginateProducts,
} from "./catalog-utils.js";

const sampleProducts = [
  {
    id: "p1",
    name: "Aero Running Jacket",
    slug: "aero-running-jacket",
    description: "Lightweight jacket for fast runs.",
    price_cents: 12900,
    currency: "USD",
    stock: 9,
    categories: ["Outerwear", "Training"],
  },
  {
    id: "p2",
    name: "Urban Carry Pack",
    slug: "urban-carry-pack",
    description: "Commuter backpack with organization.",
    price_cents: 7400,
    currency: "USD",
    stock: 13,
    categories: ["Accessories", "Lifestyle"],
  },
  {
    id: "p3",
    name: "Flex Everyday Tee",
    slug: "flex-everyday-tee",
    description: "Soft everyday t-shirt.",
    price_cents: 3200,
    currency: "USD",
    stock: 22,
    categories: ["Apparel"],
  },
];

describe("catalog utils", () => {
it("uses Uncategorized when categories are missing", () => {
  const categories = getProductCategories({
    id: "x",
    name: "No Category Product",
    slug: "no-category-product",
    description: "",
    price_cents: 1000,
    currency: "USD",
    stock: 1,
  });

  expect(categories).toEqual(["Uncategorized"]);
});

it("builds category filter options with All + unique categories", () => {
  const options = buildCategoryOptions(sampleProducts);
  expect(options).toEqual([
    "All",
    "Outerwear",
    "Training",
    "Accessories",
    "Lifestyle",
    "Apparel",
  ]);
});

it("supports category-aware search", () => {
  const filtered = filterAndSortProducts(
    sampleProducts,
    "All",
    "outerwear",
    "newest",
  );

  expect(filtered).toHaveLength(1);
  expect(filtered[0].slug).toBe("aero-running-jacket");
});

it("supports filtering by selected category", () => {
  const filtered = filterAndSortProducts(
    sampleProducts,
    "Accessories",
    "",
    "newest",
  );

  expect(filtered).toHaveLength(1);
  expect(filtered[0].slug).toBe("urban-carry-pack");
});

it("supports sorting by price ascending", () => {
  const sorted = filterAndSortProducts(sampleProducts, "All", "", "price-asc");
  expect(sorted.map((item) => item.slug)).toEqual([
    "flex-everyday-tee",
    "urban-carry-pack",
    "aero-running-jacket",
  ]);
});

it("supports sorting by name descending", () => {
  const sorted = filterAndSortProducts(sampleProducts, "All", "", "name-desc");
  expect(sorted.map((item) => item.name)).toEqual([
    "Urban Carry Pack",
    "Flex Everyday Tee",
    "Aero Running Jacket",
  ]);
});

it("paginates products and clamps out-of-range page values", () => {
  const page1 = paginateProducts(sampleProducts, 1, 2);
  expect(page1.totalPages).toBe(2);
  expect(page1.page).toBe(1);
  expect(page1.items.map((item) => item.slug)).toEqual([
    "aero-running-jacket",
    "urban-carry-pack",
  ]);

  const clamped = paginateProducts(sampleProducts, 99, 2);
  expect(clamped.page).toBe(2);
  expect(clamped.items.map((item) => item.slug)).toEqual(["flex-everyday-tee"]);
});
});
