# Stripe Payment Integration — Design

## Context

Greenfield prototype: **receive one-time payments** via Stripe with a **Go backend** and **React frontend**.

**Scope (v1)**
- Hosted Checkout (redirect) **and** Embedded Checkout (`ui_mode: embedded`)
- Async payment webhooks, full order state machine, REST API, React SPA

**Stripe integration principles** (API version **2026-05-27.dahlia**):
- **Checkout Sessions API** as the single payment backend for both UI modes
- Omit `payment_method_types` — enable dynamic payment methods from Dashboard
- Verify webhook signatures; use restricted API keys (`rk_`) in production
- Never use Charges API or legacy Card Element

---

## Architecture

```mermaid
sequenceDiagram
    participant Browser
    participant FE as React_Vite
    participant API as Go_API
    participant DB as Postgres
    participant Stripe

    Browser->>FE: Click Pay on product
    FE->>API: POST /api/checkout/sessions (+ Idempotency-Key)
    API->>DB: Create Order + items
    API->>Stripe: checkout.sessions.create (price_data line items)
    alt Stripe success
        API->>DB: Attach session id + checkout URL / client secret
        API-->>FE: session payload
    else Stripe failure
        API->>DB: Mark order canceled (soft fail, audit retained)
        API-->>FE: error
    end
    FE-->>Browser: redirect or render checkout

    alt Hosted Checkout
        Browser->>Stripe: Redirect to session.url
    else Embedded Checkout
        Browser->>Stripe: initEmbeddedCheckout(client_secret)
    end

    Stripe->>API: POST /api/webhooks/stripe
    API->>API: Verify signature
    API->>DB: Update Order (paid)
    API-->>Stripe: 200 OK

    Browser->>FE: Success page polls order
    FE->>API: GET /api/orders/by-session/:sessionId
    API-->>FE: Order status
```

### Tech stack

| Layer | Choice | Rationale |
|-------|--------|-----------|
| Backend | **Go** (`chi` router) | Typed HTTP API; idiomatic Stripe webhook handling |
| Frontend | **Vite + React + TypeScript** | Lightweight SPA; required for Stripe Embedded Checkout |
| DB access | **sqlc + pgx** | Type-safe SQL from schema |
| Migrations | **golang-migrate** | Versioned SQL migrations |
| DB | **PostgreSQL 16 (Docker)** | Production-like local dev |
| Stripe SDK | `stripe-go` (server), `@stripe/stripe-js` + `@stripe/react-stripe-js` (client) | Official SDKs; Go pins API version to SDK release |

---

## Data model

### Entity relationship

```mermaid
erDiagram
    Product ||--o{ OrderItem : "referenced_by"
    Order ||--|{ OrderItem : contains
    Order ||--o| Customer : "optional"
    WebhookEvent }o--|| Order : "may_update"

    Product {
        string id PK
        string name
        string description
        int unit_amount_cents
        string currency
        bool active
        datetime created_at
        datetime updated_at
    }

    Customer {
        string id PK
        string email
        string stripe_customer_id UK
        datetime created_at
    }

    Order {
        string id PK
        string order_number UK
        string idempotency_key UK
        string status
        int total_amount_cents
        string currency
        string customer_id FK
        string customer_email
        string stripe_checkout_session_id UK
        string stripe_payment_intent_id
        string stripe_checkout_url
        string stripe_client_secret
        string ui_mode
        string success_url
        string cancel_url
        string return_url
        json metadata
        datetime paid_at
        datetime created_at
        datetime updated_at
    }

    OrderItem {
        string id PK
        string order_id FK
        string product_id FK
        string product_name
        int quantity
        int unit_amount_cents
        int line_total_cents
    }

    WebhookEvent {
        string id PK
        string stripe_event_id UK
        string event_type
        string order_id FK
        webhook_processing_status processing_status
        json payload
        datetime processed_at
        datetime created_at
    }
```

### Order status

| Status | Meaning |
|--------|---------|
| `pending` | Order exists; awaiting Stripe session attachment and/or customer payment |
| `processing` | `checkout.session.completed` received but payment is async (e.g. bank debit); awaiting `async_payment_succeeded` |
| `paid` | Payment confirmed |
| `expired` | `checkout.session.expired` |
| `failed` | `async_payment_failed` or unrecoverable payment failure |
| `canceled` | Checkout session creation failed; order retained for audit |
| `refunded` | Reserved for v2 (`charge.refunded`) |

**State machine rules** (supports out-of-order webhooks):
- Transitions are **monotonic forward** — never downgrade `paid` or `refunded`
- `pending` may jump directly to `paid` if `async_payment_succeeded` arrives before `checkout.session.completed`
- `pending` → `processing` → `paid` is the typical async path, but `pending` → `paid` is valid
- `expired` / `canceled` only apply if not already `paid` or `processing`

