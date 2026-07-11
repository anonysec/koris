# ⚙️ Configuration

Koris is configured primarily through **environment variables** (read at boot by `internal/config`). When installed via Docker, the installer writes these into `.env` / `panel.env` and wires them into `docker-compose.yml`.

> Every knob is defined in [`internal/config/config.go`](../internal/config/config.go). That file is the source of truth when something here looks off.

---

## 🔑 Required

| Variable | Description |
|----------|-------------|
| `PANEL_SESSION_SECRET` | Secret used to sign sessions. **The panel refuses to start in production without it.** Generate with `openssl rand -hex 32`. |
| `PANEL_DB_DSN` (or `PANEL_PG_DSN`) | Postgres/TimescaleDB DSN, e.g. `postgres://user:pass@host:5432/koris?sslmode=disable`. |

> 🧪 For local development only, set `PANEL_DEV_MODE=true` to relax the session-secret and DB-DSN requirements.

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

Other network knobs: `PANEL_ADDR` (listen addr, default `0.0.0.0:2026`), `PANEL_PORT` (HTTPS port, `2026`), `PANEL_PUBLIC_BASE`, `PANEL_ALLOWED_ORIGINS`, `PANEL_TRUSTED_PROXIES`, `PANEL_SECURE_COOKIES`.

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
| **ACME** (Let's Encrypt) | Set `PANEL_TLS_ENABLED=true` + `PANEL_DOMAIN=example.com`; the panel binds `:80`/`:443` and provisions a trusted cert into `PANEL_TLS_CERT_DIR` (default `/etc/koris/certs`). Skipped automatically if custom cert files exist. |
| **Manual** | Provide `PANEL_TLS_CERT` / `PANEL_TLS_KEY` (or `/etc/koris/cert.pem` + `/etc/koris/key.pem`). |
| **Self-signed** | Default for IP-only installs. |

External traffic is **HTTPS-enforced**; plain HTTP is restricted to loopback. Binding `:80`/`:443` as non-root requires `sudo setcap cap_net_bind_service=ep <binary>` (already set in the Docker image). `PANEL_DEV_MODE=false` in production.

Other TLS knobs: `PANEL_TLS_MODE` (`acme`/`manual`/`selfsigned`), `PANEL_TLS_DOMAIN`, `PANEL_TLS_EMAIL`, `PANEL_TLS_ADDR`.

---

## 🧵 Workers, cache & DB tuning

| Variable | Default | Description |
|----------|---------|-------------|
| `PANEL_WORKERS` | `1` | Worker processes (share port via `SO_REUSEPORT`) |
| `PANEL_GRACEFUL_WAIT` | `30` | Seconds for graceful shutdown |
| `PANEL_DB_MAX_OPEN` | auto-tuned | Max open DB connections |
| `PANEL_DB_MAX_IDLE` | auto-tuned | Max idle DB connections |
| `PANEL_DB_MAX_LIFETIME` | `5m` | Max connection lifetime |
| `REDIS_ADDR` | *(unset)* | Redis host:port — enables shared queue/cache/rate-limiter (else in-memory) |
| `REDIS_PASSWORD` / `REDIS_DB` | *(unset)* | Optional Redis auth/DB |

---

## 🧠 Tuning & small-VPS profiles

### Panel (Go binary)

The panel auto-optimizes for low memory when no explicit `GOMAXPROCS`/`GOGC`/`GOMEMLIMIT` are set:

- `GOMAXPROCS=1` (single thread)
- `GOGC=50` (more frequent GC, lower peak memory)
- `GOMEMLIMIT=100MB` (soft memory cap)

Override with env vars if needed (`GOMAXPROCS=2`, `GOGC=100`, `GOMEMLIMIT=200000000`).

### PostgreSQL / TimescaleDB (recommended for 1 GB RAM)

Add to `postgresql.conf` (or a drop-in in `conf.d/`):

```ini
shared_buffers = 128MB
effective_cache_size = 384MB
work_mem = 4MB
maintenance_work_mem = 32MB
max_connections = 30
wal_buffers = 4MB
# TimescaleDB
timescaledb.max_background_workers = 2
max_worker_processes = 4
```

For the bundled Docker stack, pass these via the TimescaleDB container command or a mounted config file.

### Node Agent (knode)

Already lightweight (~5 MB RSS). No tuning needed.

### Expected memory usage (1 GB server)

| Component | Approx. RSS |
|-----------|-------------|
| PostgreSQL/TimescaleDB | ~250 MB |
| Panel binary | ~30–50 MB |
| FreeRADIUS | ~30 MB |
| OS + buffers | ~200 MB |
| **Headroom** | **~450 MB** |

---

## 🖼️ UI themes

Themes are selected in the admin UI and stored per-user. Available: **Default, Kiro, GitHub, Soft-Dark, Corporate, Midnight**, each with dark/light/system modes. Tokens live in `web/core/styles/tokens.css`; the cross-cutting polish layer is `web/theme/styles/overhaul.css`. See [UI / UX →](ui-ux.md).

---

## 🔐 Secrets & security notes

`PANEL_PG_DSN` / `PANEL_SESSION_SECRET` / `PANEL_SETUP_KEY` are **secrets** — never commit them; provide via environment or Docker secrets. `PANEL_SETUP_KEY` gates first-boot owner creation when set. See [SECURITY.md](../SECURITY.md).
