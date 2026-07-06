# 🏛️ Architecture

Koris is a **control plane** for a fleet of VPN nodes. It never sits in the data path — it pushes desired state to nodes and streams their metrics back.

```
                         ┌───────────────────────────────┐
                         │            Koris Panel 🛡️      │
                         │                               │
   Browser ── HTTPS ──▶  │  Vue 3 SPA (admin / portal /  │
                         │  landing, embedded via embed) │
                         │            +                  │
   Telegram ─────────▶   │  Go API (./cmd/panel)         │
                         └───────┬───────────────┬───────┘
                                 │ gRPC+mTLS      │ SQL
                                 ▼                ▼
                        ┌────────────────┐  ┌──────────────┐
                        │   knode 🛰️     │  │ TimescaleDB  │
                        │  (per VPN node)│  │      🐘       │
                        └────────────────┘  └──────────────┘
```

---

## Components

### Backend — `./cmd/panel` (Go 1.25)
A single binary. ~40 internal packages under `internal/`:

| Domain | Packages |
|--------|----------|
| API & auth | `api`, `auth`, `sessions`, `csrf`, `ratelimit`, `ldap` |
| Nodes | `grpcclient`, `noderegistry`, `cluster`, `loadbalance`, `certrotation`, `knodepb` |
| Business | `billing`, `payment`, `reports`, `stats` |
| Support | `support`, `notify`, `alerts`, `templates` |
| Platform | `config`, `db`, `dbstore`, `cache`, `worker`, `cleanup`, `backup`, `updater`, `health`, `logger` |
| Networking | `protocols`, `proxyconfig`, `antidpi`, `teleproxy` |
| UX | `landing`, `tui`, `bot` |

### Frontend — `web/` (pnpm workspace)
| Workspace | Purpose |
|-----------|---------|
| `admin` | Admin dashboard SPA (~50 views) |
| `portal` | Customer self-service SPA |
| `landing` | Public decoy landing page |
| `core` | Shared design tokens, styles, composables |
| `theme` | Shared components + theme CSS (incl. `overhaul.css`) |
| `themes/*` | Fan-made / alternate skins |

All three apps are compiled and **embedded into the Go binary** via `web/embed.go` (`go:embed`), so the release artifact is a single self-contained executable.

### Data — TimescaleDB (Postgres 16)
Relational data + time-series metrics. Migrations live in `migrations/` and are applied on boot.

### Nodes — [knode](https://github.com/anonysec/knode)
Push-based agents on each VPN server. See the knode repo for its internals.

---

## Editions

Two editions from one codebase via Go build tags:

| Edition | Build | Notes |
|---------|-------|-------|
| **Full** | *(no tag)* | All features |
| **Lite** | `-tags lite` | Smaller, fewer optional subsystems |

Files like `cmd/panel/services_full.go` / `services_lite.go` gate the difference.

---

## Request flow (example: enable a protocol on a node)

1. Admin toggles a core in the SPA → `POST /api/nodes/:id/cores/:core/enable`.
2. API validates session + CSRF, checks RBAC.
3. `grpcclient` opens an mTLS gRPC call to the node's `EnableCore` RPC.
4. knode configures the backend (e.g. WireGuard) and returns status.
5. knode's `StreamMetrics` reports live state; panel persists it to TimescaleDB.

See also: [Node Management →](nodes.md)
