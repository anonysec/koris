-- Per-user Telegram MTProto proxy credentials, limits and usage.
-- Each customer gets a single dedicated secret/token for the MTProto proxy
-- running on a node, with per-user connection and bandwidth limits.
CREATE TABLE IF NOT EXISTS user_telegram_proxies (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  node_id BIGINT NOT NULL,
  port INTEGER NOT NULL DEFAULT 443,
  secret TEXT NOT NULL,
  token TEXT NOT NULL,
  max_connections INTEGER NOT NULL DEFAULT 0,
  bandwidth_limit_bytes BIGINT NOT NULL DEFAULT 0,
  used_connections INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (user_id)
);
CREATE INDEX IF NOT EXISTS idx_user_telegram_proxies_user ON user_telegram_proxies (user_id);
