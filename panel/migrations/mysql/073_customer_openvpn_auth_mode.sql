-- Add per-user OpenVPN auth mode: 'userpass' (default) or 'certificate' (passwordless).
-- The server supports both simultaneously; this column determines which .ovpn profile to generate.
ALTER TABLE customers ADD COLUMN openvpn_auth_mode VARCHAR(16) NOT NULL DEFAULT 'userpass';
