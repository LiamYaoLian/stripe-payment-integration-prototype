# Stripe Payment Integration — Design

One-time card payments via **Stripe Checkout**. Stack: **Go API** (`:8080`) · **React** (`:5173`) · **Postgres** · **Stripe** (`2026-05-27.dahlia`, Checkout Sessions only).

**Golden rule:** Prices come from **Postgres**, not the browser. Client sends only `productId` + `quantity`.

| Checkout mode | UX | Frontend uses |
|---------------|-----|---------------|
| Hosted | Redirect to Stripe | `session.url` |
| Embedded | Pay on your page | `session.clientSecret` |

---

## How a payment works

Two paths run in parallel:

| Path | Driver | Role |
|------|--------|------|
| **User** | Browser | Checkout UX; polls order status on success page |
| **Webhook** | Stripe → API | **Source of truth** — flips order to `paid` |

```mermaid
sequenceDiagram
    participant FE as React
    participant API as Go API
    participant DB as Postgres
    participant Stripe

    FE->>API: POST /api/checkout/sessions + Idempotency-Key
    API->>DB: pending order
    API->>Stripe: checkout.sessions.create
    API-->>FE: sessionId, url/clientSecret, accessToken

    FE->>Stripe: Pay (hosted redirect or embedded widget)
    Stripe->>API: webhook checkout.session.completed
    API->>DB: order → paid

    FE->>API: GET /api/orders/by-session/:id + X-Order-Token
    API-->>FE: status paid
```

1. API inserts `pending` order, creates Stripe session, returns `accessToken`.
2. User pays on Stripe.
3. Webhook verifies signature, dedupes by `stripe_event_id`, updates order atomically.
4. Frontend polls every ~1s (≤30s) for UX — **webhook decides payment truth**.

---

## Backend layout

`handler/` → `service/` → `db/` (wired in `internal/server/router.go`, boot in `cmd/server/main.go`).

Also: `stripeclient/` (SDK), `middleware/` (rate limit, JWT, logging), `auth/` (order tokens + guest JWT).

---

## Data model

| Table | Purpose |
|-------|---------|
| `products` | Catalog — price source of truth |
| `orders` | Purchase attempt + Stripe session id + status |
| `order_items` | Line items snapshotted at checkout |
| `webhook_events` | Dedup log — each `stripe_event_id` once |

`customers` exists but unused in v1.

**Order status:** `pending` → `processing` → `paid` | `expired` | `failed` | `canceled` | `refunded`. Forward-only; no downgrade from `paid`/`refunded`. Webhooks may arrive out of order.

**Webhook processing:** `received` → `processing` → `processed` | `ignored` | `failed` (stale `processing` reclaimed after 5 min).

**Key rules:**
- Pricing via Stripe `price_data` from DB; one currency per order.
- Order inserted before Stripe call; failures → `canceled` (never deleted).
- Redirect URLs from `APP_FRONTEND_URL` only (no client-supplied URLs).
- `Idempotency-Key`: same key+body replays session; mismatch → `409`; in-flight → `409 CHECKOUT_IN_PROGRESS`.
- `accessToken` → `X-Order-Token` for order reads; guest JWT for `/api/orders/mine` (requires `orderId` + `accessToken` proof at `POST /api/auth/session`).

---

## API

Envelope: `{ "data": …, "error": null }` or `{ "data": null, "error": { "code", "message" } }`.

| Endpoint | Notes |
|----------|-------|
| `GET /health/live`, `/health/ready` | Liveness / readiness |
| `GET /api/products` | Catalog |
| `POST /api/checkout/sessions` | `uiMode`, `items[]`; header `Idempotency-Key` |
| `GET /api/orders/by-session/:id` | Poll after pay; `X-Order-Token` |
| `GET /api/orders/:id` | Same token |
| `POST /api/auth/session` | Guest JWT; requires `email` + `orderId` + `accessToken` proof |
| `GET /api/orders/mine` | Bearer JWT |
| `POST /api/webhooks/stripe` | Stripe events |
| `GET /metrics` | Prometheus; `X-API-Key` |

**Checkout 201:** `orderId`, `orderNumber`, `sessionId`, `accessToken`, `url` or `clientSecret`.

**Webhook → order:**

