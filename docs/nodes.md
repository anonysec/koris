# 🛰️ Node Management

VPN nodes run the [**knode**](https://github.com/anonysec/knode) agent. Koris talks to each node over **gRPC secured with mTLS**; nodes never poll — the panel pushes desired state and receives streamed metrics.

---

## Adding a node

1. **Install knode** on the VPN server:
   ```bash
   bash <(curl -Ls https://raw.githubusercontent.com/anonysec/knode/master/install.sh) --port=2083
   ```
2. In the admin UI, go to **Nodes → Add Node** and enter:
   - Host / IP and knode port (default `2083`)
   - The node's API key (from its `config.toml`)
3. Koris performs an mTLS handshake, registers the node, and begins streaming metrics.

The Koris installer can also auto-provision a bundled knode unless you pass `--no-knode`.

---

## What a node exposes (gRPC)

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

---

## Protocols

Each node can run any subset of: **OpenVPN, WireGuard, L2TP/IPsec, IKEv2, SSH tunnel, MTProto**, plus outbound tunnels (VLESS+Reality, WireGuard, SSH, Rathole, GRE/IPIP).

Enable/disable per node from **Nodes → *node* → Cores**.

---

## Fleet features

- 🧩 **Node groups** — organise nodes by region/role
- ⚖️ **Load balancing** — distribute users across nodes
- 🔄 **Migrate / provision** — move users, spin up new nodes
- 📊 **Compare** — side-by-side node metrics
- 🔐 **Certificate rotation** — panel-driven mTLS cert rollover

---

## Health & recovery

- Nodes stream health continuously; the panel flags degraded/offline nodes.
- knode auto-restarts failed cores and hot-reloads config on `SIGHUP`.
- Certificate expiry is tracked and rotated proactively (`internal/certrotation`).

See also: [Architecture →](architecture.md)
