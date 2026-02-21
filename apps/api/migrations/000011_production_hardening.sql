ALTER TABLE users
  ADD CONSTRAINT users_email_trimmed_chk CHECK (email = btrim(email)),
  ADD CONSTRAINT users_email_format_chk CHECK (
    email ~* '^[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}$'
  );

ALTER TABLE orders
  ADD CONSTRAINT orders_shipping_required_when_shipped_chk CHECK (
    status <> 'shipped'
    OR (
      shipping_name IS NOT NULL
      AND shipping_address_line1 IS NOT NULL
      AND shipping_city IS NOT NULL
      AND shipping_postal_code IS NOT NULL
      AND shipping_country IS NOT NULL
    )
  );

ALTER TABLE payments
  ALTER COLUMN status SET DEFAULT 'pending',
  ADD CONSTRAINT payments_provider_payment_required_on_success_chk CHECK (
    status <> 'succeeded' OR provider_payment_id IS NOT NULL
  );

CREATE UNIQUE INDEX payments_provider_provider_payment_id_uq
  ON payments (provider, provider_payment_id)
  WHERE provider_payment_id IS NOT NULL;

CREATE UNIQUE INDEX payments_checkout_session_id_uq
  ON payments (checkout_session_id)
  WHERE checkout_session_id IS NOT NULL;

CREATE UNIQUE INDEX payments_payment_intent_id_uq
  ON payments (payment_intent_id)
  WHERE payment_intent_id IS NOT NULL;

CREATE OR REPLACE FUNCTION set_row_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS users_set_updated_at ON users;
CREATE TRIGGER users_set_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION set_row_updated_at();

DROP TRIGGER IF EXISTS products_set_updated_at ON products;
CREATE TRIGGER products_set_updated_at
BEFORE UPDATE ON products
FOR EACH ROW
EXECUTE FUNCTION set_row_updated_at();

DROP TRIGGER IF EXISTS carts_set_updated_at ON carts;
CREATE TRIGGER carts_set_updated_at
BEFORE UPDATE ON carts
FOR EACH ROW
EXECUTE FUNCTION set_row_updated_at();

DROP TRIGGER IF EXISTS orders_set_updated_at ON orders;
CREATE TRIGGER orders_set_updated_at
BEFORE UPDATE ON orders
FOR EACH ROW
EXECUTE FUNCTION set_row_updated_at();

DROP TRIGGER IF EXISTS payments_set_updated_at ON payments;
CREATE TRIGGER payments_set_updated_at
BEFORE UPDATE ON payments
FOR EACH ROW
EXECUTE FUNCTION set_row_updated_at();

