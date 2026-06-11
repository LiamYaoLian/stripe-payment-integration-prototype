# Stripe Payment Integration Prototype

Go API + React frontend for one-time Stripe Checkout (hosted redirect and embedded on-site), with email/password accounts and order history. See [PLAN.md](PLAN.md) for design details.

## Prerequisites

Docker, Go 1.25+, Node.js 20+, [Stripe CLI](https://stripe.com/docs/stripe-cli)

## Quick start (dev)

```bash
docker compose up -d          # Postgres on host :5434
cp backend/.env.example backend/.env    # add STRIPE_SECRET_KEY (test mode)
cp frontend/.env.example frontend/.env  # add VITE_STRIPE_PUBLISHABLE_KEY
make migrate && make seed
make dev
```

Open [http://localhost:5173](http://localhost:5173). API listens on `:8080`.

## Production-like stack (Docker)

```bash
export STRIPE_SECRET_KEY=sk_test_...
export STRIPE_WEBHOOK_SECRET=whsec_...
export VITE_STRIPE_PUBLISHABLE_KEY=pk_test_...
make up-prod
```

Open [http://localhost:8081](http://localhost:8081). The web container serves the SPA and proxies `/api/` to the API (same-origin; `VITE_API_URL` is empty).

## Observability (optional)

```bash
make observability-up   # Jaeger UI :16686, OTLP HTTP :4318
```

In `backend/.env`, enable tracing:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
OTEL_SERVICE_NAME=stripe-payment-api
```

Restart the API (`make dev` or the `api` container). Traces appear in Jaeger.

## Webhooks

In another terminal:

```bash
stripe listen --forward-to localhost:8080/api/webhooks/stripe
```

Copy the printed `whsec_...` into `backend/.env` as `STRIPE_WEBHOOK_SECRET` and restart the API. Use the CLI secret (not Dashboard); it changes each `stripe listen` run.

## Test payment

Card `4242 4242 4242 4242`, any future expiry/CVC/ZIP. More cards: [Stripe testing](https://docs.stripe.com/testing#cards).

## Port map

| Service | Dev (`make dev`) | Prod stack (`make up-prod`) |
|---------|------------------|-----------------------------|
| Frontend | 5173 (Vite) | 8081 (nginx) |
| API | 8080 | 8080 |
| Postgres | 5434 (host) | internal |
| Jaeger UI | — | 16686 (`make observability-up`) |
| OTLP HTTP | — | 4318 |

## Commands

| Target | Action |
|--------|--------|
| `make dev` | API (`:8080`) + Vite (`:5173`) |
| `make up-prod` | Full Docker stack (web `:8081`, API `:8080`) |
| `make observability-up` | Jaeger + OTLP collector |
| `make test` | Unit tests (Go + frontend) |
| `make test-integration` | DB integration tests (Postgres required) |
| `make lint` | golangci-lint + ESLint |
| `make k8s-validate` | Validate Kubernetes manifests |
| `make release-build` | Build local API + web images |

## Accounts & order history

1. **Sign up** at [/signup](http://localhost:5173/signup) or **sign in** at [/login](http://localhost:5173/login).
2. Checkout from the catalog (hosted or embedded). After payment, the success page polls order status using a per-order `accessToken` (not your login session).
3. Open [/orders](http://localhost:5173/orders) for order history (requires login). Orders appear when `orders.customer_id` matches your account — checkout does not set this yet, so the list may be empty until that linkage is implemented.

**Dev-only account emails:** On sign-up, the API logs an email verification link. Password reset links are logged when you use [/forgot-password](http://localhost:5173/forgot-password). No SMTP is configured yet — copy the URL from the API terminal.

Auth uses an httpOnly `session` cookie. The Vite dev server proxies `/api` to `:8080` with `credentials: 'include'`.

## Kubernetes

Copy `deploy/kubernetes/configmap.example.yaml` → `configmap.yaml`, `secrets.example.yaml` → `secrets.yaml`, and optionally `ingress.example.yaml` → `ingress.yaml`. Build images (web needs `NGINX_CONF=nginx.k8s.conf` and empty `VITE_API_URL`). Then:

```bash
./scripts/k8s-deploy.sh
```
