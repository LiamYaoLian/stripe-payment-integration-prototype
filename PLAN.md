# Stripe Payment Integration — PLAN.md

## Context

The repo [`stripe-payment-integration-prototype`](.) is an empty greenfield scaffold (no app code yet). Goal: **receive one-time payments** via Stripe with a **Go backend** and **React frontend**.

**Delivery scope (single phase)**
- Hosted Checkout (redirect) **and** Embedded Checkout (`ui_mode: embedded` via `EmbeddedCheckoutProvider`)
- Async payment webhooks + full order state machine + Go API + React frontend

Per Stripe best practices (API version **2026-05-27.dahlia**):
- Use **Checkout Sessions API** as the single payment backend for both UI modes
- Omit `payment_method_types` (enable dynamic payment methods from Dashboard)
- Verify webhook signatures; use restricted API keys (`rk_`) in production
- Never use Charges API or legacy Card Element

---

## Architecture

```mermaid
sequenceDiagram
    participant Browser
    participant FE as React_Vite
    participant API as Go_API
    participant DB as Postgres_Docker
    participant Stripe

    Browser->>FE: Click Pay on product
    FE->>API: POST /api/checkout/sessions (+ Idempotency-Key)
    API->>DB: BEGIN; insert Order + items; COMMIT
    API->>Stripe: checkout.sessions.create (price_data line items)
    alt Stripe success
        API->>DB: UPDATE order with session id + checkout_url
        API-->>FE: session payload
    else Stripe failure
        API->>DB: UPDATE order status=canceled (soft fail, audit retained)
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

### Tech stack (recommended defaults)

| Layer | Choice | Rationale |
|-------|--------|-----------|
| Backend | **Go 1.22+** (`chi` router) | Fast, typed HTTP API; idiomatic Stripe webhook handling |
| Frontend | **Vite + React + TypeScript** | Lightweight SPA; required for Stripe Embedded Checkout |
| DB access | **sqlc + pgx/v5** | Type-safe SQL queries generated from schema |
| Migrations | **golang-migrate** | Versioned SQL migrations, standard Go tooling |
| DB | **PostgreSQL 16 (Docker)** | Production-like local dev; same engine in all environments |
| Container | **docker-compose** | One-command Postgres startup with persisted volume |
| Stripe SDK | `stripe-go` (server), `@stripe/stripe-js` + `@stripe/react-stripe-js` (client) | Official SDKs; **Go pins API version to SDK release** — use latest `stripe-go` for `2026-05-27.dahlia` |
| Validation | `go-playground/validator` (API) | Request validation on API; frontend has minimal input in v1 |
| Logging | `log/slog` (structured JSON) | Request ID middleware; never log secrets or full card data |

**Ports (dev)**
- Go API: `http://localhost:8080`
- React (Vite): `http://localhost:5173`
- Postgres: `localhost:5432`

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

### Enums

**`OrderStatus`**
- `pending` — order row exists; awaiting Stripe session attachment and/or customer payment. If `stripe_checkout_session_id` is null, checkout creation is still in-flight
- `processing` — `checkout.session.completed` received but payment is async (e.g. bank debit); awaiting `async_payment_succeeded`
- `paid` — payment confirmed (`checkout.session.completed` with `payment_status=paid`, or `async_payment_succeeded`)
- `expired` — `checkout.session.expired`
- `failed` — `async_payment_failed` or unrecoverable payment failure
- `canceled` — checkout session creation failed (Stripe API error, UPDATE failure, or `session.Expire` cleanup); order retained for audit with `metadata.cancel_reason`
- `refunded` — future: `charge.refunded` (out of v1 scope, reserved in schema)

**State machine rules** (supports out-of-order webhooks):
- Transitions are **monotonic forward** — never downgrade `paid` or `refunded`
- `pending` may jump directly to `paid` if `async_payment_succeeded` arrives before `checkout.session.completed`
- `pending` → `processing` → `paid` is the typical async path, but `pending` → `paid` is valid
- `expired` / `canceled` only apply if not already `paid` or `processing`

