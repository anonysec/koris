# 📘 Installation

This guide covers every supported way to install **Koris**.

> **TL;DR** — Docker one-liner:
> ```bash
> bash <(curl -Ls https://raw.githubusercontent.com/anonysec/koris/main/install.sh)
> ```

---

## 1. 🐳 Docker (recommended)

The installer provisions the full stack via `docker compose`: the Koris panel, TimescaleDB, and pgAdmin.

```bash
bash <(curl -Ls https://raw.githubusercontent.com/anonysec/koris/main/install.sh)
```

Running with **no flags** starts an interactive prompt for:

| Prompt | Default | Notes |
|--------|---------|-------|
| Edition | `full` | `full` or `lite` |
| Domain | *(none)* | Enables Let's Encrypt ACME when set |
| Port | `8080`/`443` | HTTP + HTTPS |
| Database | bundled TimescaleDB | or point at an external Postgres |
| SSL mode | self-signed | `acme` / `manual` / `selfsigned` |
| URL routing | path | `/admin/` + `/account/`, or subdomains |

If an existing install is detected you'll be offered **reinstall / wipe / update / cancel**.

### Non-interactive flags

```bash
install.sh --full                       # Full edition (default)
install.sh --lite                       # Lite edition (smaller, fewer deps)
install.sh --port=8080                  # Custom HTTP port
install.sh --domain=panel.example.com   # Public domain → ACME TLS
install.sh --admin-path=/manage/        # Custom admin path
install.sh --portal-path=/app/          # Custom portal path
install.sh --admin-host=admin.example.com   # Subdomain routing
install.sh --no-knode                   # Skip bundling the knode agent
```

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
  -p 443:443 -p 8080:8080 \
  -e PANEL_SESSION_SECRET="$(openssl rand -hex 32)" \
  -e DATABASE_URL="postgres://user:pass@db:5432/koris" \
  ghcr.io/anonysec/koris:latest
```

Multi-arch (amd64 + arm64) tags: `latest`, `<major>`, `<major>.<minor>`, `<version>`.

---

## 4. 🧱 From source

```bash
git clone https://github.com/anonysec/koris.git && cd koris
make build          # frontends + backend → ./koris
```

**Requirements:** Go 1.25+, Node 20+, pnpm 9+.

---

## ✅ Post-install checklist

1. Browse to `https://<host>/admin/` and complete first-run setup.
2. Add a node — install [knode](https://github.com/anonysec/knode) on each VPN server.
3. Configure TLS (ACME recommended for public domains).
4. Set a strong `PANEL_SESSION_SECRET` — the panel refuses to start in production without it.

Next: [Configuration →](configuration.md)
