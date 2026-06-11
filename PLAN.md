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

Also: `stripeclient/` (SDK), `middleware/` (Postgres rate limit, session auth, logging), `auth/` (order tokens + password hashing + session cookies).

---

## Data model

| Table | Purpose |
|-------|---------|
| `products` | Catalog — price source of truth |
| `customers` | User accounts (`email`, `password_hash`, optional `stripe_customer_id`) |
| `orders` | Purchase attempt + Stripe session id + status; optional `customer_id` FK |
| `order_items` | Line items snapshotted at checkout |
| `webhook_events` | Dedup log — each `stripe_event_id` once |
| `user_sessions` | Server-side auth sessions (revocable httpOnly cookie) |
| `rate_limit_buckets` | Distributed per-IP rate-limit counters |
| `password_reset_tokens` | One-time password reset tokens |
| `email_verification_tokens` | One-time email verification tokens |

**Order status:** `pending` → `processing` → `paid` | `expired` | `failed` | `canceled` | `refunded`. Forward-only; no downgrade from `paid`/`refunded`. Webhooks may arrive out of order.

**Webhook processing:** `received` → `processing` → `processed` | `ignored` | `failed` (stale `processing` reclaimed after 5 min).

**Key rules:**
- Pricing via Stripe `price_data` from DB; one currency per order.
- Order inserted before Stripe call; failures → `canceled` (never deleted).
- Redirect URLs from `APP_FRONTEND_URL` only (no client-supplied URLs).
- `Idempotency-Key`: same key+body replays session; mismatch → `409`; in-flight → `409 CHECKOUT_IN_PROGRESS`.
- `accessToken` → `X-Order-Token` for per-order reads (poll after checkout).
- `/api/orders/mine` requires httpOnly `session` cookie (email/password account); lists orders by `customer_id` only.

---

## API

Envelope: `{ "data": …, "error": null }` or `{ "data": null, "error": { "code", "message" } }`.

| Endpoint | Notes |
|----------|-------|
| `GET /health/live`, `/health/ready` | Liveness / readiness |
| `GET /api/products` | Catalog |
| `POST /api/auth/register` | `{ email, password }` → sets `session` cookie |
| `POST /api/auth/login` | `{ email, password }` → sets `session` cookie |
| `POST /api/auth/logout` | Clears `session` cookie; revokes server session |
| `GET /api/auth/me` | Session cookie |
| `POST /api/auth/forgot-password` | `{ email }` — reset link dev-logged (no SMTP yet) |
| `POST /api/auth/reset-password` | `{ token, password }` |
| `POST /api/auth/verify-email` | `{ token }` |
| `POST /api/checkout/sessions` | `uiMode`, `items[]`; header `Idempotency-Key` |
| `GET /api/orders/by-session/:id` | Poll after pay; `X-Order-Token` |
| `GET /api/orders/:id` | Same token |
| `GET /api/orders/mine` | Session cookie |
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
| `/login`, `/signup` | Account sign-in / sign-up |
| `/forgot-password`, `/reset-password` | Password recovery |
| `/verify-email` | Email verification (from sign-up link) |
| `/orders` | Order history (protected — session cookie) |
| `/checkout/hosted?productId=` | Redirect to Stripe |
| `/checkout/embedded?productId=` | Embedded widget |
| `/checkout/success?session_id=` | Poll (hosted) |
| `/checkout/complete` | Poll (embedded) |
| `/checkout/cancel` | Cancel page |

`Idempotency-Key` in `sessionStorage` per product + `uiMode`; clear before hosted redirect / after embedded complete. Auth uses httpOnly `session` cookie (`credentials: 'include'`); no client-side token storage.

---

## Security & ops

Webhook signature on raw body · server-side pricing · order tokens · Postgres rate limits · httpOnly session cookies (revocable) · bcrypt passwords · CORS `AllowCredentials` to `CORS_ORIGIN` · no `client_secret` on order reads · structured logs (no secrets/card data).

**Prod:** `ENV=production`, live keys, `METRICS_API_KEY`, TLS on DB, `Secure` session cookies, secrets in K8s/vault. Tracing: `OTEL_EXPORTER_OTLP_ENDPOINT` + `OTEL_SERVICE_NAME`. Observability: `deploy/prometheus/*`, `deploy/grafana/dashboard.json`. Deploy: `scripts/k8s-deploy.sh`. Compliance: `docs/COMPLIANCE.md`.

**Stack:** Go + chi + pgx · Postgres 16 · `golang-migrate` · `stripe-go` + `@stripe/react-stripe-js`.

**Tests:** service (pricing, idempotency, auth), webhook (out-of-order, refunds), DB integration, E2E webhook in CI, Vitest (checkout + auth pages).

---

## Scope

**v1 out:** subscriptions, Connect, Tax, refund UI, SMTP delivery, OAuth/SSO, CSRF tokens.

**v2:** Stripe Price sync, disputes/reconciliation, frontend trace propagation, Playwright E2E, load tests.

---

## Production readiness

**~84%** payment microservice · **~70%** full platform.

**Design & code quality (snapshot):**

| | |
|--|--|
| **Strong** | Handler→service→db layering with injectable ports (`handler/services.go`, `stripeclient/`). Money path: server-side pricing, order-before-Stripe + compensation, webhook as source of truth, atomic claim/complete, amount verify, Dahlia `ui_mode` mapping, canonical idempotency hash + per-`uiMode` client keys. ~29 Go test files + Vitest components; E2E webhook in CI. |
| **Weak** | Handlers return `db.Order` / `db.Product` (HTTP coupled to schema). Stripe idempotency key (`orderID`) ≠ client `Idempotency-Key`. Checkout does not attach `customer_id` when user is signed in. Verify/reset emails dev-log only. No Playwright E2E, load tests, or frontend traces. |

| Area | Grade | Gap |
|------|-------|-----|
| Payments | A | — |
| Security | A− | verify/reset emails dev-log only (no SMTP); no CSRF tokens beyond SameSite=Lax |
| Reliability | B+ | no multi-region |
| Observability | A− | no frontend trace propagation |
| Ops | B | registry push & alert apply still manual |
| Compliance | C− | no retention/DSR automation or legal sign-off |

| Scenario | OK? |
|----------|-----|
| MVP / internal tool | Yes |
| Behind your app's auth | Yes |
| Public standalone checkout | Mostly — add SMTP + CSRF before broad public launch |
| Public at scale | No — add load tests, registry CD, frontend traces |
| Enterprise | No |

**Before go-live:** migrations through `000005_auth_hardening` · prod env vars · [Stripe go-live checklist](https://docs.stripe.com/get-started/checklist/go-live) · `scripts/k8s-deploy.sh` · review `docs/COMPLIANCE.md` · import Grafana dashboard + Prometheus rules · configure SMTP for account emails.
