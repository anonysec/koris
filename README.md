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

## 📚 Table of Contents

- [🚀 Quick Install](#-quick-install)
- [🧩 Feature Tour](#-feature-tour)
- [🏗️ Architecture](#️-architecture)
- [🛠️ Development](#️-development)
- [📦 Releases](#-releases)
- [🔐 Security](#-security)
- [📖 Documentation](#-documentation)
- [🤝 Contributing](#-contributing)

---

## 🚀 Quick Install

### 🐳 Docker (recommended)

```bash
bash <(curl -Ls https://raw.githubusercontent.com/anonysec/koris/main/install.sh)
```

Run without flags for an **interactive setup** (edition, domain, port, DB, SSL). If an install is detected, you'll be offered reinstall / wipe / update / cancel.

<details>
<summary>⚙️ Non-interactive flags</summary>

```bash
install.sh --full                       # Full edition (default)
install.sh --lite                       # Lite edition
install.sh --port=8080                  # Custom port
install.sh --domain=panel.example.com   # Public domain (enables ACME TLS)
install.sh --no-knode                   # Skip bundling the knode agent
```
</details>

### 📥 Pre-built binary

Grab a self-contained binary from the [**Releases**](https://github.com/anonysec/koris/releases) page:

```bash
curl -LO https://github.com/anonysec/koris/releases/latest/download/koris-full-linux-amd64
chmod +x koris-full-linux-amd64
./koris-full-linux-amd64
```

> 💡 Binaries are named `koris-<edition>-linux-<arch>` (`full`/`lite` × `amd64`/`arm64`). Verify with the `SHA256SUMS` attached to each release.

### 🧱 From source

```bash
git clone https://github.com/anonysec/koris.git && cd koris
make build          # frontends + backend  →  ./koris
make help           # list all targets
```

---

## 🧩 Feature Tour

### 🔒 VPN & Networking
- **Multi-protocol** cores per node — OpenVPN, WireGuard, L2TP/IPsec, IKEv2, SSH, MTProto
- **Outbound tunnels** — VLESS+Reality, WireGuard, SSH, Rathole, GRE/IPIP
- **FreeRADIUS** AAA with session accounting
- **Anti-DPI / teleproxy** helpers for censored networks

### 💼 Business & Billing
- **Subscription plans** — quota-based or pay-as-you-go
- **Wallets & payments** — manual, crypto, and gateway (e.g. Zarinpal)
- **Resellers** — sub-accounts with credit and their own customers
- **Invoices** — auto-generated, PDF export

### 🙋 Customer Experience
- **Self-service portal** — usage, profiles, VPN configs, support
- **Telegram bot** — admin ops + customer self-service via inline buttons
- **Ticketing** — canned responses + knowledge base

### 🖥️ Admin Dashboard
- **Drag-and-drop sidebar**, command palette, onboarding checklist
- **6 UI themes** (Default, Kiro, GitHub, Soft-Dark, Corporate, Midnight) + dark/light/system
- **i18n** — English, Persian (RTL), Chinese, Russian

### 🏭 Infrastructure
- **Two editions** — Full & Lite from one codebase via Go build tags
- **Auto-TLS** — Let's Encrypt (ACME), manual, or self-signed
- **HTTPS enforced** externally; plain HTTP restricted to loopback
- **Decoy landing page** to blend in

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

## 📦 Releases

Each tagged release publishes:

- 🔗 **Raw binaries** — `koris-{full,lite}-linux-{amd64,arm64}` + `SHA256SUMS`
- 🐳 **Multi-arch image** — `ghcr.io/anonysec/koris:<version>` (amd64 + arm64)

> Want a tarball? Clone the repo — GitHub's auto-generated **Source code (zip/tar.gz)** is on every release. We intentionally keep release assets to binaries only. 📉

Cut a release:

```bash
git tag v0.94.0 && git push origin v0.94.0
```

---

## 🔐 Security

- 🔑 mTLS between panel and every node
- 🍪 CSRF protection, session management, rate limiting
- 🔒 HTTPS enforced for external traffic
- 🛡️ See [SECURITY.md](SECURITY.md) to report vulnerabilities

---

## 📖 Documentation

Full guides live in [`docs/`](docs/) and the [project wiki](https://github.com/anonysec/koris/wiki):

- 📘 [Installation](docs/installation.md)
- 🏛️ [Architecture](docs/architecture.md)
- ⚙️ [Configuration](docs/configuration.md)
- 🛰️ [Node Management](docs/nodes.md)
- 🚀 [Release Process](RELEASING.md)

---

## 🤝 Contributing

PRs welcome! Read [CONTRIBUTING.md](CONTRIBUTING.md), run `make check` before pushing, and keep commits conventional (`feat:`, `fix:`, `docs:` …).

<div align="center">

Made with 🛡️ by the Koris team · [Report a bug](https://github.com/anonysec/koris/issues) · [Nodes → knode](https://github.com/anonysec/knode)

</div>
