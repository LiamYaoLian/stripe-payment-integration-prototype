ALTER TABLE webhook_events ADD COLUMN IF NOT EXISTS processing_started_at TIMESTAMPTZ;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS access_token_hash TEXT;

CREATE INDEX IF NOT EXISTS idx_webhook_events_stale_processing
    ON webhook_events (processing_started_at)
    WHERE processing_status = 'processing';
