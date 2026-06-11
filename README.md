# Stripe Payment Integration Prototype

Full-stack prototype for one-time Stripe Checkout payments: Go API, Postgres, React frontend with **Hosted** and **Embedded** checkout flows.

## Prerequisites

- Docker
- Go 1.22+
- Node.js 20+
- [Stripe CLI](https://stripe.com/docs/stripe-cli)
- Docker (also used to run DB migrations — no separate `migrate` CLI needed)

## Quick start

```bash
# 1. Start Postgres
docker compose up -d

# 2. Configure backend
cp backend/.env.example backend/.env
# Fill STRIPE_SECRET_KEY from Stripe Dashboard (test mode)

# 3. Schema + sample products
make migrate && make seed

# 4. Frontend env (embedded checkout needs publishable key)
cp frontend/.env.example frontend/.env

# 5. Run API + frontend
make dev
```

Open [http://localhost:5173](http://localhost:5173).

## Webhooks

Install and authenticate the Stripe CLI (once):

```bash
brew install stripe/stripe-cli/stripe
stripe login
```

In a separate terminal:

```bash
stripe listen --forward-to localhost:8080/api/webhooks/stripe
```

Copy the `whsec_...` secret printed by the CLI into `backend/.env` as `STRIPE_WEBHOOK_SECRET`, then restart the backend.

> **Note:** Use the secret from `stripe listen` output — not the Dashboard webhook secret. It changes each time you restart `stripe listen`.

## Pay with a test card

Use **test mode** keys (`pk_test_...`, `sk_test_...`) in `backend/.env` and `frontend/.env`.

1. Start the app (`make dev`) and forward webhooks (`stripe listen --forward-to localhost:8080/api/webhooks/stripe`).
2. Open [http://localhost:5173](http://localhost:5173) and click **Pay (Hosted)** or **Pay (Embedded)** on a product.
3. On the Stripe Checkout form, enter:

| Field | Value |
|-------|-------|
| Card number | `4242 4242 4242 4242` |
| Expiry | Any future date (e.g. `12/34`) |
| CVC | Any 3 digits (e.g. `123`) |
| ZIP / postal | Any (e.g. `12345`) |

4. Complete payment — the success/complete page should show order status **`paid`**.

**Other test cards**

| Card | Result |
|------|--------|
| `4242 4242 4242 4242` | Success |
| `4000 0000 0000 0002` | Declined |
| `4000 0025 0000 3155` | Requires 3D Secure |

More: [Stripe test cards](https://docs.stripe.com/testing#cards)

## Flows

| Flow | Path |
|------|------|
| Hosted (redirect) | Catalog → Pay (Hosted) → Stripe → `/checkout/success` |
| Embedded (on-site) | Catalog → Pay (Embedded) → `/checkout/complete` |

Both flows poll order status via `GET /api/orders/by-session/:sessionId` after payment.

## API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check (DB ping) |
| GET | `/api/products` | List active products |
| POST | `/api/checkout/sessions` | Create Checkout Session (`Idempotency-Key` header) |
| GET | `/api/orders/:id` | Get order by ID |
| GET | `/api/orders/by-session/:sessionId` | Get order by Stripe session ID |
| POST | `/api/webhooks/stripe` | Stripe webhook endpoint |

## Makefile

| Target | Action |
|--------|--------|
| `make migrate` | Run migrations |
| `make seed` | Insert sample products |
| `make run` | Start Go API only (`:8080`) |
| `make dev` | Start API + Vite (`:5173`) |
| `make test` | Unit tests (Go `-short` + frontend Vitest) |
| `make test-integration` | DB integration tests (requires Postgres on `:5434`) |
| `make test-frontend` | Frontend Vitest only |
| `make test-all` | All Go tests + frontend (includes integration) |

## Testing

```bash
# Fast unit tests (default — no Postgres required)
make test

# DB integration tests (docker compose up -d first)
make test-integration

# Everything
make test-all
```

**Structure**
- **Go unit tests** — service/handler logic with fakes in `internal/testutil`
- **Go integration tests** — `internal/db` against Postgres; skipped with `-short`
- **Frontend tests** — Vitest + Testing Library (`frontend/src/**/*.test.ts(x)`)

## Project layout

- `backend/` — Go API (chi, pgx, stripe-go)
- `frontend/` — Vite + React + TypeScript
- `PLAN.md` — Full design spec
