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

The repo also builds two companion binaries (Full edition only):
- **`cmd/worker`** — a gRPC server (`korispb.WorkerService`) owning the job queue (`internal/jobs`: billing, invoice, email, report jobs). The panel enqueues work and polls `GetJobStatus`.
- **`cmd/gateway`** — a TLS-terminating reverse proxy doing rate limiting (`internal/ratelimit`), API-key auth (`internal/gateway`), and TLS handling.

### Frontend — `web/` (pnpm workspace)

| Workspace | Purpose |
|-----------|---------|
| `admin` | Admin dashboard SPA (~50 views) |
| `portal` | Customer self-service SPA |
| `landing` | Public decoy landing page |
| `core` | Shared design tokens, styles, composables |
| `theme` | Shared components + theme CSS (incl. `overhaul.css`) |
| `themes/*` | Fan-made / alternate skins |

All three apps are compiled and **embedded into the Go binary** via `web/embed.go` (`go:embed`), so the release artifact is a single self-contained executable. See [UI / UX](ui-ux.md) for the design system.

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
| **Lite** | `-tags lite` | Smaller, strips billing/payment/support subsystems (and related API routes/handlers) |

Files like `cmd/panel/services_full.go` / `services_lite.go` gate the difference. When modifying shared features, keep both the `_full.go` and `_lite.go` counterparts in sync.

---

## Request flow (example: enable a protocol on a node)

1. Admin toggles a core in the SPA → `POST /api/nodes/:id/cores/:core/enable`.
2. API validates session + CSRF, checks RBAC.
3. `grpcclient` opens an mTLS gRPC call to the node's `EnableCore` RPC.
4. knode configures the backend (e.g. WireGuard) and returns status.
5. knode's `StreamMetrics` reports live state; panel persists it to TimescaleDB.

---

## Node management (knode)

VPN nodes run the [**knode**](https://github.com/anonysec/knode) agent. Koris talks to each node over **gRPC secured with mTLS**; nodes never poll — the panel pushes desired state and receives streamed metrics.

### Adding a node

1. **Install knode** on the VPN server:
   ```bash
   bash <(curl -Ls https://raw.githubusercontent.com/anonysec/knode/master/install.sh) --port=2083
   ```
2. In the admin UI, go to **Nodes → Add Node** and enter the host/IP, knode port (default `2083`), and the node's API key (from its `config.toml`).
3. Koris performs an mTLS handshake, registers the node, and begins streaming metrics.

The Koris installer can also auto-provision a bundled knode unless you pass `--no-knode`.

### What a node exposes (gRPC)

| Capability | RPCs |
|------------|------|
| Health | `Health`, `AllCoreStatuses` |
| Cores (protocols) | `EnableCore`, `DisableCore` |
| Users | `SyncUsers`, `ConnectUser`, `DisconnectUser` |
| Traffic | `GetTraffic`, `ResetTraffic`, `StreamMetrics` |
| Firewall | `OpenPort`, `ClosePort` |
| Certificates | `SetCertificates`, `GenerateClientCert` |
| Tunnels | `SetupTunnel`, `TeardownTunnel` |

Full schema: [`knode/proto/knode/v1/knode.proto`](https://github.com/anonysec/knode/blob/master/proto/knode/v1/knode.proto).

### Protocols

Each node can run any subset of: **OpenVPN, WireGuard, L2TP/IPsec, IKEv2, SSH tunnel, MTProto**, plus outbound tunnels (VLESS+Reality, WireGuard, SSH, Rathole, GRE/IPIP). Enable/disable per node from **Nodes → *node* → Cores**.

### Fleet features

- 🧩 **Node groups** — organise nodes by region/role
- ⚖️ **Load balancing** — distribute users across nodes
- 📊 **Compare** — side-by-side node metrics
- 🔐 **Certificate rotation** — panel-driven mTLS cert rollover

### Health & recovery

- Nodes stream health continuously; the panel flags degraded/offline nodes.
- knode auto-restarts failed cores and hot-reloads config on `SIGHUP`.
- Certificate expiry is tracked and rotated proactively (`internal/certrotation`).

See also: [Installation & Operations →](installation.md), [Configuration →](configuration.md).