**`UiMode`**: `hosted` | `embedded`

**`WebhookProcessingStatus`**: `received` | `processed` | `ignored` | `failed` (Postgres enum, not free text)

### SQL schema (target file: [`backend/migrations/000001_init.up.sql`](backend/migrations/000001_init.up.sql))

```sql
CREATE TYPE order_status AS ENUM ('pending', 'processing', 'paid', 'expired', 'failed', 'canceled', 'refunded');
CREATE TYPE ui_mode AS ENUM ('hosted', 'embedded');
CREATE TYPE webhook_processing_status AS ENUM ('received', 'processed', 'ignored', 'failed');

CREATE TABLE products (
    id                TEXT PRIMARY KEY,
    name              TEXT NOT NULL,
    description       TEXT,
    unit_amount_cents INT NOT NULL CHECK (unit_amount_cents > 0),
    currency          TEXT NOT NULL DEFAULT 'usd',
    active            BOOLEAN NOT NULL DEFAULT true,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE customers (
    id                 TEXT PRIMARY KEY,
    email              TEXT NOT NULL UNIQUE,
    stripe_customer_id TEXT UNIQUE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE orders (
    id                         TEXT PRIMARY KEY,
    order_number               TEXT NOT NULL UNIQUE,
    idempotency_key            TEXT UNIQUE,
    status                     order_status NOT NULL DEFAULT 'pending',
    total_amount_cents         INT NOT NULL CHECK (total_amount_cents > 0),
    currency                   TEXT NOT NULL DEFAULT 'usd',
    customer_id                TEXT REFERENCES customers(id),  -- v2: set only with authenticated user
    customer_email             TEXT,   -- guest checkout: denormalized string, no Customer upsert
    stripe_checkout_session_id TEXT UNIQUE,
    stripe_payment_intent_id   TEXT,
    stripe_checkout_url        TEXT,   -- hosted: session.url; stored for idempotency replay
    stripe_client_secret       TEXT,   -- embedded: session.client_secret; stored for idempotency replay
    ui_mode                    ui_mode NOT NULL,
    success_url                TEXT,   -- hosted only; server-built
    cancel_url                 TEXT,   -- hosted only; server-built
    return_url                 TEXT,   -- embedded only; server-built
    metadata                   JSONB,
    paid_at                    TIMESTAMPTZ,
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE order_items (
    id                TEXT PRIMARY KEY,
    order_id          TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id        TEXT REFERENCES products(id),
    product_name      TEXT NOT NULL,
    quantity          INT NOT NULL DEFAULT 1 CHECK (quantity > 0),
    unit_amount_cents INT NOT NULL CHECK (unit_amount_cents > 0),
    line_total_cents  INT NOT NULL CHECK (line_total_cents > 0)
);

CREATE TABLE webhook_events (
    id                TEXT PRIMARY KEY,
    stripe_event_id   TEXT NOT NULL UNIQUE,
    event_type        TEXT NOT NULL,
    order_id          TEXT REFERENCES orders(id),
    processing_status webhook_processing_status NOT NULL,
    payload           JSONB NOT NULL,
    processed_at      TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_orders_stripe_checkout_session_id ON orders(stripe_checkout_session_id);
CREATE INDEX idx_orders_stripe_payment_intent_id ON orders(stripe_payment_intent_id);
```

**Line items → Stripe**: use dynamic `price_data` (not Stripe Price IDs) since the catalog lives in Postgres:

```go
&stripe.CheckoutSessionLineItemParams{
    PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
        Currency:   stripe.String(product.Currency),
        UnitAmount: stripe.Int64(int64(product.UnitAmountCents)),
        ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
            Name: stripe.String(product.Name),
        },
    },
    Quantity: stripe.Int64(int64(item.Quantity)),
}
```

**sqlc** generates typed Go structs and query methods from [`backend/sql/queries/`](backend/sql/queries/). IDs use `github.com/rs/xid` (sortable, URL-safe; e.g. `d4j8k2m9q1p7n3s6`).

