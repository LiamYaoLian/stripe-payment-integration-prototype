DROP TABLE IF EXISTS email_verification_tokens;
DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS rate_limit_buckets;
DROP TABLE IF EXISTS user_sessions;
ALTER TABLE customers DROP COLUMN IF EXISTS email_verified_at;
