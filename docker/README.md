# Koris — Docker deployment

Single-host stack: **panel** (Go API + 3 embedded Vue SPAs), **db**
(TimescaleDB/Postgres 16), **knode** (VPN agent, built from `../knode`), plus
optional **redis** and **pgadmin** behind profiles.

## Quick start

```bash
cd /home/dev/koris

# 1. Create your env (never commit .env)
cp .env.example .env && chmod 600 .env
#    Edit .env and set at least: POSTGRES_PASSWORD, PANEL_SESSION_SECRET,
#    PANEL_SETUP_KEY, PGADMIN_PASSWORD.

# 2. Build and start the core stack (db + panel + knode)
docker compose up -d --build

# 3. (optional) add Redis and/or pgAdmin
docker compose --profile redis --profile pgadmin up -d

# 4. Watch it come up
docker compose ps
docker compose logs -f panel
```

Panel is served on `http://localhost:2026` (or `:80`/`:443` if TLS is
configured). First visit shows the setup wizard; use `PANEL_SETUP_KEY`.

## knode and the panel

- knode is built from the sibling repo at `../knode` using its own Dockerfile.
- On first boot the knode entrypoint writes `/etc/knode/config.toml` from env
  vars. If `KNODE_API_KEYS` is empty, a secure random key is generated and
  printed to the logs.
- Read the key: `docker compose exec knode cat /etc/knode/config.toml`
- In the panel, add the node as **`knode:2083`** with that API key. knode and
  panel share the `koris` network, so the hostname `knode` resolves.
- knode needs `NET_ADMIN`/`NET_RAW` and `/dev/net/tun` (granted in compose).
  Publish VPN data-plane ports as needed, e.g. `"51820:51820/udp"` for
  WireGuard.

## Profiles

| Profile  | Service | Notes                              |
|----------|---------|------------------------------------|
| `redis`  | redis   | Set `REDIS_ADDR=redis:6379` in `.env` for panel/worker to use it. |
| `pgadmin`| pgadmin | Bound to `127.0.0.1` only — use an SSH/VPN tunnel. |

## Notes

- `.env` is auto-loaded by Compose and must stay out of version control.
- The panel image embeds the SPA assets at build time and also ships
  `migrations/` + the `www/` dirs at its working dir for runtime overrides.
- No `docker` CLI is available in the authoring environment, so images were
  not built here — validate with `docker compose config` on your machine.
