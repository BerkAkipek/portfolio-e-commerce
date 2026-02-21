ALTER TABLE users
  ADD CONSTRAINT users_email_not_blank_chk CHECK (length(btrim(email)) > 0),
  ADD CONSTRAINT users_full_name_not_blank_chk CHECK (length(btrim(full_name)) > 0);

ALTER TABLE products
  ADD CONSTRAINT products_name_not_blank_chk CHECK (length(btrim(name)) > 0),
  ADD CONSTRAINT products_slug_not_blank_chk CHECK (length(btrim(slug)) > 0),
  ADD CONSTRAINT products_sku_not_blank_chk CHECK (length(btrim(sku)) > 0),
  ADD CONSTRAINT products_currency_format_chk CHECK (currency ~ '^[A-Z]{3}$');

ALTER TABLE categories
  ADD CONSTRAINT categories_name_not_blank_chk CHECK (length(btrim(name)) > 0),
  ADD CONSTRAINT categories_slug_not_blank_chk CHECK (length(btrim(slug)) > 0),
  ADD CONSTRAINT categories_parent_not_self_chk CHECK (parent_id IS NULL OR parent_id <> id);

ALTER TABLE carts
  ADD CONSTRAINT carts_exactly_one_owner_chk CHECK (
    (user_id IS NOT NULL AND session_id IS NULL)
    OR (user_id IS NULL AND session_id IS NOT NULL)
  );

CREATE UNIQUE INDEX carts_one_active_user_cart_uq
  ON carts (user_id)
  WHERE user_id IS NOT NULL AND status = 'active';

CREATE UNIQUE INDEX carts_one_active_session_cart_uq
  ON carts (session_id)
  WHERE session_id IS NOT NULL AND status = 'active';

ALTER TABLE orders
  ALTER COLUMN status SET DEFAULT 'pending',
  ADD CONSTRAINT orders_currency_format_chk CHECK (currency ~ '^[A-Z]{3}$'),
  ADD CONSTRAINT orders_total_gte_subtotal_chk CHECK (total_cents >= subtotal_cents);

