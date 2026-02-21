# Database Schema Diagram

This document reflects the current backend schema under `apps/api/migrations`.

## ER Diagram

```mermaid
erDiagram
    USERS {
        uuid id PK
        text email "UNIQUE (lower)"
        text password_hash "nullable"
        text phone "nullable, UNIQUE when not null"
        text full_name
        text role
        boolean is_verified
        timestamptz last_login_at "nullable"
        timestamptz created_at
        timestamptz updated_at
    }

    PRODUCTS {
        uuid id PK
        text name
        text slug "UNIQUE (lower)"
        text description
        int price_cents
        text currency
        int stock
        boolean is_active
        text sku "UNIQUE"
        int weight_grams "nullable"
        text image_url "nullable"
        timestamptz created_at
        timestamptz updated_at
    }

    CATEGORIES {
        uuid id PK
        text name
        text slug "UNIQUE (lower)"
        uuid parent_id "nullable FK -> categories.id"
        timestamptz created_at
    }

    PRODUCT_CATEGORIES {
        uuid product_id "PK, FK -> products.id"
        uuid category_id "PK, FK -> categories.id"
    }

    CARTS {
        uuid id PK
        uuid user_id "nullable FK -> users.id"
        text session_id "nullable"
        text status
        timestamptz created_at
        timestamptz updated_at
    }

    CART_ITEMS {
        uuid id PK
        uuid cart_id "FK -> carts.id"
        uuid product_id "FK -> products.id"
        int quantity
        int price_cents_snapshot
        timestamptz created_at
    }

    ORDERS {
        uuid id PK
        uuid user_id "FK -> users.id"
        text status
        text currency
        int subtotal_cents
        int total_cents
        timestamptz created_at
        timestamptz updated_at
        text shipping_name "nullable"
        text shipping_address_line1 "nullable"
        text shipping_address_line2 "nullable"
        text shipping_city "nullable"
        text shipping_postal_code "nullable"
        text shipping_country "nullable"
    }

    ORDER_ITEMS {
        uuid id PK
        uuid order_id "FK -> orders.id"
        uuid product_id "FK -> products.id"
        text product_name_snapshot
        int price_cents_snapshot
        int quantity
        timestamptz created_at
    }

    PAYMENTS {
        uuid id PK
        uuid order_id "FK -> orders.id"
        text provider
        text provider_payment_id "nullable"
        int amount_cents
        text currency
        text status
        timestamptz created_at
        timestamptz updated_at
        text checkout_session_id "nullable"
        text payment_intent_id "nullable"
    }

    USERS ||--o{ CARTS : owns
    USERS ||--o{ ORDERS : places
    ORDERS ||--o{ ORDER_ITEMS : contains
    PRODUCTS ||--o{ ORDER_ITEMS : snapshotted_as

    CARTS ||--o{ CART_ITEMS : contains
    PRODUCTS ||--o{ CART_ITEMS : referenced_by

    ORDERS ||--o{ PAYMENTS : paid_by

    PRODUCTS ||--o{ PRODUCT_CATEGORIES : linked
    CATEGORIES ||--o{ PRODUCT_CATEGORIES : linked
    CATEGORIES ||--o{ CATEGORIES : parent_of
```

## Key Constraints

- `users.email` is unique case-insensitively (`LOWER(email)` index).
- `users.phone` is optional, must be normalized E.164 when present, and unique when not null.
- `products.slug` and `categories.slug` are unique case-insensitively.
- `product_categories` uses composite PK (`product_id`, `category_id`) to prevent duplicates.
- Cart ownership is exclusive: exactly one of `user_id` or `session_id` must be set.
- Only one active cart is allowed per user and per session.
- `orders.status` is constrained to: `pending`, `paid`, `failed`, `refunded`, `shipped`.
- Monetary fields enforce non-negative values; `orders.total_cents >= orders.subtotal_cents`.
- If `orders.status = shipped`, core shipping fields are required.
- `payments.status` is constrained to: `pending`, `succeeded`, `failed`, `refunded`.
- `payments` enforces idempotency with unique references on:
  - (`provider`, `provider_payment_id`) when payment id is present
  - `checkout_session_id` when present
  - `payment_intent_id` when present
