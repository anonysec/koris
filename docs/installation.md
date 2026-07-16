# 📘 Installation & Operations

This guide covers every supported way to install **Koris** and how to operate the resulting stack.

> **TL;DR — Docker one-liner:**
> ```bash
> bash <(curl -Ls https://raw.githubusercontent.com/anonysec/koris/main/install.sh)
> ```

---

## 1. 🐳 Docker (recommended)

The installer provisions the full stack via `docker compose`: the Koris panel, TimescaleDB, and pgAdmin.

```bash
bash <(curl -Ls https://raw.githubusercontent.com/anonysec/koris/main/install.sh)
```

Running with **no flags** installs with sensible defaults: edition `full`, port `2096`, `KORIS_HOME=/opt/koris`, and **HTTP on `127.0.0.1` only** (loopback, not public). Tune via env vars before running: `KORIS_SRC`, `KORIS_HOME`, `KORIS_REPO`, `KNODE_REPO`.

If an existing install is detected, the script preserves your `.env` files (secrets/DB password kept) and re-deploys.

### Install & manage

```bash
# Install (curl-pipeable):
bash <(curl -Ls https://raw.githubusercontent.com/anonysec/koris/main/install.sh)

# All management is done via the `koris` binary (host wrapper installed above):
koris status                                       # panel status
koris cert selfsign                                # self-signed cert → HTTPS
koris cert letsencrypt --domain=panel.example.com  # Let's Encrypt (ACME)
koris cert path --cert=/p/cert.pem --key=/p/key.pem # your own cert/key
koris start | stop | restart                       # stack lifecycle (docker compose)
koris logs                                         # follow container logs
```

### Exposing HTTPS publicly

1. Install a cert: `koris cert selfsign|letsencrypt|path`
2. Publish the port on the host interface: in `docker-compose.yml` change
   `127.0.0.1:2096:2096` → `2096:2096`, then `koris restart`.
3. Point DNS at the host and open `https://<host>:2096/`.

### Manual Docker deployment (no installer)

If you prefer to build and run the compose stack yourself, the Docker agent maintains the final `docker-compose.yml`. The canonical flow:

```bash
# 1. Configure environment (copy the example and edit secrets)
cp .env.example .env
nano .env          # set PANEL_SESSION_SECRET, POSTGRES_PASSWORD, PANEL_DOMAIN, ...

# 2. Build and start (panel + db always; add optional profiles)
docker compose up -d --build
docker compose --profile redis   up -d     # optional: shared Redis cache/queue
docker compose --profile pgadmin  up -d    # optional: pgAdmin UI on localhost
```

- **`.env`** holds every variable the compose file references (`PANEL_PG_DSN`, `PANEL_SESSION_SECRET`, `PANEL_SETUP_KEY`, `PANEL_RADIUS_SECRET`, `PANEL_DOMAIN`, `PANEL_*_PATH`, `POSTGRES_*`, `PGADMIN_*`, `REDIS_ADDR`).
- **pgAdmin** is gated behind `--profile pgadmin` and should never be exposed publicly — it binds to `127.0.0.1` by default.
- **Redis** (`--profile redis`) is optional; when `REDIS_ADDR` is unset, the panel falls back to in-memory implementations (see [Configuration](configuration.md)).
- Migrations run automatically on first startup.

No Nginx / reverse proxy — the panel terminates TLS directly (ACME, manual certs, or self-signed).

---

## 2. 📥 Pre-built binary

Every release ships **self-contained binaries** (frontends + migrations embedded):

```bash
# Pick your edition/arch: koris-{full,lite}-linux-{amd64,arm64}
curl -LO https://github.com/anonysec/koris/releases/latest/download/koris-full-linux-amd64
curl -LO https://github.com/anonysec/koris/releases/latest/download/SHA256SUMS

# Verify integrity
sha256sum -c SHA256SUMS --ignore-missing

chmod +x koris-full-linux-amd64
sudo mv koris-full-linux-amd64 /usr/local/bin/koris
koris
```

You must provide a Postgres/TimescaleDB connection and required env vars (see [Configuration](configuration.md)).

---

## 3. 🐋 GHCR image

```bash
docker run -d --name koris \
  -p 2026:2026 -p 80:80 \
  -e PANEL_SESSION_SECRET="$(openssl rand -hex 32)" \
  -e PANEL_PG_DSN="postgres://user:pass@db:5432/koris?sslmode=disable" \
  ghcr.io/anonysec/koris:latest
```

Multi-arch (amd64 + arm64) tags: `latest`, `<major>`, `<major>.<minor>`, `<version>`.

---

## 4. 🧱 From source

```bash
git clone https://github.com/anonysec/koris.git && cd koris
make build          # frontends + backend → ./koris
```

**Requirements:** Go 1.25+, Node 20+, pnpm 9+. See the [root README](../README.md) for all `make` targets.

---

## ✅ Post-install checklist

