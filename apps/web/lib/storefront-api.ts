export type Product = {
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

export type ProductsResponse = {
  data: Product[];
  limit: number;
  offset: number;
};

export type AddToCartResponse = {
  data?: { cart_id?: string };
  error?: string;
};

export async function fetchProducts(limit = 4, offset = 0): Promise<ProductsResponse> {
  const response = await fetch(
    `/api/products?limit=${encodeURIComponent(String(limit))}&offset=${encodeURIComponent(String(offset))}`,
    { cache: "no-store" },
  );

  if (!response.ok) {
    throw new Error("failed to load products");
  }

  return (await response.json()) as ProductsResponse;
}

export async function addToCart(productID: string, quantity: number): Promise<AddToCartResponse> {
  const response = await fetch("/api/cart/items", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ product_id: productID, quantity }),
  });

  const payload = (await response.json()) as AddToCartResponse;
  if (!response.ok) {
    throw new Error(payload.error || "failed to add to cart");
  }

  return payload;
}
