CREATE TABLE IF NOT EXISTS payments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  provider TEXT NOT NULL CHECK (provider IN ('stripe', 'paypal')),
  provider_payment_id TEXT,
  amount_cents INTEGER NOT NULL CHECK (amount_cents >= 0),
  currency TEXT NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
  status TEXT NOT NULL CHECK (status IN ('pending', 'succeeded', 'failed', 'refunded')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  checkout_session_id TEXT,
  payment_intent_id TEXT
);

CREATE INDEX IF NOT EXISTS payments_order_id_idx ON payments (order_id);
CREATE INDEX IF NOT EXISTS payments_provider_payment_id_idx ON payments (provider_payment_id);
CREATE INDEX IF NOT EXISTS payments_checkout_session_id_idx ON payments (checkout_session_id);
CREATE INDEX IF NOT EXISTS payments_payment_intent_id_idx ON payments (payment_intent_id);
CREATE INDEX IF NOT EXISTS payments_status_idx ON payments (status);
