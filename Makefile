.PHONY: dev up build rebuild down stop restart ps logs logs-api logs-web logs-db logs-migrate \
	seed-products seed-dev-catalog db-shell migrate-status health check-api check-web clean \
	api web build-api build-web test-api test-api-verbose test-api-smoke

dev: up

up:
	docker compose up -d --build

build:
	docker compose build

rebuild:
	docker compose up -d --build

down:
	docker compose down

stop:
	docker compose stop

restart:
	docker compose restart

ps:
	docker compose ps

logs:
	docker compose logs -f

logs-api:
	docker compose logs -f api

logs-web:
	docker compose logs -f web

logs-db:
	docker compose logs -f db

logs-migrate:
	docker compose logs -f migrate

db-shell:
	docker compose exec db psql postgresql://postgres:postgres@localhost:5432/ecommerce

migrate-status:
	docker compose exec db psql postgresql://postgres:postgres@localhost:5432/ecommerce -c "SELECT version, applied_at FROM schema_migrations ORDER BY version;"

seed-products:
	docker compose exec -T db psql postgresql://postgres:postgres@localhost:5432/ecommerce -c "INSERT INTO products (name, slug, description, price_cents, currency, stock, is_active, sku) VALUES ('Basic Tee','basic-tee','Simple t-shirt',1999,'USD',10,TRUE,'TEE-001') ON CONFLICT DO NOTHING;"

seed-dev-catalog:
	cat scripts/seed_dev_catalog.sql | docker compose exec -T db psql postgresql://postgres:postgres@localhost:5432/ecommerce -v ON_ERROR_STOP=1

health:
	curl -i http://localhost:8080/health

check-api:
	curl -i http://localhost:8080/products

check-web:
	curl -I http://localhost:3000

clean:
	docker compose down -v

api:
	cd apps/api && go run ./cmd/server

web:
	cd apps/web && npm run dev

build-api:
	cd apps/api && go build -o bin/server ./cmd/server

build-web:
	cd apps/web && npm run build

test-api:
	cd apps/api && GOCACHE=/tmp/go-build-cache go test ./...

test-api-verbose:
	cd apps/api && GOCACHE=/tmp/go-build-cache go test -v ./...

test-api-smoke:
	cd apps/api && GOCACHE=/tmp/go-build-cache go test -v ./internal/httpapi -run Smoke
