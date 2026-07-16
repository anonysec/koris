<div align="center">

# 🛡️ Koris

### Multi-protocol VPN management platform — nodes, billing, resellers & a modern web UI, all from one dashboard.

[![Release](https://img.shields.io/github/v/tag/anonysec/koris?label=release&sort=semver)](https://github.com/anonysec/koris/releases)
[![CI](https://github.com/anonysec/koris/actions/workflows/ci.yml/badge.svg)](https://github.com/anonysec/koris/actions/workflows/ci.yml)
[![Docker](https://img.shields.io/badge/ghcr.io-anonysec%2Fkoris-2496ED?logo=docker&logoColor=white)](https://github.com/anonysec/koris/pkgs/container/koris)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Vue](https://img.shields.io/badge/Vue-3-42B883?logo=vuedotjs&logoColor=white)](https://vuejs.org)
[![Docs](https://img.shields.io/badge/docs-github.io-22d3ee?logo=readthedocs&logoColor=white)](https://anonysec.github.io/koris/)

Manage your **entire VPN infrastructure** — nodes, customers, subscriptions, payments, support tickets and more — from a single, self-contained binary. 🚀

</div>

---

## ✨ Highlights

- 🔌 **6 VPN protocols** — OpenVPN, WireGuard, L2TP/IPsec, IKEv2, SSH Tunnel, MTProto
- 🌐 **Multi-node fleet** — unlimited [knode](https://github.com/anonysec/knode) agents over gRPC + mTLS
- 💳 **Billing built-in** — plans, wallets, invoices, crypto & gateway payments
- 🧑‍💼 **Reseller system** — sub-accounts, credit allocation, customer provisioning
- 📊 **Real-time metrics** — live streaming, health checks, bandwidth accounting
- 🎨 **Modern UI** — Vue 3 SPA, 6 themes, dark/light/system, RTL, drag-and-drop nav
- 📦 **One binary** — frontends + migrations embedded; no external assets to ship
- 🐳 **Docker-native** — TimescaleDB + pgAdmin + Panel in one `compose` file

---

## 🚀 Quick start with Docker

Docker is the **primary, recommended path**. The installer provisions the whole stack (panel + TimescaleDB + pgAdmin) for you:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/anonysec/koris/main/install.sh)
```

No flags → interactive setup (edition, domain, port, DB, SSL). If an install is detected you'll be offered reinstall / wipe / update / cancel.

Prefer to build and run the compose stack yourself? The Docker agent maintains the final `docker-compose.yml`:

```bash
cp .env.example .env && nano .env     # set PANEL_SESSION_SECRET, POSTGRES_PASSWORD, PANEL_DOMAIN, ...
docker compose up -d --build                 # panel + db
docker compose --profile redis   up -d       # optional: shared Redis cache/queue
docker compose --profile pgadmin  up -d       # optional: pgAdmin UI (binds localhost)
```

Then open the panel. The **default install serves HTTP on `127.0.0.1:2096`** (loopback only, not public). To expose HTTPS run `koris cert selfsign|letsencrypt|path`, then republish the port on `0.0.0.0`. → full details in [docs/installation.md](docs/installation.md).

<details>
<summary>⚙️ Install & manage</summary>

```bash
# Install (curl-pipeable, no flags → uses defaults):
bash <(curl -Ls https://raw.githubusercontent.com/anonysec/koris/main/install.sh)

# All management is done through the `koris` binary (host wrapper installed by install.sh):
koris status                                  # panel status
koris cert selfsign                           # install a self-signed cert (HTTPS)
koris cert letsencrypt --domain=panel.example.com   # Let's Encrypt
koris cert path --cert=/p/cert.pem --key=/p/key.pem # your own cert
koris start | stop | restart                  # stack lifecycle (docker compose)
koris logs                                    # follow container logs
```

</details>
### 📥 Pre-built binary (no Docker)

```bash
curl -LO https://github.com/anonysec/koris/releases/latest/download/koris-full-linux-amd64
chmod +x koris-full-linux-amd64
./koris-full-linux-amd64
```

> Binaries are named `koris-<edition>-linux-<arch>` (`full`/`lite` × `amd64`/`arm64`). Verify with the `SHA256SUMS` attached to each release.

### 🧱 From source

```bash
git clone https://github.com/anonysec/koris.git && cd koris
make build          # frontends + backend  →  ./koris
make help           # list all targets
```

---

## 🏗️ Architecture

```
┌──────────────┐        gRPC + mTLS        ┌──────────────┐
│              │  ───────────────────────▶ │   knode 🛰️   │  VPN node agent
│   Koris 🛡️   │  ◀─────────────────────── │  (OpenVPN,   │
│    Panel     │     metrics / health      │  WireGuard…) │
│              │                           └──────────────┘
│  Vue 3 SPA   │        ┌───────────────┐
│  + Go API    │ ─────▶ │ TimescaleDB 🐘 │  metrics + relational data
└──────────────┘        └───────────────┘
```

- **Backend:** Go 1.25, `./cmd/panel`, ~40 internal packages
- **Frontend:** pnpm workspace — `admin`, `portal`, `landing` (+ shared `core`/`theme`), embedded via `go:embed`
- **Nodes:** [`anonysec/knode`](https://github.com/anonysec/knode) — see its README

See [docs/architecture.md](docs/architecture.md) for components, editions, and node management.

---

## 🛠️ Development

```bash
make help            # 📋 list every target
make build           # 🏗️  frontends + backend (full)
make build-lite      # 🪶 lite edition
make backend         # ⚙️  Go binary only
make frontend        # 🎨 all three SPAs
make vet             # 🔍 go vet ./...
make test            # 🧪 go test ./...
make check           # ✅ vet + test (CI gate)
make clean           # 🧹 remove artifacts
```

**Requirements:** Go 1.25+, Node 20+, pnpm 9+.

---

## 🔐 Security

- 🔑 mTLS between panel and every node
- 🍪 CSRF protection, session management, rate limiting
- 🔒 HTTPS enforced for external traffic
- 🛡️ See [SECURITY.md](SECURITY.md) to report vulnerabilities

---

## 📖 Documentation

Full guides live in [`docs/`](docs/):

- 📘 [Installation & Operations](docs/installation.md)
- 🏛️ [Architecture & Nodes](docs/architecture.md)
- ⚙️ [Configuration](docs/configuration.md)
- 📡 [API Reference](docs/API.md)
- 🎨 [UI / UX](docs/ui-ux.md)
- 🚀 [Release Process](RELEASING.md)

---

## 🤝 Contributing

PRs welcome! Read [CONTRIBUTING.md](CONTRIBUTING.md), run `make check` before pushing, and keep commits conventional (`feat:`, `fix:`, `docs:` …).

<div align="center">

Made with 🛡️ by the Koris team · [Report a bug](https://github.com/anonysec/koris/issues) · [Nodes → knode](https://github.com/anonysec/knode)

</div>
