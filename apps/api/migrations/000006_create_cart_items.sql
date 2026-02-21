CREATE TABLE IF NOT EXISTS cart_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  cart_id UUID NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
  product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
  quantity INTEGER NOT NULL CHECK (quantity > 0),
  price_cents_snapshot INTEGER NOT NULL CHECK (price_cents_snapshot >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS cart_items_cart_product_uq ON cart_items (cart_id, product_id);
CREATE INDEX IF NOT EXISTS cart_items_cart_id_idx ON cart_items (cart_id);
CREATE INDEX IF NOT EXISTS cart_items_product_id_idx ON cart_items (product_id);

