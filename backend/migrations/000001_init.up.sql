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
    customer_id                TEXT REFERENCES customers(id),
    customer_email             TEXT,
    stripe_checkout_session_id TEXT UNIQUE,
    stripe_payment_intent_id   TEXT,
    stripe_checkout_url        TEXT,
    stripe_client_secret       TEXT,
    ui_mode                    ui_mode NOT NULL,
    success_url                TEXT,
    cancel_url                 TEXT,
    return_url                 TEXT,
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