**Other enums**: `UiMode` (`hosted` | `embedded`); `WebhookProcessingStatus` (`received` | `processed` | `ignored` | `failed`)

### Design decisions

**Catalog & line items**
- Catalog lives in Postgres; Stripe line items use dynamic `price_data` (not Stripe Price IDs) in v1
- **Snapshot line items** on `OrderItem` so historical orders stay accurate if catalog prices change
- **Single currency per order** — reject mixed-currency line items before creating a session
- **`price_data` tradeoff** — creates ephemeral Stripe Products per checkout; fine for prototype. **v2**: sync catalog to Stripe Products/Prices

**Order identity & correlation**
- **`orderNumber`** human-readable (e.g. `ORD-20250610-XXXX`) for support/receipts; internal `id` is opaque and sortable
- **`stripeCheckoutSessionId`** is the primary Stripe correlation key; all webhook handlers resolve orders via session ID
- **`WebhookEvent.stripeEventId`** deduplicates deliveries — `processed` events are no-ops; `failed` events are retried on Stripe redelivery

**Customer handling (v1)**
- No auth in v1. Guest email stored on `orders.customer_email` and passed to Stripe for checkout pre-fill
- **Do not** upsert `customers` by email — prevents account pollution. Link `customer_id` only when JWT auth exists (v2)

**Checkout lifecycle**
- **Soft-fail, never hard-delete orders** — DB insert commits **before** the Stripe HTTP call. On Stripe API error or persist failure: set `status = canceled`, write `metadata.cancel_reason`, retain row for audit. Best-effort `session.Expire` on failure paths
- **Stale in-flight orders** — `pending` orders with no `stripe_checkout_session_id` older than 15 minutes may be marked `canceled` with `cancel_reason = stale_checkout`
- **Redirect URLs are server-built** from `APP_FRONTEND_URL` — never accepted from the client (prevents open redirects)

**Idempotency**
- Optional `Idempotency-Key` header on checkout creation. Duplicate key + identical body → return cached session. Duplicate key + different body → `409 IDEMPOTENCY_CONFLICT`
- **In-flight guard**: if a row exists for the key but session ID is still null → `409 CHECKOUT_IN_PROGRESS`; client retries after backoff
- Same key pointing to a `canceled` order → create a **new** order (old row kept for audit)
- **`stripe_client_secret`** stored in DB for idempotency replay only; never returned from order lookup endpoints

**Stripe constraints**
- User-supplied `metadata` validated: max 50 keys, string values only, max 500 chars each

---

## API contract

All JSON responses use an envelope: `{ "data": { ... }, "error": null }`. Errors: `{ "data": null, "error": { "code", "message", "details" } }`.

### `GET /health`

Liveness/readiness. Returns `{ "status": "ok", "db": "connected" }`.

### `GET /api/products`

List active catalog products. Amounts and currency returned for display; prices are never trusted from the client on checkout.

### `POST /api/checkout/sessions`

Create an order and Stripe Checkout Session.

**Request**
| Field | Required | Notes |
|-------|----------|-------|
| `uiMode` | yes | `"hosted"` or `"embedded"` |
| `items` | yes | Min 1 item; `productId` must exist and be active; amounts computed server-side |
| `customerEmail` | no | Pre-fills Stripe Checkout; stored on order only (no Customer upsert) |
| `metadata` | no | Stored on Order + passed to Stripe metadata |

**Headers**: optional `Idempotency-Key` (UUID). Frontend should generate once per checkout attempt and persist across page refresh to prevent duplicate orders on double-click.

**Redirect URLs** (server-built from `APP_FRONTEND_URL`, never from client):

| `uiMode` | URL |
|----------|-----|
| `hosted` | `success_url`: `{APP_FRONTEND_URL}/checkout/success?session_id={CHECKOUT_SESSION_ID}` |
| `hosted` | `cancel_url`: `{APP_FRONTEND_URL}/checkout/cancel` |
| `embedded` | `return_url`: `{APP_FRONTEND_URL}/checkout/complete` |

**Response 201 — hosted**: `orderId`, `orderNumber`, `sessionId`, `url`

**Response 201 — embedded**: `orderId`, `orderNumber`, `sessionId`, `clientSecret`

**Error codes**: `VALIDATION_ERROR` (400), `PRODUCT_NOT_FOUND` (404), `IDEMPOTENCY_CONFLICT` (409), `CHECKOUT_IN_PROGRESS` (409), `STRIPE_ERROR` (502)

### `GET /api/orders/:id` and `GET /api/orders/by-session/:sessionId`