| Event | Status |
|-------|--------|
| `checkout.session.completed` | `paid` or `processing` |
| `checkout.session.async_payment_succeeded` | `paid` |
| `checkout.session.async_payment_failed` | `failed` |
| `checkout.session.expired` | `expired` |
| `charge.refunded` | `refunded` |

Invalid signature → `400`. Handler failure → `500` (Stripe retries). Concurrent claim → `503`.

---

## Frontend

Vite + React; dev proxy `/api` → `:8080`.

| Route | Action |
|-------|--------|
| `/` | Catalog |
| `/checkout/hosted?productId=` | Redirect to Stripe |
| `/checkout/embedded?productId=` | Embedded widget |
| `/checkout/success?session_id=` | Poll (hosted) |
| `/checkout/complete` | Poll (embedded) |
| `/checkout/cancel` | Cancel page |

`Idempotency-Key` in `sessionStorage` per product + `uiMode`; clear before hosted redirect / after embedded complete.

---

## Security & ops

Webhook signature on raw body · server-side pricing · order tokens · rate limits · CORS to `CORS_ORIGIN` · no `client_secret` on order reads · structured logs (no secrets/card data).

**Prod:** `ENV=production`, live keys, `AUTH_JWT_SECRET`, `METRICS_API_KEY`, TLS on DB, secrets in K8s/vault. Tracing: `OTEL_EXPORTER_OTLP_ENDPOINT` + `OTEL_SERVICE_NAME`. Observability: `deploy/prometheus/*`, `deploy/grafana/dashboard.json`. Deploy: `scripts/k8s-deploy.sh`. Compliance: `docs/COMPLIANCE.md`.

**Stack:** Go + chi + pgx · Postgres 16 · `golang-migrate` · `stripe-go` + `@stripe/react-stripe-js`.

**Tests:** service (pricing, idempotency), webhook (out-of-order, refunds), DB integration, E2E webhook in CI.

---

## Scope

**v1 out:** subscriptions, Connect, Tax, refund UI, full accounts.

**v2:** Stripe Price sync, real auth, disputes/reconciliation, frontend trace propagation, Playwright E2E.

---

## Production readiness

**~84%** payment microservice · **~62%** full platform.

**Design & code quality (snapshot):**

| | |
|--|--|
| **Strong** | Handler→service→db layering with injectable ports (`handler/services.go`, `stripeclient/`). Money path: server-side pricing, order-before-Stripe + compensation, webhook as source of truth, atomic claim/complete, amount verify, Dahlia `ui_mode` mapping, canonical idempotency hash + per-`uiMode` client keys. ~29 Go test files + Vitest components; E2E webhook in CI. |
| **Weak** | Handlers return `db.Order` / `db.Product` (HTTP coupled to schema). In-memory rate limits (`middleware/ratelimit.go`) ×2 K8s replicas = uneven 2× caps. Guest JWT grants full email order history after one access-token proof — no email verify. Stripe idempotency key (`orderID`) ≠ client `Idempotency-Key`. No Playwright E2E, load tests, or frontend traces. `customers` table unused. |

| Area | Grade | Gap |
|------|-------|-----|
| Payments | A | — |
| Security | B+ | localStorage Bearer JWT (XSS); no refresh/revocation; per-process rate limits at 2 replicas; no email verify / password reset |
| Reliability | B+ | Webhook retry/reclaim, probes, PDB, graceful shutdown; no multi-region or distributed limits |
| Observability | A− | OTLP (Jaeger), log `trace_id`, metrics + Prometheus rules; no frontend trace propagation |
| Ops | B | CI + release build, `k8s-deploy.sh`, web + ingress manifests, kubeconform; registry push & alert apply still manual |
| Compliance | C− | `docs/COMPLIANCE.md` (PCI SAQ A, data inventory); no retention/DSR automation or legal sign-off |

| Scenario | OK? |
|----------|-----|
| MVP / internal tool | Yes |
| Behind your app's auth | Yes |
| Public standalone checkout | Harden guest auth + distributed rate limits first |
| Public at scale | Add load tests, registry CD, frontend traces |
| Enterprise | No |

**Before go-live:** migrations through `000003_production_hardening` · prod env vars · [Stripe go-live checklist](https://docs.stripe.com/get-started/checklist/go-live) · `scripts/k8s-deploy.sh` · review `docs/COMPLIANCE.md` · import Grafana dashboard + Prometheus rules.
