dev:
	docker compose up -d
	cd apps/api && go run ./cmd/server
	cd apps/web && npm run dev

api:
	cd apps/api && go run ./cmd/server

web:
	cd apps/web && npm run dev

build-api:
	cd apps/api && go build -o bin/server ./cmd/server

build-web:
	cd apps/web && npm run build