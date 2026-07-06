# KorisPanel

**Multi-protocol VPN management platform** with real-time node monitoring, customer billing, reseller system, and modern web UI.

Manage your entire VPN infrastructure from a single dashboard: nodes, customers, subscriptions, payments, support tickets, and more.

---

## Key Features

### VPN & Networking
- **Multi-Protocol** — OpenVPN, WireGuard, L2TP/IPsec, IKEv2, SSH Tunnel, MTProto
- **Multi-Node** — Manage unlimited VPN nodes via gRPC with mTLS
- **Outbound Tunnels** — VLESS+Reality, WireGuard, SSH, Rathole, GRE/IPIP
- **FreeRADIUS Integration** — Standards-based AAA with session accounting
- **Real-Time Monitoring** — Live metrics streaming, health checks, bandwidth tracking

### Business & Billing
- **Subscription Plans** — Quota-based or pay-as-you-go pricing
- **Wallet & Payments** — Per-user wallet, manual/crypto payments, payment gateways
- **Reseller System** — Sub-accounts with credit allocation and customer provisioning
- **Invoices** — Auto-generated invoices with PDF export

### Customer Experience
- **Self-Service Portal** — Single-page app for usage, profiles, VPN configs, support
- **Telegram Bot** — Admin management and customer self-service via inline buttons
- **Ticket System** — Customer support with canned responses and knowledge base

### Admin Dashboard
- **Drag-and-Drop Sidebar** — Customizable navigation with category/item reordering
- **Theming** — Multiple themes with 18 CSS tokens, dark/light/system mode
- **Multi-Language** — English, Persian (RTL), Chinese, Russian
- **Sliding Panels** — Consistent slide-over pattern for all entity creation forms

### Infrastructure
- **Two Editions** — Full and Lite from same codebase via Go build tags
- **Docker Deploy** — TimescaleDB + pgAdmin + Panel in one compose file
- **Auto-TLS** — Let's Encrypt (ACME), manual cert, or self-signed modes
- **HTTPS Enforced** — External traffic requires TLS; HTTP restricted to loopback
- **Decoy Landing Page** — Neutral business content with VPN blocklist validation

---

## Quick Install

### Docker (recommended)

```bash
bash <(curl -Ls https://raw.githubusercontent.com/anonysec/koris/main/install.sh)
```

Running without flags launches an interactive prompt for edition, domain, port, DB, and SSL configuration. If an existing installation is detected, the installer offers reinstall, full wipe, update, or cancel options before proceeding.

Options:
```bash
install.sh --lite              # Lite edition
install.sh --full             # Full edition (default)
install.sh --port=8080         # Custom port
install.sh --domain=panel.example.com
install.sh --no-knode          # Skip knode agent installation
install.sh --version=v1.2.0    # Install a specific version tag
install.sh --reinstall         # Force reinstall (preserves DB data)
install.sh --uninstall         # Remove KorisPanel
```

### Node Agent (on each VPN server)

```bash
bash <(curl -Ls https://raw.githubusercontent.com/anonysec/knode/master/install.sh)
```

---

## Architecture

```
                    Panel Server (Docker)
 +-------------------------------------------------+
 |  Panel Binary (:443 HTTPS / :8080 HTTP local)   |
 |      |                                          |
 |      v                                          |
 |  TimescaleDB (pg16)     pgAdmin (:5050)         |
 |      (metrics + data)                           |
 +-------------------------------------------------+
          |  gRPC + mTLS (bidirectional sync)
          v
 +-------------------------------------------------+
 |              knode Server(s)                     |
 |                                                  |
 |  gRPC/REST API --> OpenVPN, WireGuard, L2TP,    |
 |                    IKEv2, SSH, MTProto           |
 |                --> Outbound tunnels              |
 |                --> Traffic shaping               |
 +-------------------------------------------------+
```

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.25+, `net/http` (no framework) |
| Frontend | Vue 3, TypeScript, Vite, Pinia |
| Database | TimescaleDB (PostgreSQL 16) |
| Node Communication | gRPC with mTLS, protobuf |
| Frontend Workspaces | pnpm monorepo (admin, portal, landing, core, theme) |
| Testing | gopter (Go PBT), fast-check (TS PBT), vitest |
| Containerization | Docker Compose |

---

## Development

```bash
# Clone
git clone https://github.com/anonysec/koris.git
cd koris

# Backend
go run ./cmd/panel

# Frontend (pnpm workspace)
cd web
pnpm install
pnpm --filter admin dev     # Admin dashboard
pnpm --filter portal dev    # Customer portal
pnpm --filter landing dev   # Marketing landing page

# Build everything
make build          # Frontend + full backend
make build-lite     # Frontend + lite backend

# Tests
make test           # Go tests
make test-frontend  # Vitest
```

---

## Build Commands

