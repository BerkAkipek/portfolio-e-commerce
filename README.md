# E-Commerce Platform (Monorepo)

Production-minded e-commerce application built with a Go API, PostgreSQL, and a Next.js frontend.

This project demonstrates backend-heavy architecture and data integrity design for real-world commerce flows (catalog, cart, checkout, orders, and payments).

## Why This Project

- Built a modular monorepo with separate API and web apps.
- Designed a relational schema with strict constraints and migration-first evolution.
- Implemented actor-aware carts for both guests and authenticated users.
- Integrated Stripe checkout + idempotent webhook processing.
- Added solid backend test coverage for service and HTTP layers.

## Tech Stack

- Backend: Go, `chi`, `pgx`, `sqlc`, `net/http`
- Database: PostgreSQL 16
- Frontend: Next.js 16, React 19, TypeScript, Tailwind CSS 4
- Infrastructure: Docker Compose, multi-stage Docker builds, Makefile automation

## Architecture

- `web` (Next.js): UI app on `http://localhost:3000`
- `api` (Go): REST API on `http://localhost:8080`
- `db` (PostgreSQL): persistent data store
- `migrate`: one-shot migration runner for SQL files in `apps/api/migrations`

High-level flow:
1. User browses catalog and adds items to cart.
2. Cart ownership is resolved by authenticated `user_id` or guest `session_id`.
3. Authenticated user creates Stripe checkout session.
4. Stripe webhook finalizes order/payment and converts cart idempotently.

## Core Features

- Product catalog with category-aware data, search/sort/filter/pagination UI
- Product detail page with image gallery, stock status, and add-to-cart
- Cart page with quantity updates, remove item, subtotal, empty state, and guest-cart messaging
- Authentication (`register`, `login`, `logout`) with cookie-based tokens
- Google Sign-In via backend OAuth authorization code flow
- Session restoration UX (silent refresh-token rotation surfaced in UI)
- Account security hint UI (short-lived access token, silent refresh, logout revocation)
- Guest + authenticated cart lifecycle and ownership protections
- Stripe Checkout redirect flow from cart
- Checkout success + cancel pages
- Authenticated order history (`/orders`) and order details (`/orders/{id}`)
- Receipt page for eligible orders (`/orders/{id}/receipt`)
- Webhook verification + order, order_items, payment creation
- Strong database integrity constraints (uniqueness, check constraints, state modeling)

## Currently Testable Functionality

- Auth:
  - Register with email/password
  - Login with email/password
  - Login with Google (`/api/auth/google/start`)
  - Logout with refresh-token revocation
  - Session restoration banner when refresh rotation occurs
- Catalog & Product:
  - Browse catalog
  - Search/filter/sort/paginate
  - Open product detail pages
  - Add products to cart
- Cart:
  - View guest cart
  - Update item quantity
  - Remove item
  - Start Stripe checkout from cart
- Checkout:
  - Redirect to Stripe Checkout session URL
  - Land on `/checkout/success` after successful payment flow
  - Land on `/checkout/cancel` when checkout is canceled
- Orders:
  - View authenticated order list (`/orders`)
  - View authenticated order details (`/orders/{id}`)
  - View receipt (`/orders/{id}/receipt`) for paid/completed statuses

## Repository Structure

```text
.
├── apps/
│   ├── api/                  # Go API, SQL migrations, sqlc queries, tests
│   └── web/                  # Next.js app
├── docs/
│   └── schema-diagram.md     # ER diagram and constraints
├── deployments/
│   ├── Dockerfile.api
│   └── Dockerfile.web
├── packages/
│   └── contracts/            # Shared contract scaffold
├── docker-compose.yml
└── Makefile
```

## API Endpoints (Sample)

Available at both root and `/api` prefix:

