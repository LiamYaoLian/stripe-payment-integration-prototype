DROP INDEX IF EXISTS idx_webhook_events_stale_processing;
ALTER TABLE orders DROP COLUMN IF EXISTS access_token_hash;
ALTER TABLE webhook_events DROP COLUMN IF EXISTS processing_started_at;
