export const PRODUCTS_PER_PAGE = 8;

export function getProductCategories(product) {
  if (!product.categories || product.categories.length === 0) {
    return ["Uncategorized"];
  }
  return product.categories;
}

export function buildCategoryOptions(products) {
  const set = new Set(["All"]);
  products.forEach((product) => {
    getProductCategories(product).forEach((category) => set.add(category));
  });
  return Array.from(set);
}

export function filterAndSortProducts(products, selectedCategory, search, sortBy) {
  const normalizedSearch = search.trim().toLowerCase();

  const filtered = products.filter((product) => {
    const productCategories = getProductCategories(product);
    const categoryMatch =
      selectedCategory === "All" || productCategories.includes(selectedCategory);
    const searchMatch =
      normalizedSearch === "" ||
      product.name.toLowerCase().includes(normalizedSearch) ||
      product.description.toLowerCase().includes(normalizedSearch) ||
      product.slug.toLowerCase().includes(normalizedSearch) ||
      productCategories.some((category) =>
        category.toLowerCase().includes(normalizedSearch),
      );
    return categoryMatch && searchMatch;
  });

  filtered.sort((a, b) => {
    if (sortBy === "price-asc") return a.price_cents - b.price_cents;
    if (sortBy === "price-desc") return b.price_cents - a.price_cents;
    if (sortBy === "name-asc") return a.name.localeCompare(b.name);
    if (sortBy === "name-desc") return b.name.localeCompare(a.name);
    return 0;
  });

  return filtered;
}

export function paginateProducts(products, page, perPage = PRODUCTS_PER_PAGE) {
  const totalPages = Math.max(1, Math.ceil(products.length / perPage));
  const safePage = Math.min(Math.max(1, page), totalPages);
  const start = (safePage - 1) * perPage;
  return {
    items: products.slice(start, start + perPage),
    page: safePage,
    totalPages,
  };
}