- `GET /health`
- `GET /products?limit=&offset=`
- `GET /products/{slug}`
- `POST /auth/register`
- `POST /auth/login`
- `GET /auth/google/start`
- `GET /auth/google/callback`
- `POST /auth/logout`
- `POST /cart/items`
- `PATCH /cart/items/{id}`
- `DELETE /cart/items/{id}`
- `GET /cart/{id}/items`
- `POST /checkout/session`
- `GET /orders`
- `GET /orders/{id}`
- `POST /webhooks/stripe`

Web proxy endpoints (`apps/web`) for same-origin frontend calls:
- `GET /api/products`
- `GET /api/products/{slug}`
- `POST /api/auth/login`
- `POST /api/auth/register`
- `POST /api/auth/logout`
- `GET /api/auth/session`
- `GET /api/auth/google/start`
- `GET /api/auth/google/callback`
- `POST /api/checkout/session`
- `GET /api/orders`
- `GET /api/orders/{id}`
- `POST /api/cart/items`
- `GET /api/cart/{cartId}/items`
- `PATCH /api/cart/items/{itemId}`
- `DELETE /api/cart/items/{itemId}`

## Frontend Routes

- `/` Home page
- `/catalog` Catalog page (search/filter/sort/pagination + loading skeletons)
- `/products/{slug}` Product detail page
- `/cart` Cart page
- `/checkout/success` Checkout success page
- `/checkout/cancel` Checkout cancel page
- `/orders` Authenticated order history page
- `/orders/{id}` Authenticated order details page
- `/orders/{id}/receipt` Receipt view (when payment status allows)

## Local Development

### Option 1: Docker Compose (recommended)

```bash
make up
```

Services:
- Web: `http://localhost:3000`
- API: `http://localhost:8080`
- Postgres: `localhost:5432`

Useful commands:

```bash
make ps
make logs
make logs-api
make logs-web
make logs-db
make health
make check-api
make check-web
make down
make clean
```

Seed realistic dev catalog data (categories + products + mappings):

```bash
make seed-dev-catalog
```

### Option 2: Run components locally

API:

```bash
cd apps/api
DATABASE_URL=postgres://postgres:postgres@localhost:5432/ecommerce?sslmode=disable PORT=8080 go run ./cmd/server
```

Web:

```bash
cd apps/web
npm run dev
```

## Environment Variables

API (`apps/api`):
- Required: `DATABASE_URL`
- Optional: `PORT`, `JWT_SECRET`, `COOKIE_SECURE`, `STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET`, `CHECKOUT_SUCCESS_URL`, `CHECKOUT_CANCEL_URL`, `GOOGLE_OAUTH_CLIENT_ID`, `GOOGLE_OAUTH_CLIENT_SECRET`, `GOOGLE_OAUTH_REDIRECT_URL`, `GOOGLE_OAUTH_POST_LOGIN_URL`, `GOOGLE_OAUTH_AUTH_URL`, `GOOGLE_OAUTH_TOKEN_URL`, `GOOGLE_OAUTH_USERINFO_URL`

Web (`apps/web`):
- `NEXT_PUBLIC_API_URL`
- `PORT`

## Testing

Backend tests:

```bash
make test-api
```

Or directly:

```bash
cd apps/api
go test ./...
```

Integration test note:
- `apps/api/internal/db/integration_test.go` requires `TEST_DATABASE_URL`.

Frontend UI tests (Vitest + Testing Library):

```bash
npm run test:ui
```

Or directly:

```bash
cd apps/web
npm run test:ui
```

Latest local validation:
- `apps/api/internal/httpapi`: tests passing
- `apps/web`: lint passing, UI tests passing

## Deployment Notes

- API and web use multi-stage Dockerfiles for smaller production images.
- Schema migrations are idempotent and tracked in `schema_migrations`.
- Checkout webhook handling is idempotent to prevent duplicate orders/payments.

## Portfolio Notes

If you are reviewing this project for hiring purposes, focus areas are:
- Data modeling and DB constraints
- Backend service design and request ownership rules
- Checkout/payment correctness and idempotency
- Testability and maintainable monorepo structure
