# ⚙️ Configuration

Koris is configured primarily through **environment variables** (read at boot by `internal/config`). When installed via Docker, the installer writes these into `panel.env` and wires them into `docker-compose.yml`.

---

## 🔑 Required

| Variable | Description |
|----------|-------------|
| `PANEL_SESSION_SECRET` | Secret used to sign sessions. **The panel refuses to start in production without it.** Generate with `openssl rand -hex 32`. |
| `DATABASE_URL` | Postgres/TimescaleDB DSN, e.g. `postgres://user:pass@host:5432/koris`. |

> 🧪 For local development only, set `PANEL_DEV_MODE=true` to relax the session-secret requirement.

---

## 🌐 Networking & routing

| Variable | Default | Description |
|----------|---------|-------------|
| `PANEL_ADMIN_PATH` | `/admin/` | Path prefix for the admin SPA |
| `PANEL_PORTAL_PATH` | `/account/` | Path prefix for the customer portal |
| `PANEL_ADMIN_HOST` | *(unset)* | Serve admin on a subdomain instead of a path |
| `PANEL_PORTAL_HOST` | *(unset)* | Serve portal on a subdomain |
| `KORIS_ADMIN_BASE` | `/admin/` | **Build-time** Vite base — must match `PANEL_ADMIN_PATH` |
| `KORIS_PORTAL_BASE` | `/account/` | **Build-time** Vite base — must match `PANEL_PORTAL_PATH` |

> ⚠️ `PANEL_*_PATH` is read at **runtime** (router mounts adjust live), but `KORIS_*_BASE` is baked into the frontend bundle at **build time**. Change both together and rebuild the image if you move the SPA base.

---

## 🎨 Web asset dirs (advanced)

Frontends are embedded by default. These override with on-disk assets (fallback order: disk → embed):

| Variable | Default |
|----------|---------|
| `PANEL_ADMIN_WEB_DIR` | `/opt/koris/web/admin/www` |
| `PANEL_PORTAL_WEB_DIR` | `/opt/koris/web/portal/www` |
| `PANEL_LANDING_WEB_DIR` | `/opt/koris/web/landing/www` |

You normally never set these — the embedded assets are used automatically.

---

## 🔒 TLS

| Mode | How |
|------|-----|
| **ACME** (Let's Encrypt) | Set a public `--domain` at install; certs auto-issue & renew |
| **Manual** | Provide cert/key paths |
| **Self-signed** | Default for IP-only installs |

External traffic is **HTTPS-enforced**; plain HTTP is restricted to loopback.

---

## 🧵 Workers, cache & DB tuning

See [`docs/low-memory-tuning.md`](low-memory-tuning.md) for small-VPS profiles (worker counts, cache sizes, connection pools).

---

## 🖼️ UI themes

Themes are selected in the admin UI and stored per-user. Available: **Default, Kiro, GitHub, Soft-Dark, Corporate, Midnight**, each with dark/light/system modes. Tokens live in `web/core/styles/tokens.css`; the cross-cutting polish layer is `web/theme/styles/overhaul.css`. See [UI/UX →](ui-ux.md).

---

## 📄 Full reference

Every knob is defined in [`internal/config/config.go`](../internal/config/config.go). When in doubt, that file is the source of truth.