**Schema source of truth**: `backend/migrations/` is authoritative. Keep [`backend/sql/schema.sql`](backend/sql/schema.sql) in sync manually (or generate from migrations) so sqlc matches the live DB.

### Design decisions

- **Snapshot line items** (`productName`, `unitAmountCents` on `OrderItem`) so historical orders stay accurate if catalog prices change
- **`orderNumber`** human-readable (e.g. `ORD-20250610-XXXX`) for support/receipts; internal `id` uses `xid` (e.g. `d4j8k2m9q1p7n3s6`)
- **`stripeCheckoutSessionId`** is the primary Stripe correlation key; all webhook handlers resolve orders via session ID in event payload
- **`WebhookEvent.stripeEventId`** deduplicates deliveries — `processed` events are no-ops; `failed` events are retried on Stripe redelivery
- **Guest email, no Customer upsert** — v1 has no auth. If `customerEmail` is provided, store it on `orders.customer_email` and pass to Stripe `customer_email` for checkout pre-fill. **Do not** upsert `customers` by email — prevents account pollution (guest entering another user's email). Link `customer_id` only when JWT auth exists (v2)
- **Single currency per order** — service layer rejects mixed-currency line items before creating a session
- **Soft-fail, never hard-delete orders** — DB insert commits **before** the Stripe HTTP call (short transaction). On Stripe API error or UPDATE failure: set order `status = canceled`, write `metadata.cancel_reason` (e.g. `stripe_api_error`, `persist_failed`, `session_expire_failed`). Retain row for audit. Attempt `session.Expire` on best-effort basis; log if expire also fails
- **Stale in-flight orders** — background job or checkout retry may mark `pending` orders with null `stripe_checkout_session_id` older than 15 minutes as `canceled` with `cancel_reason = stale_checkout`
- **Redirect URLs are server-built** from `APP_FRONTEND_URL` — never accepted from the client (prevents open redirects). In dev, `APP_FRONTEND_URL` and `CORS_ORIGIN` are the same value; only the backend needs `APP_FRONTEND_URL`
- **Checkout idempotency** — optional `Idempotency-Key` header on `POST /api/checkout/sessions`. Duplicate key with identical body → return cached `201` from `stripe_checkout_url` / `stripe_client_secret` on the existing order. Duplicate key with different body → `409 IDEMPOTENCY_CONFLICT`. **In-flight**: if a row exists for the key but `stripe_checkout_session_id` is still null, return `409 CHECKOUT_IN_PROGRESS` (do not create a second Stripe session); client should retry after a short backoff
- **Stripe UPDATE failure** — if Stripe succeeds but persisting `session_id` / `url` fails, retry the UPDATE up to 3 times; on final failure, best-effort `session.Expire`, then set order `canceled` with `cancel_reason = persist_failed`
- **`stripe_client_secret` never exposed** — stored in DB for idempotency replay only; never returned from `GET /api/orders/*` (only from `POST /api/checkout/sessions` response)
- **Stripe metadata limits** — user-supplied `metadata` validated: max 50 keys, string values only, max 500 chars each (Stripe constraint)
- **`price_data` tradeoff** — creates ephemeral Stripe Products per checkout; Stripe Dashboard product insights will be fragmented. Fine for prototype; **v2**: sync catalog to Stripe Products/Prices on create/update (see implementation order)
- **Idempotency on `canceled` orders** — same `Idempotency-Key` pointing to a `canceled` order → create a **new** order (old row kept for audit); do not replay canceled sessions

---

## Database (Docker)

Postgres runs locally via Docker Compose. File: [`docker-compose.yml`](docker-compose.yml)

```yaml
services:
  postgres:
    image: postgres:16-alpine
    container_name: stripe-payment-db
    restart: unless-stopped
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: stripe
      POSTGRES_PASSWORD: stripe
      POSTGRES_DB: stripe_payment
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U stripe -d stripe_payment"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
```

**Notes**
- Port `5432` exposed to host so the Go API and migration CLI connect via `localhost`
- Named volume `postgres_data` persists data across container restarts
- Healthcheck ensures migrations run only after Postgres is ready

---

## API contract

Base URL: `http://localhost:8080` (Go API, dev)

Frontend calls the API via `VITE_API_URL`. Vite dev server proxies `/api` → `http://localhost:8080` for same-origin requests.

All JSON responses use envelope:

```json
{ "data": { ... }, "error": null }
```

Errors:

```json
{ "data": null, "error": { "code": "VALIDATION_ERROR", "message": "...", "details": [] } }
```

### `GET /health`

Liveness/readiness for Docker and local dev.

**Response 200**
```json
{ "status": "ok", "db": "connected" }
```

---

### `GET /api/products`

List active catalog products.

**Response 200**
```json
{
  "data": {
    "products": [
      {
        "id": "d4j8k2m9q1p7n3s6",
        "name": "Pro License",
        "description": "One-time purchase",
        "unitAmountCents": 4900,
        "currency": "usd"
      }
    ]
  }
}
```

---

### `POST /api/checkout/sessions`

Create an order and Stripe Checkout Session.

**Headers**

| Header | Required | Notes |
|--------|----------|-------|
| `Idempotency-Key` | no | UUID; see idempotency rules below |

**Request body (shared fields)**
```json
{
  "uiMode": "hosted",
  "items": [
    { "productId": "d4j8k2m9q1p7n3s6", "quantity": 1 }
  ],
  "customerEmail": "buyer@example.com",
  "metadata": { "source": "web" }
}
```

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `uiMode` | `"hosted"` \| `"embedded"` | yes | Determines redirect vs embedded UI response |
| `items` | array | yes | Min 1 item; `productId` must exist and be active; amounts computed server-side |
| `customerEmail` | string | no | Pre-fills Stripe Checkout; stored on `orders.customer_email` only (no `Customer` upsert in v1) |
| `metadata` | object | no | Stored on Order + passed to Stripe `metadata` (string values only; max 50 keys, 500 chars/value) |

**Idempotency rules**
- Same `Idempotency-Key` + identical body + session already created + order not `canceled` → return cached `201` from `stripe_checkout_url` / `stripe_client_secret`
- Same key + existing order `canceled` → create a **new** order and proceed with checkout
- Same key + identical body + session not yet created (`stripe_checkout_session_id` is null) + order `pending` → `409 CHECKOUT_IN_PROGRESS`; client retries after ~500ms
- Same `Idempotency-Key` + different request body → `409 IDEMPOTENCY_CONFLICT`
- Request body comparison uses canonical JSON hash (sorted keys) stored on order or computed at runtime
- No key → always create a new order (double-click risk; frontend should always send a key)

**Frontend idempotency key**: generate once per checkout attempt and persist in `sessionStorage` keyed by `productId` so page refresh reuses the same key instead of creating duplicate orders

**Redirect URLs — server-built only** (not in request body):

| `uiMode` | URLs constructed by backend from `APP_FRONTEND_URL` |
|----------|------------------------------------------------------|
| `hosted` | `success_url`: `{APP_FRONTEND_URL}/checkout/success?session_id={CHECKOUT_SESSION_ID}` |
| `hosted` | `cancel_url`: `{APP_FRONTEND_URL}/checkout/cancel` |
| `embedded` | `return_url`: `{APP_FRONTEND_URL}/checkout/complete` |

**Server-side Stripe call** (Go, no `payment_method_types`, `price_data` line items):
```go
params := &stripe.CheckoutSessionParams{
    Mode:   stripe.String(string(stripe.CheckoutSessionModePayment)),
    UIMode: stripe.String(uiMode),
    LineItems: lineItems, // price_data from Product table
    ClientReferenceID: stripe.String(orderNumber),
    Metadata: map[string]string{
        "order_id":     orderID,
        "order_number": orderNumber,
    },
}
if customerEmail != "" {
    params.CustomerEmail = stripe.String(customerEmail)
}
switch uiMode {
case "hosted":
    params.SuccessURL = stripe.String(successURL)
    params.CancelURL = stripe.String(cancelURL)
case "embedded":
    params.ReturnURL = stripe.String(returnURL)
}
session, err := session.New(params)
// On success: UPDATE order SET stripe_checkout_session_id, stripe_checkout_url (retry up to 3x)
// On UPDATE failure after Stripe success: best-effort session.Expire + order status=canceled
// On Stripe API error: order status=canceled, metadata.cancel_reason set
```

**Response 201 — hosted**
```json
{
  "data": {
    "orderId": "d4j8k2m9q1p7n3s6",
    "orderNumber": "ORD-20250610-A1B2",
    "sessionId": "cs_test_...",
    "url": "https://checkout.stripe.com/c/pay/cs_test_..."
  }
}
```

**Response 201 — embedded**
```json
{
  "data": {
    "orderId": "d4j8k2m9q1p7n3s6",
    "orderNumber": "ORD-20250610-A1B2",
    "sessionId": "cs_test_...",
    "clientSecret": "cs_test_..._secret_..."
  }
}
```

**Error codes**: `VALIDATION_ERROR` (400), `PRODUCT_NOT_FOUND` (404), `IDEMPOTENCY_CONFLICT` (409), `CHECKOUT_IN_PROGRESS` (409), `STRIPE_ERROR` (502)

---

### `GET /api/orders/:id`

Poll order status after redirect (success page).

**Response 200**
```json
{
  "data": {
    "id": "d4j8k2m9q1p7n3s6",
    "orderNumber": "ORD-20250610-A1B2",
    "status": "paid",
    "totalAmountCents": 4900,
    "currency": "usd",
    "paidAt": "2025-06-10T12:00:00.000Z",
    "items": [
      { "productName": "Pro License", "quantity": 1, "lineTotalCents": 4900 }
    ]
  }
}
```

**Polling behavior**: Treat `pending` and `processing` as in-flight; poll until `paid`, `failed`, `expired`, `canceled`, or timeout (~30s). Final payment truth comes from webhooks, not the redirect alone.

**Access control (prototype)**: Endpoints are unauthenticated. Responses expose `orderNumber`, status, and line items only — no PII. Acceptable for a local prototype; add session-scoped tokens before production.

---

### `GET /api/orders/by-session/:sessionId`

Lookup order by Stripe Checkout Session ID (used on success page with `?session_id=`).

Same response shape as `GET /api/orders/:id`. The `sessionId` must match `^cs_` (reject malformed IDs with 400).

---

### `POST /api/webhooks/stripe`

Stripe webhook endpoint. **Raw body required** for signature verification.

**Handled events**

| Event | Action |
|-------|--------|
| `checkout.session.completed` | Verify `session.status == "complete"`. If `payment_status=paid` → `paid` (skip `processing`). If `payment_status=unpaid` → `processing`. Store `payment_intent`, set `paidAt` when `paid` |
| `checkout.session.async_payment_succeeded` | If order is `pending` or `processing` → `paid` (out-of-order safe). Store `payment_intent`, set `paidAt` |
| `checkout.session.async_payment_failed` | Set order `failed` (only if not `paid`) |
| `checkout.session.expired` | Set order `expired` (only if not `paid` or `processing`) |

All handlers resolve the order via `session.id` → `orders.stripe_checkout_session_id`. Future `charge.refunded` may resolve via `orders.stripe_payment_intent_id` (indexed). Do **not** use standalone `payment_intent.payment_failed`. Store `stripe_payment_intent_id` on **every** path that reaches `paid`.

**Behavior**
1. Read raw body + `Stripe-Signature` header
2. `webhook.ConstructEvent(body, sig, STRIPE_WEBHOOK_SECRET)` (stripe-go)
3. Upsert `WebhookEvent` by `stripeEventId`:
   - **New event** → insert with `processing_status = received`, proceed to step 4
   - **Existing + `processed`** → return `200` immediately (already handled)
   - **Existing + `failed`** → retry processing (Stripe is retrying after our failure)
   - **Existing + `ignored`** → return `200` (unhandled event type, already recorded)
4. Process event in a DB transaction with optimistic lock: `UPDATE webhook_events SET processing_status = 'processed' WHERE stripe_event_id = $1 AND processing_status IN ('received', 'failed')`
   - **1 row affected** → update `Order`, return `200`
   - **0 rows affected** → query current `processing_status`:
     - `processed` → return `200` (another worker finished)
     - `failed` → return `500` (trigger Stripe retry)
     - `received` → return `503` (another worker in progress; Stripe will retry later — do not swallow)
5. On handler error: set `processing_status = failed`, log error, return `500` so Stripe retries (on retry, step 3 catches `failed` and re-attempts)
6. On success: return `200 { "received": true }`
7. Return `400` only for invalid signatures (not a Stripe retry scenario)
8. **Unhandled event types** (not in table above) → insert with `processing_status = ignored`, return `200`

---

## Frontend pages

React SPA (Vite + React Router). All API calls go to the Go backend. Checkout requests send an `Idempotency-Key` (UUID) stored in `sessionStorage` per `productId` — survives page refresh, prevents duplicate orders on double-click. Clear the key from `sessionStorage` immediately before redirecting to `session.url` so a repeat purchase of the same product starts fresh. On `409 CHECKOUT_IN_PROGRESS`, retry after 500ms.

**Vite proxy** ([`frontend/vite.config.ts`](frontend/vite.config.ts)): proxy `/api` → `http://localhost:8080` with `changeOrigin: true` (required for Embedded Checkout iframe / third-party cookie compatibility).

**Checkout navigation**
1. Catalog fetches `GET /api/products`
2. Each product shows two actions:
   - **Pay (Hosted)** → `/checkout/hosted?productId={id}`
   - **Pay (Embedded)** → `/checkout/embedded?productId={id}`
3. `HostedCheckout` reads `productId`, calls `POST /api/checkout/sessions` with `uiMode: "hosted"`, then `window.location.href = data.url`
4. `EmbeddedCheckout` reads `productId`, calls `POST /api/checkout/sessions` with `uiMode: "embedded"`, then renders `EmbeddedCheckoutProvider` with `data.clientSecret`

| Route | File | Purpose |
|-------|------|---------|
| `/` | [`frontend/src/pages/Catalog.tsx`](frontend/src/pages/Catalog.tsx) | Product catalog + hosted/embedded Pay links |
| `/checkout/hosted` | [`frontend/src/pages/HostedCheckout.tsx`](frontend/src/pages/HostedCheckout.tsx) | Reads `productId` → creates session → redirects to `session.url` |
| `/checkout/embedded` | [`frontend/src/pages/EmbeddedCheckout.tsx`](frontend/src/pages/EmbeddedCheckout.tsx) | Reads `productId` → creates session → renders `EmbeddedCheckoutProvider` |
| `/checkout/success` | [`frontend/src/pages/CheckoutSuccess.tsx`](frontend/src/pages/CheckoutSuccess.tsx) | Hosted: reads `session_id`, polls order status |
| `/checkout/cancel` | [`frontend/src/pages/CheckoutCancel.tsx`](frontend/src/pages/CheckoutCancel.tsx) | Hosted cancel confirmation |
| `/checkout/complete` | [`frontend/src/pages/CheckoutComplete.tsx`](frontend/src/pages/CheckoutComplete.tsx) | Embedded `return_url` landing page; polls order status |

## Backend packages (Go)

| Package | Responsibility |
|---------|----------------|
| [`cmd/server`](backend/cmd/server/main.go) | Entry point, wiring, graceful shutdown |
| [`internal/config`](backend/internal/config) | Env parsing (`DATABASE_URL`, Stripe keys, `CORS_ORIGIN`, `APP_FRONTEND_URL`) |
| [`internal/middleware`](backend/internal/middleware) | Request ID, CORS |
| [`internal/handler`](backend/internal/handler) | HTTP handlers: products, checkout, orders, webhooks |
| [`internal/service`](backend/internal/service) | Business logic: order creation, price validation, webhook processing |
| [`internal/db`](backend/internal/db) | sqlc-generated queries + transaction helpers |
| [`internal/stripe`](backend/internal/stripe) | Stripe client wrapper |

---

## Environment variables

### Backend — [`backend/.env.example`](backend/.env.example)

```bash
# Server
PORT=8080
CORS_ORIGIN=http://localhost:5173
APP_FRONTEND_URL=http://localhost:5173   # server-built redirect URLs; never from client

# Stripe (test mode for dev)
STRIPE_SECRET_KEY=rk_test_...          # prefer restricted key
STRIPE_WEBHOOK_SECRET=whsec_...

# Postgres (matches docker-compose.yml defaults)
DATABASE_URL=postgresql://stripe:stripe@localhost:5432/stripe_payment?sslmode=disable
```

### Frontend — [`frontend/.env.example`](frontend/.env.example)

```bash
VITE_API_URL=http://localhost:8080
VITE_STRIPE_PUBLISHABLE_KEY=pk_test_...   # required for Embedded Checkout
VITE_APP_URL=http://localhost:5173
```

**Key handling**
- `STRIPE_SECRET_KEY` and `STRIPE_WEBHOOK_SECRET` live only in the Go backend — never in frontend env
- `VITE_STRIPE_PUBLISHABLE_KEY` is safe to expose (publishable key only)
- Use `stripe sandbox create` or Dashboard for test keys
- Add pre-commit hook to block `sk_`/`rk_` in committed files

---

## Project structure (files to create)

```
stripe-payment-integration-prototype/
├── PLAN.md
├── README.md
├── docker-compose.yml               # Postgres 16 for local dev
├── Makefile                         # migrate, seed, run, dev, test
├── backend/
│   ├── go.mod
│   ├── .env.example
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── config/config.go
│   │   ├── handler/
│   │   │   ├── health.go
│   │   │   ├── products.go
│   │   │   ├── checkout.go
│   │   │   ├── orders.go
│   │   │   ├── webhooks.go
│   │   │   └── testdata/            # webhook fixture JSON
│   │   ├── middleware/
│   │   │   ├── requestid.go
│   │   │   └── cors.go
│   │   ├── service/
│   │   │   ├── order.go
│   │   │   └── webhook.go
│   │   ├── db/                      # sqlc-generated
│   │   └── stripe/client.go
│   ├── migrations/
│   │   ├── 000001_init.up.sql
│   │   └── 000001_init.down.sql
│   ├── sql/
│   │   ├── schema.sql
│   │   └── queries/
│   │       ├── products.sql
│   │       ├── orders.sql
│   │       └── webhooks.sql
│   ├── sqlc.yaml
│   └── seed/seed.go
└── frontend/
    ├── package.json
    ├── .env.example
    ├── vite.config.ts               # proxy /api → localhost:8080, changeOrigin: true
    └── src/
        ├── api/client.ts            # typed fetch wrapper
        ├── pages/
        │   ├── Catalog.tsx
        │   ├── HostedCheckout.tsx
        │   ├── EmbeddedCheckout.tsx
        │   ├── CheckoutSuccess.tsx
        │   ├── CheckoutCancel.tsx
        │   └── CheckoutComplete.tsx
        ├── App.tsx
        └── main.tsx
```

---

## Local dev workflow

1. `docker compose up -d` — start Postgres (wait for healthy status)
2. `cp backend/.env.example backend/.env` — fill Stripe keys
3. `make migrate && make seed` — schema + sample products
4. `make dev` — starts Go API (`:8080`) and Vite (`:5173`) concurrently
5. Forward webhooks: `stripe listen --forward-to localhost:8080/api/webhooks/stripe`
   - Copy the `whsec_...` secret printed by the CLI into `backend/.env` as `STRIPE_WEBHOOK_SECRET` (changes each `stripe listen` run)
6. Open `http://localhost:5173` and test with card `4242 4242 4242 4242`

**Makefile targets**

| Target | Action |
|--------|--------|
| `make migrate` | Run golang-migrate against Postgres |
| `make seed` | Insert sample products |
| `make run` | Start Go API only |
| `make dev` | Start Go API + Vite frontend |
| `make test` | Run `go test ./...` |

**Useful Docker commands**
- `docker compose ps` — check container health
- `docker compose logs -f postgres` — tail DB logs
- `docker compose down` — stop Postgres (data kept in volume)
- `docker compose down -v` — stop and wipe DB volume (fresh start)

---

## Security checklist

- Webhook signature verification on every event
- Idempotent webhook processing via `WebhookEvent.stripeEventId`
- Idempotent checkout via `Idempotency-Key` header + `orders.idempotency_key`; in-flight guard returns `409 CHECKOUT_IN_PROGRESS`
- `stripe_client_secret` never returned from order lookup endpoints
- Server-side price validation (never trust client-submitted amounts; compute from `Product` table)
- Redirect URLs server-built from `APP_FRONTEND_URL` — never client-supplied (no open redirects)
- Restricted API key with permissions: `Checkout Sessions (Write)`, `Customers (Write)`, `Webhook Endpoints (Read)` only
- No secret keys in client bundle or logs
- CORS restricted to `CORS_ORIGIN` in dev; tighten for production
- Webhook handler reads raw request body (do not JSON-decode before signature verification)
- Webhook processing failures return `500` to trigger Stripe retry; `failed` events are reprocessed on redelivery; concurrent `received` returns `503` (not `200`)
- Orders are never hard-deleted; checkout failures set `canceled` with `metadata.cancel_reason`
- No `Customer` upsert by email without authentication; guest email stored on order only
- Structured logging via `slog`; include request ID, never log Stripe secrets or full webhook payloads in production

---

## Testing strategy

| Layer | What to test |
|-------|--------------|
| **Service** | Price validation, guest email (no Customer upsert), idempotency replay, in-flight guard, soft-cancel on failure, stale checkout cleanup, out-of-order `pending` → `paid` |
| **Webhook handler** | Optimistic lock 0-row branch (`processed`→200, `failed`→500, `received`→503), out-of-order async events, `payment_intent` on all paid paths |
| **Checkout handler** | Stripe error → `canceled` (not DELETE), idempotency on canceled orders creates new row, canonical JSON body hash |
| **Integration** | `stripe trigger checkout.session.completed` via CLI against local webhook endpoint |

Use Stripe [test fixtures](https://docs.stripe.com/testing) and webhook sample payloads in `backend/internal/handler/testdata/`.

---

## Out of scope

- Subscriptions / recurring billing
- Refunds UI (schema reserves `refunded` status)
- Stripe Connect / marketplace splits
- Tax (Stripe Tax)
- Multi-currency adaptive pricing
- Authenticated order lookup
- Customer table linkage (requires JWT auth)

---

## Implementation order

1. Add `docker-compose.yml` + `Makefile`; scaffold Go module (`chi`, `pgx`, `stripe-go`, `sqlc`, `golang-migrate`)
2. Write SQL migrations + sqlc queries; implement seed script
3. Build `GET /health`, `GET /api/products`, `POST /api/checkout/sessions` (hosted + embedded), order lookups
4. Build webhook handler (all session events above) with signature verification + idempotent processing
5. Scaffold Vite + React frontend with API client, CORS/proxy (`changeOrigin: true`), `Idempotency-Key` generation
6. Build hosted checkout flow end-to-end + webhook tests
7. Build embedded checkout flow (`EmbeddedCheckoutProvider`, `/checkout/complete`)
8. Add README with setup instructions (include `stripe listen` webhook secret step)

**v2 (production hardening)**
9. Stripe catalog sync — on product create/update, write Stripe Product + Price IDs to `products.stripe_price_id`; switch line items from `price_data` to `price` ID
10. JWT auth + `Customer` linkage via authenticated user ID (replace guest-only `customer_email`)
11. `charge.refunded` webhook handler resolving via `stripe_payment_intent_id` index
