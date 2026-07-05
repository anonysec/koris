-- Add username column to vpn_certificates for per-user client certs
ALTER TABLE vpn_certificates ADD COLUMN IF NOT EXISTS username VARCHAR(255) DEFAULT NULL;
CREATE INDEX IF NOT EXISTS idx_vpn_certificates_username ON vpn_certificates(username);
