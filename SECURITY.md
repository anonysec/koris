# Security Policy - KorisPanel

KorisPanel is the control plane for the Koris VPN platform. It stores
**customer credentials, RADIUS/shared secrets, session secrets and TLS
private keys**. A compromise of the panel is a compromise of the whole
platform, so treat it as a high-value target.

## Supported Versions

| Branch | Supported |
|--------|-----------|
| `main` (latest) | ✅ |
| older releases | ❌ |

## Reporting a Vulnerability

Do **not** open public issues for security problems.
Email the maintainers or open a private security advisory on GitHub.
We aim to acknowledge within 72h and ship a fix within 14 days.

## Authentication & Sessions

- Admin/customer passwords are verified against `radcheck` (RADIUS) / the
  user store; never logged.
- Sessions use `SecureCookies` + `SessionSecret` (set via `PANEL_SESSION_SECRET`).
  Rotate the secret to invalidate all sessions.
- Login endpoints are rate-limited (`login_attempts`, 5 attempts / 15 min).
- First-boot owner creation is gated by `PANEL_SETUP_KEY` when set.

## Secrets & Configuration

- `PANEL_PG_DSN`, `PANEL_SESSION_SECRET`, `PANEL_SETUP_KEY` are **secrets**.
  Never commit them; provide via environment / Docker secrets.
- The DB password in `docker-compose.yml` is an example only - override it.
- TLS private keys live under `/etc/koris`; restrict to root `0600`.
- `PANEL_DEV_MODE=false` in production (disables debug surfaces).

## Deployment Hardening

- [ ] Bind the database to `127.0.0.1` (or an internal network) only.
- [ ] Do not expose pgAdmin publicly; bind it to `127.0.0.1` or a VPN.
- [ ] Terminate TLS at the panel (or a trusted proxy) with HSTS.
- [ ] Set resource limits on every container.
- [ ] Rotate `PANEL_SESSION_SECRET` and `PANEL_SETUP_KEY` after provisioning.
- [ ] Keep `PANEL_DEV_MODE=false` in production.
- [ ] Regularly update dependencies (`go mod tidy` + `pnpm audit`).
