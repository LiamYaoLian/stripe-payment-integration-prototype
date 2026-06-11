.PHONY: migrate migrate-release seed run dev up-prod observability-up test test-integration test-frontend test-all lint lint-backend lint-frontend k8s-validate release-build

DATABASE_URL ?= postgresql://stripe:stripe@localhost:5434/stripe_payment?sslmode=disable
# host.docker.internal lets the migrate container reach Postgres on the Mac host
MIGRATE_DATABASE_URL ?= postgresql://stripe:stripe@host.docker.internal:5434/stripe_payment?sslmode=disable
MIGRATE = docker run --rm -v "$(CURDIR)/backend/migrations:/migrations" migrate/migrate:v4.18.2

migrate:
	$(MIGRATE) -path=/migrations -database "$(MIGRATE_DATABASE_URL)" up

migrate-down:
	$(MIGRATE) -path=/migrations -database "$(MIGRATE_DATABASE_URL)" down 1

migrate-release:
	DATABASE_URL="$(DATABASE_URL)" ./scripts/migrate-release.sh

seed:
	cd backend && DATABASE_URL="$(DATABASE_URL)" go run ./cmd/seed

run:
	cd backend && DATABASE_URL="$(DATABASE_URL)" go run ./cmd/server

up-prod:
	docker compose -f docker-compose.prod.yml up --build -d

observability-up:
	docker compose -f docker-compose.observability.yml up -d
	@echo "Jaeger UI: http://localhost:16686 — set OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318"

dev:
	@echo "Starting backend (:8080)..."
	@(cd backend && DATABASE_URL="$(DATABASE_URL)" go run ./cmd/server) & \
	until curl -sf http://localhost:8080/health/ready >/dev/null 2>&1; do sleep 0.5; done; \
	echo "Starting frontend (:5173)..."; \
	(cd frontend && npm run dev) & \
	wait

test:
	cd backend && go test -short ./...
	cd frontend && npm test

test-integration:
	cd backend && go test ./internal/db/...

test-frontend:
	cd frontend && npm test

test-all:
	cd backend && go test ./...
	cd frontend && npm test

lint: lint-backend lint-frontend

lint-backend:
	cd backend && golangci-lint run ./...

lint-frontend:
	cd frontend && npm run lint

k8s-validate:
	docker run --rm -v "$(CURDIR):/work" ghcr.io/yannh/kubeconform:latest -summary /work/deploy/kubernetes

release-build:
	docker build -t stripe-payment-api:local ./backend
	docker build \
		--build-arg VITE_API_URL= \
		--build-arg NGINX_CONF=nginx.k8s.conf \
		--build-arg VITE_STRIPE_PUBLISHABLE_KEY=pk_test_local \
		-t stripe-payment-web:local ./frontend