Poll order status after redirect. Returns `orderNumber`, `status`, `totalAmountCents`, `currency`, `paidAt`, line items. No PII beyond what is needed for the receipt view.

**Polling**: treat `pending` and `processing` as in-flight; poll until terminal status or timeout (~30s). Final payment truth comes from webhooks, not the redirect alone.

**Access control (prototype)**: unauthenticated. Acceptable for local dev; add session-scoped tokens before production.

### `POST /api/webhooks/stripe`

Stripe webhook endpoint. **Raw body required** for signature verification.

**Handled events**

| Event | Action |
|-------|--------|
| `checkout.session.completed` | If `payment_status=paid` → `paid` (skip `processing`). If `unpaid` → `processing`. Store `payment_intent`, set `paidAt` when `paid` |
| `checkout.session.async_payment_succeeded` | If `pending` or `processing` → `paid`. Store `payment_intent`, set `paidAt` |
| `checkout.session.async_payment_failed` | Set `failed` (only if not `paid`) |
| `checkout.session.expired` | Set `expired` (only if not `paid` or `processing`) |

All handlers resolve the order via `session.id` → `orders.stripe_checkout_session_id`. Store `stripe_payment_intent_id` on every path that reaches `paid`. Do **not** use standalone `payment_intent.payment_failed`.

**Processing model**
1. Verify `Stripe-Signature` against raw body
2. Upsert `WebhookEvent` by `stripeEventId` — `processed` → no-op `200`; `failed` → retry on redelivery
3. Process in a DB transaction with optimistic lock on `processing_status`
4. Handler error → `500` (Stripe retries); invalid signature → `400`
5. Unhandled event types → record as `ignored`, return `200`

---

## Frontend flows

React SPA. All API calls go to the Go backend. Checkout requests send an `Idempotency-Key` stored in `sessionStorage` per product — survives refresh, prevents duplicate orders. Clear the key before redirecting to hosted checkout so repeat purchases start fresh. On `409 CHECKOUT_IN_PROGRESS`, retry after ~500ms.

**User journey**
1. Catalog lists products from `GET /api/products`
2. Each product offers **Pay (Hosted)** and **Pay (Embedded)**
3. Hosted: create session → redirect to `session.url`
4. Embedded: create session → render `EmbeddedCheckoutProvider` with `clientSecret`
5. Success/complete pages poll order status via session ID

| Route | Purpose |
|-------|---------|
| `/` | Product catalog |
| `/checkout/hosted` | Create hosted session → redirect |
| `/checkout/embedded` | Create embedded session → render checkout |
| `/checkout/success` | Hosted success; poll order |
| `/checkout/cancel` | Hosted cancel confirmation |
| `/checkout/complete` | Embedded return URL; poll order |

---

## Security

- Webhook signature verification on every event; raw body before JSON decode
- Idempotent webhook processing via `WebhookEvent.stripeEventId`
- Idempotent checkout via `Idempotency-Key` + in-flight guard (`409 CHECKOUT_IN_PROGRESS`)
- `stripe_client_secret` never returned from order lookup endpoints
- Server-side price validation — never trust client-submitted amounts
- Redirect URLs server-built from `APP_FRONTEND_URL` — no open redirects
- Restricted API key in production; no secret keys in client bundle or logs
- CORS restricted to known origin
- Webhook failures return `500` to trigger Stripe retry; concurrent in-progress returns `503`
- Orders never hard-deleted; failures set `canceled` with `cancel_reason`
- No `Customer` upsert by email without authentication
- Structured logging with request ID; never log secrets or full card data

---

## Testing focus

| Layer | What to verify |
|-------|----------------|
| **Service** | Price validation, guest email (no Customer upsert), idempotency replay, in-flight guard, soft-cancel on failure, stale checkout cleanup, out-of-order `pending` → `paid` |
| **Webhook handler** | Optimistic lock concurrency, out-of-order async events, `payment_intent` on all paid paths |
| **Checkout handler** | Stripe error → `canceled` (not delete), idempotency on canceled orders creates new row |
| **Integration** | Stripe CLI webhook triggers against local endpoint |

---

## Out of scope (v1)

- Subscriptions / recurring billing
- Refunds UI (schema reserves `refunded` status)
- Stripe Connect / marketplace splits
- Tax (Stripe Tax)
- Multi-currency adaptive pricing
- Authenticated order lookup
- Customer table linkage (requires JWT auth)

## Future (v2)

- Stripe catalog sync — persist Stripe Product/Price IDs; switch line items from `price_data` to Price ID
- JWT auth + `Customer` linkage via authenticated user ID
- `charge.refunded` webhook handler resolving via `stripe_payment_intent_id`
