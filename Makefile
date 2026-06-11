.PHONY: migrate seed run dev test sqlc

DATABASE_URL ?= postgresql://stripe:stripe@localhost:5432/stripe_payment?sslmode=disable

migrate:
	cd backend && migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	cd backend && migrate -path migrations -database "$(DATABASE_URL)" down 1

seed:
	cd backend && go run ./cmd/seed

run:
	cd backend && go run ./cmd/server

dev:
	@echo "Starting backend (:8080) and frontend (:5173)..."
	@(cd backend && go run ./cmd/server) & \
	(cd frontend && npm run dev) & \
	wait

test:
	cd backend && go test ./...

sqlc:
	cd backend && sqlc generate