```bash
make frontend       # Build all frontend apps (admin, portal, landing)
make backend        # Build full Go binary
make backend-lite   # Build lite Go binary (excludes billing / reseller / bot)
make build          # frontend + backend
make build-lite     # frontend + backend-lite
make test           # go test ./...
make test-frontend  # pnpm test
make clean          # Remove all build artifacts
```

---

## Editions

Both editions are built from this single repository using Go build tags:

| Feature | Full | Lite |
|---------|------|------|
| Node management, users, VPN protocols | ✓ | ✓ |
| gRPC sync, monitoring, health checks | ✓ | ✓ |
| Backup, certificates, sessions | ✓ | ✓ |
| Billing, invoices, payment gateways | ✓ | ✗ |
| Xray, MTProto, AnyConnect | ✓ | ✗ |
| Tickets, knowledge base, SLA | ✓ | ✗ |
| Reseller, LDAP, reports, segments | ✓ | ✗ |

```bash
go build ./cmd/panel              # Full
go build -tags lite ./cmd/panel   # Lite
```

The `/api/info` endpoint returns `{"edition": "full"}` or `{"edition": "lite"}`.

---

## Configuration

Panel config directory: `/etc/koris/`
- `panel.env` — environment variables for the Docker stack
- `version` — installed version tag (written on install/update)

| Variable | Description | Default |
|----------|-------------|---------|
| `PANEL_ADDR` | HTTP listen address | `127.0.0.1:8080` |
| `PANEL_TLS_ADDR` | HTTPS listen address | `0.0.0.0:443` |
| `PANEL_TLS_MODE` | TLS mode (acme/manual/selfsigned) | `selfsigned` |
| `PANEL_TLS_CERT` | Cert path (manual mode) | — |
| `PANEL_TLS_KEY` | Key path (manual mode) | — |
| `PANEL_DOMAIN` | Domain for ACME | — |
| `POSTGRES_DB` | Database name | `koris` |
| `POSTGRES_USER` | Database user | `koris` |
| `POSTGRES_PASSWORD` | Database password | — |
| `PANEL_ADMIN_PATH` | Admin panel URL prefix | `/admin/` |
| `PANEL_PORTAL_PATH` | Customer portal URL prefix | `/account/` |
| `PANEL_ADMIN_HOST` | Serve admin at subdomain (empty = path routing) | — |
| `PANEL_PORTAL_HOST` | Serve portal at subdomain (empty = path routing) | — |

### URL routing

By default the panel serves the admin dashboard at `/admin/` and the customer
portal at `/account/`. Legacy paths `/dashboard/` and `/portal/` redirect to
the new locations for backward compatibility.

**Path mode** (default) — everything on one host:
```
https://your-domain/admin/       # admin dashboard
https://your-domain/account/     # customer portal
https://your-domain/             # decoy landing page
```

**Subdomain mode** — dedicated subdomains:
```yaml
# In panel.env:
PANEL_ADMIN_HOST=admin.your-domain
PANEL_PORTAL_HOST=account.your-domain
PANEL_ADMIN_PATH=/     # path becomes root under the subdomain
PANEL_PORTAL_PATH=/
```

The interactive installer prompts for this at setup; also settable
via `--admin-path=`, `--portal-path=`, `--admin-host=`, `--portal-host=`
flags. Changing after install: edit `/etc/koris/panel.env` and
`docker compose up -d panel` (also rebuild if you changed a path — the
Vite base path is baked in at build time via `KORIS_ADMIN_BASE` /
`KORIS_PORTAL_BASE`).

---

## Theming

The frontend lives in two swappable packages:

| Package | Role | Swappable? |
|---------|------|------------|
| `@koris/core` | Framework layer — composables, API client, types, base CSS (reset/tokens/utilities) | **No** — required |
| `@koris/theme` | Visual layer — 30+ UI components (Button, DataTable, Modal, …), component CSS, polish CSS | **Yes** — replace to reskin |

To create a custom theme:
1. Copy `web/theme/` to `web/themes/my-theme/`
2. Edit `manifest.ts` — change `id`, `name`, `author`
3. Override any subset of `.vue` components or `.css` files
4. Update the Vite alias `@koris/theme` in each app to point at your directory

Both apps (admin, portal) resolve visual components via the alias, so a
theme swap doesn'''t touch app code. See `web/theme/manifest.ts` for the
component slot registry.

---

## Docker Services

| Service | Image | Port |
|---------|-------|------|
| `panel` | Built from source | 443 (HTTPS), 80 (HTTP) |
| `knode` | Built from anonysec/knode | 62050 (gRPC, per-node) |
| `koris-db` | `timescale/timescaledb:latest-pg16` | 5432 (internal) |
| `pgadmin` | `dpage/pgadmin4` | 5050 (localhost only) |

---

## License

Private repository. All rights reserved.
