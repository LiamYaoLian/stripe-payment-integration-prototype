# Stripe Payment Integration Prototype

Go API + React frontend for one-time Stripe Checkout (hosted redirect and embedded on-site). See [PLAN.md](PLAN.md) for design details.

## Prerequisites

Docker, Go 1.22+, Node.js 20+, [Stripe CLI](https://stripe.com/docs/stripe-cli)

## Quick start

```bash
docker compose up -d
cp backend/.env.example backend/.env    # add STRIPE_SECRET_KEY (test mode)
cp frontend/.env.example frontend/.env  # add VITE_STRIPE_PUBLISHABLE_KEY
make migrate && make seed
make dev
```

Open [http://localhost:5173](http://localhost:5173).

## Webhooks

In another terminal:

```bash
stripe listen --forward-to localhost:8080/api/webhooks/stripe
```

Copy the printed `whsec_...` into `backend/.env` as `STRIPE_WEBHOOK_SECRET` and restart the API. Use the CLI secret (not Dashboard); it changes each `stripe listen` run.

## Test payment

Card `4242 4242 4242 4242`, any future expiry/CVC/ZIP. More cards: [Stripe testing](https://docs.stripe.com/testing#cards).

## Commands

| Target | Action |
|--------|--------|
| `make dev` | API (`:8080`) + Vite (`:5173`) |
| `make test` | Unit tests (Go + frontend) |
| `make test-integration` | DB integration tests (Postgres required) |
| `make lint` | golangci-lint + ESLint |
