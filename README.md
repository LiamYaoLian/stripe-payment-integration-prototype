# Stripe Payment Integration Prototype

Full-stack prototype for one-time Stripe Checkout payments: Go API, Postgres, React frontend with **Hosted** and **Embedded** checkout flows.

## Prerequisites

- Docker
- Go 1.22+
- Node.js 20+
- [Stripe CLI](https://stripe.com/docs/stripe-cli)
- `golang-migrate` CLI (`brew install golang-migrate`)

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

In a separate terminal:

```bash
stripe listen --forward-to localhost:8080/api/webhooks/stripe
```

Copy the `whsec_...` secret printed by the CLI into `backend/.env` as `STRIPE_WEBHOOK_SECRET`, then restart the backend.

## Test card

Use Stripe test card `4242 4242 4242 4242`, any future expiry, any CVC.

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
| `make test` | Run Go tests |

## Project layout

- `backend/` — Go API (chi, pgx, stripe-go)
- `frontend/` — Vite + React + TypeScript
- `PLAN.md` — Full design spec
