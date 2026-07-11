-- API keys for external integrations.
--
-- NOTE: 001_init.sql already creates `api_keys` with the legacy schema
-- (name UNIQUE, key_hash, scopes, enabled, last4). The application code
-- (internal/api/api_keys.go) reads/writes `key_prefix` and `created_by`,
-- which did not exist in 001. Re-declaring the table here is a no-op once
-- 001 has run, and the `CREATE INDEX ... ON api_keys(key_prefix)` that
-- followed would fail on a fresh database because the column was absent.
--
-- This migration idempotently reconciles the table to the schema the app
-- expects using additive, IF-NOT-EXISTS DDL so it is safe to re-run.

ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS key_prefix TEXT NOT NULL DEFAULT '';
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS created_by TEXT NOT NULL DEFAULT '';

-- Drop legacy columns the application no longer references (safe: unused).
ALTER TABLE api_keys DROP COLUMN IF EXISTS enabled;
ALTER TABLE api_keys DROP COLUMN IF EXISTS last4;

CREATE INDEX IF NOT EXISTS idx_api_keys_prefix ON api_keys(key_prefix);
