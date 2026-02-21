-- name: CreatePayment :one
INSERT INTO payments (
  order_id,
  provider,
  provider_payment_id,
  amount_cents,
  currency,
  status,
  checkout_session_id,
  payment_intent_id
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetPaymentByID :one
SELECT * FROM payments
WHERE id = $1
LIMIT 1;

-- name: GetPaymentByProviderPaymentID :one
SELECT * FROM payments
WHERE provider_payment_id = sqlc.arg(provider_payment_id)::text
LIMIT 1;

-- name: GetPaymentByCheckoutSessionID :one
SELECT * FROM payments
WHERE checkout_session_id = sqlc.arg(checkout_session_id)::text
LIMIT 1;

-- name: GetPaymentByPaymentIntentID :one
SELECT * FROM payments
WHERE payment_intent_id = sqlc.arg(payment_intent_id)::text
LIMIT 1;

-- name: ListPaymentsByOrderID :many
SELECT * FROM payments
WHERE order_id = $1
ORDER BY created_at DESC;

-- name: UpdatePaymentStatus :one
UPDATE payments
SET
  status = $2,
  updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: SetProviderPaymentID :one
UPDATE payments
SET
  provider_payment_id = $2,
  updated_at = NOW()
WHERE id = $1
RETURNING *;
