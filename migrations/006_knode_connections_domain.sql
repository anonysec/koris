-- Migration 006: add domain / backup_domains columns to knode_connections.
-- The panel's knode registry SELECTs these columns (registry.go), but the
-- original DDL in 001_init.sql omitted them, causing
-- "ERROR: column "domain" does not exist" on list/get. Additive + idempotent
-- so it is safe to re-run on an already-migrated database.

ALTER TABLE knode_connections
    ADD COLUMN IF NOT EXISTS domain VARCHAR(255) NOT NULL DEFAULT '';

ALTER TABLE knode_connections
    ADD COLUMN IF NOT EXISTS backup_domains TEXT NOT NULL DEFAULT '';