1. Browse to `https://<host>:2026/admin/` and complete first-run setup (create the owner account).
2. Add a node — install [knode](https://github.com/anonysec/knode) on each VPN server.
3. Configure TLS (ACME recommended for public domains).
4. Set a strong `PANEL_SESSION_SECRET` — the panel refuses to start in production without it.

---

## Operating the stack

### CLI management

After installation, use the `koris` CLI:

```bash
koris                # Launch interactive menu (numbered options + submenus)
koris start          # Start all services
koris stop           # Stop all services
koris restart        # Restart all services
koris status         # Show service status
koris logs           # View panel logs
koris update         # Update to latest version
koris downgrade v1.x # Downgrade to a specific version
koris reinstall      # Rebuild from source (preserves DB)
koris db backup      # Backup database
koris db restore     # Restore database
koris pgadmin status # Manage pgAdmin service
koris clean          # Remove unused images and build cache
koris uninstall      # Full uninstall
```

### Database management

```bash
koris db backup                 # Backup to /var/backups/koris/ (gzipped pg_dump)
koris db backup --path=/mnt/backups
koris db restore <file>        # Drop+recreate DB, restore dump (confirms first)
koris db migrate               # Run pending migrations in the panel container
koris db reset                 # Drop+recreate DB, run all migrations (prompts)
koris db shell                 # Interactive psql in the DB container
koris db status                # DB size, connections, TimescaleDB version
```

Manual equivalent (`koris-db` container):

```bash
docker exec koris-db pg_dump -U koris -d koris | gzip > backup_$(date +%Y%m%d).sql.gz
gunzip -c backup_20240101.sql.gz | docker exec -i koris-db psql -U koris -d koris
```

### Scaling & low-memory profiles

Scale worker processes within a single container via `PANEL_WORKERS` (each worker shares the port via `SO_REUSEPORT`; only one holds the background-task leader lock). See [Configuration → Tuning](configuration.md#tuning--small-vps-profiles) for DB-pool and memory guidance.

| RAM | `PANEL_WORKERS` | `PANEL_DB_MAX_OPEN` |
|-----|-----------------|---------------------|
| 1 GB | 1 | 10 |
| 2 GB | 2 | 25 |
| 4 GB | 4 | 50 |
| 8 GB+ | 4 | 50 |

### Updating & version pinning

```bash
koris update                    # Pull latest and rebuild (health-checks after)
koris update --version=v1.2.3   # Update to a specific version
koris downgrade v1.1.0          # Roll back
```

The installed version is recorded in `/etc/koris/version`.

### Cleanup

```bash
koris clean                   # Remove project images + prune build cache
koris clean --volumes         # Also remove panel-data and pgadmin-data (preserves DB)
koris clean --volumes --include-db  # Also remove the database volume
koris clean --all --force     # Remove everything (no confirmation)
```

### Uninstall

```bash
# Remove the stack and data:
docker compose -f /opt/koris-src/docker-compose.yml down -v --remove-orphans
rm -rf /opt/koris /opt/koris-src /usr/local/bin/koris
```

### Troubleshooting

```bash
# Health & logs
docker inspect --format='{{.State.Health.Status}}' koris
docker compose logs --tail=50 koris
docker compose logs -f koris-db

# Port conflict
sudo ss -tlnp | grep :2026     # then set PANEL_PORT / PANEL_ADDR in .env and `docker compose up -d`

# DB not ready
docker compose ps koris-db
docker compose restart koris

# Migrations failing
docker compose logs koris | grep -i migrat
```

---

## Admin quick reference

Day-to-day operations from the admin UI (`/admin/`).

### Customer statuses

| Status | Description |
|--------|-------------|
| `active` | Working, can connect |
| `disabled` | Manually suspended by admin |
| `expired` | Subscription end date passed |
| `limited` | Data limit reached |
| `deleted` | Archived (soft-deleted) |

### Node statuses

| Status | Description |
|--------|-------------|
| `online` | Agent is reporting, node is healthy |
| `offline` | Agent has not reported recently |
| `stale` | Agent missed multiple heartbeats |
| `disabled` | Manually disabled by admin |

### Common tasks

- **Add a node** — *Nodes → Add Node*; the panel generates an auth token and auto-syncs the master OpenVPN CA to the new node.
- **Install knode** on the VPN server: `bash <(curl -Ls https://raw.githubusercontent.com/anonysec/knode/master/install.sh)`.
- **VPN cores** — configure OpenVPN / L2TP / IKEv2 / WireGuard per node under *Nodes → node → Cores*; restart nodes after changes.
- **Domains** — manage hostnames (Cloudflare-managed DNS) used in client configs; IKEv2 domains auto-issue/renew Let's Encrypt certs.
- **Settings** — Telegram bot, certificates, promo codes, branding, session timeout, rate limiting, notification templates, scheduled backups, DNS failover.
- **Reports & audits** — revenue/users/bandwidth charts (CSV-exportable) and per-action audit logs (*Settings → Audit Logs*).
- **Diagnostics** — CPU/memory/disk status, recent logs, and AI-assisted troubleshooting under *Diagnostics*.

Next: [Configuration →](configuration.md) · [Architecture →](architecture.md)
