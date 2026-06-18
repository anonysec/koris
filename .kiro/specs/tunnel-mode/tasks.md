# Implementation Plan: Tunnel Mode (Iran Traffic)

## Overview

This plan implements the Tunnel Mode feature for KorisPanel — a two-hop VPN architecture where customers connect to Iran-based entry nodes, which forward traffic through encrypted WireGuard tunnels to foreign outbound nodes. The implementation covers: database schema, backend tunnel service (subnet allocation, key generation, pair lifecycle, health monitoring, failover), node agent task handlers (setup, teardown, failover, health reporting), admin API endpoints, frontend tunnel management views, and property-based tests.

## Tasks

- [ ] 1. Database schema and core backend modules
  - [ ] 1.1 Create SQL migration for tunnel mode schema
    - Create `panel/migrations/0XX_tunnel_mode.sql` (use next sequential number)
    - Add `tunnel_role ENUM('none','entry','outbound') NOT NULL DEFAULT 'none'` column to `nodes` table
    - Create `tunnel_pairs` table with all fields (id, entry_node_id, outbound_node_id, tunnel_subnet, WireGuard keys encrypted, tunnel_port, mtu, interface_name, status, priority, health_status, last_health_check_at, error_message, timestamps)
    - Create `tunnel_stats` table (id, tunnel_pair_id, entry/outbound tx/rx bytes, recorded_at)
    - Create `tunnel_events` table (id, tunnel_pair_id, event_type, from/to outbound_id, reason, created_at)
    - Add indexes: `idx_entry_status(entry_node_id, status)`, `idx_outbound_status(outbound_node_id, status)`, `uniq_subnet(tunnel_subnet)`, `idx_pair_time(tunnel_pair_id, recorded_at)`, `idx_pair_events(tunnel_pair_id, created_at)`
    - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5_

  - [ ] 1.2 Create tunnel subnet allocator `panel/internal/tunnel/subnet.go`
    - Implement `AllocateNextSubnet(usedSubnets []string) (subnet, entryIP, outboundIP string, err error)`
    - Allocate sequential /30 subnets from 10.200.0.0/16 range
    - Entry gets .1, outbound gets .2 within each /30
    - Implement `ParseTunnelSubnet(subnet string) (entryIP, outboundIP string, err error)`
    - Return error if pool is exhausted (16,384 possible /30 subnets)
    - _Requirements: 2.3, 3.1, 3.2_

  - [ ] 1.3 Create tunnel key generation `panel/internal/tunnel/keygen.go`
    - Implement `GenerateTunnelKeyPair() (privateKey, publicKey string, err error)` using `crypto/rand` + Curve25519
    - Reuse existing WireGuard key generation pattern from `panel/internal/wireguard/` if available
    - Keys must be 44-char base64 encoding of 32 random bytes
    - _Requirements: 2.2_

  - [ ] 1.4 Create tunnel health classifier `panel/internal/tunnel/health.go`
    - Implement `ClassifyHealth(handshakeAgeSec int) string` returning "healthy" (<30s), "degraded" (30-90s), "down" (≥90s)
    - Implement `ParseWireGuardDump(output string) []PeerStatus` to parse `wg show dump` output
    - _Requirements: 5.1, 5.2, 5.3_

  - [ ]* 1.5 Write property test for subnet allocation uniqueness (Property 1)
    - **Property 1: Tunnel subnet allocation uniqueness**
    - Use `pgregory.net/rapid` to generate random sets of used subnets; verify returned subnet doesn't overlap
    - File: `panel/internal/tunnel/subnet_test.go`
    - **Validates: Requirements 2.3, 11.2**

  - [ ]* 1.6 Write property test for subnet parsing validity (Property 2)
    - **Property 2: Tunnel subnet parsing produces valid entry/outbound IPs**
    - Generate all valid /30 subnets; verify entryIP ends in .1 and outboundIP ends in .2
    - File: `panel/internal/tunnel/subnet_test.go`
    - **Validates: Requirements 2.3, 3.1, 3.2**

  - [ ]* 1.7 Write property test for health classification (Property 4)
    - **Property 4: Tunnel health classification from handshake age**
    - Generate random non-negative integers; verify classification boundaries (0-29=healthy, 30-89=degraded, 90+=down)
    - File: `panel/internal/tunnel/health_test.go`
    - **Validates: Requirements 5.1, 5.2, 5.3**

  - [ ]* 1.8 Write property test for key generation (Property 9)
    - **Property 9: WireGuard key generation produces valid keys for tunnel**
    - Generate multiple key pairs; verify 44-char base64 → 32 bytes, and entry_pub ≠ outbound_pub
    - File: `panel/internal/tunnel/keygen_test.go`
    - **Validates: Requirements 2.2**

- [ ] 2. Backend tunnel service (pair lifecycle, role management)
  - [ ] 2.1 Create tunnel service `panel/internal/tunnel/service.go`
    - Implement `CreateTunnelPair(entryNodeID, outboundNodeID int64, priority int) (*TunnelPair, error)`
    - Validate both nodes exist and have correct roles (entry/outbound)
    - Validate entry ≠ outbound (same_node check)
    - Check max pairs per entry node (default 3, configurable)
    - Generate WireGuard key pairs for both sides
    - Allocate tunnel subnet
    - Insert tunnel_pairs record with status "pending"
    - Dispatch `tunnel.setup_entry` and `tunnel.setup_outbound` tasks
    - Insert tunnel_events record (type=created)
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 3.1, 3.2_

  - [ ] 2.2 Implement node role management in tunnel service
    - Implement `SetNodeTunnelRole(nodeID int64, role string) error`
    - Validate role is one of: "none", "entry", "outbound"
    - Reject role change if node has active tunnel pairs (status in pending/active/error)
    - Update `nodes.tunnel_role` column
    - _Requirements: 1.2, 1.3, 1.4, 1.5_

  - [ ] 2.3 Implement tunnel pair removal in service
    - Implement `RemoveTunnelPair(pairID int64) error`
    - If pair is active primary with backup available, trigger failover first
    - Dispatch `tunnel.teardown` tasks to both entry and outbound nodes
    - Set pair status to "removed"
    - Insert tunnel_events record (type=removed)
    - Idempotent: skip if already removed
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_

  - [ ] 2.4 Implement tunnel pair listing and detail queries
    - `ListTunnelPairs(filters) ([]TunnelPair, error)` — join with nodes for names/IPs
    - `GetTunnelPair(id int64) (*TunnelPairDetail, error)` — include recent stats
    - `ListTunnelEvents(pairID *int64, limit int) ([]TunnelEvent, error)`
    - `ListNodesByRole(role string) ([]Node, error)`
    - _Requirements: 12.1, 12.4_

  - [ ]* 2.5 Write property test for role validation (Property 3)
    - **Property 3: Node role validation prevents dual-role assignment**
    - Generate random node states with active pairs; verify role changes are correctly rejected
    - File: `panel/internal/tunnel/service_test.go`
    - **Validates: Requirements 1.4, 1.5**

  - [ ]* 2.6 Write property test for max pairs enforcement (Property 10)
    - **Property 10: Maximum tunnel pairs per entry node enforcement**
    - Generate random entry nodes with varying pair counts; verify limit enforcement
    - File: `panel/internal/tunnel/service_test.go`
    - **Validates: Requirements 2.6**

  - [ ]* 2.7 Write property test for removal idempotency (Property 6)
    - **Property 6: Tunnel pair removal is idempotent**
    - Generate pairs in various states; verify double-removal doesn't create duplicate events/tasks
    - File: `panel/internal/tunnel/service_test.go`
    - **Validates: Requirements 8.1, 8.4**

- [ ] 3. Backend failover controller
  - [ ] 3.1 Create failover controller `panel/internal/tunnel/failover.go`
    - Implement `CheckAndFailover(entryNodeID int64) error`
    - Query active tunnel pairs for entry node ordered by priority
    - If primary (lowest priority) has health_status "down", select next healthy backup
    - Dispatch `tunnel.failover` task to entry node
    - Insert tunnel_events record (type=failover)
    - Send admin alert notification if no backup available
    - _Requirements: 6.1, 6.2, 6.3, 6.6_

  - [ ] 3.2 Implement failback logic
    - Implement `CheckAndFailback(entryNodeID int64) error`
    - If primary tunnel pair recovers (health returns to "healthy") and entry is currently using backup, dispatch `tunnel.failback` task
    - Insert tunnel_events record (type=failback)
    - _Requirements: 6.4, 6.5_

  - [ ] 3.3 Integrate health monitoring with failover in status push handler
    - In `/api/node/push` handler, parse `tunnel_health` array from entry node push
    - Update `tunnel_pairs.health_status` and `last_health_check_at`
    - Store traffic bytes in `tunnel_stats` (hourly aggregation)
    - Call `CheckAndFailover` when health transitions to "down"
    - Call `CheckAndFailback` when primary health transitions to "healthy"
    - _Requirements: 5.2, 5.4, 5.5, 7.1, 7.2, 7.3_

  - [ ]* 3.4 Write property test for failover priority selection (Property 5)
    - **Property 5: Failover selects next priority backup**
    - Generate random priority sets with various health states; verify correct backup selection
    - File: `panel/internal/tunnel/failover_test.go`
    - **Validates: Requirements 6.1, 6.2**

  - [ ]* 3.5 Write property test for traffic stats aggregation (Property 8)
    - **Property 8: Traffic statistics aggregation preserves totals**
    - Generate random sequences of traffic byte values; verify sums and monotonicity
    - File: `panel/internal/tunnel/stats_test.go`
    - **Validates: Requirements 7.1, 7.3**

- [ ] 4. Checkpoint - Ensure all backend core tests pass
  - Run `go test ./panel/internal/tunnel/...`
  - Ensure all property tests and unit tests pass
  - Ask the user if questions arise

- [ ] 5. Node agent tunnel task handlers
  - [ ] 5.1 Create tunnel task handler file `node/cmd/node/tunnel.go`
    - Implement `executeTunnelSetupEntry(payload map[string]any) (string, map[string]any, string)` task handler
    - Write WireGuard config to `/etc/wireguard/{interface_name}.conf`
    - Execute `wg-quick up {interface_name}`
    - Apply iptables mangle PREROUTING rules to mark VPN traffic (tun+, ppp+, wg0) with fwmark 0x100
    - Add ip rule (fwmark 0x100 table 100 priority 100) and ip route (default via outbound tunnel IP)
    - Apply iptables FORWARD rules for bidirectional traffic
    - Apply tc fq_codel qdisc on tunnel interface
    - Apply DSCP EF marking via iptables mangle POSTROUTING
    - _Requirements: 3.3, 4.1, 4.3, 10.1, 10.2, 10.3_

  - [ ] 5.2 Implement `tunnel.setup_outbound` task handler
    - Write WireGuard config to `/etc/wireguard/{interface_name}.conf`
    - Execute `wg-quick up {interface_name}`
    - Enable ip_forward via sysctl (persist to /etc/sysctl.d/99-tunnel.conf)
    - Detect WAN interface from default route
    - Apply iptables NAT MASQUERADE rule for tunnel subnet traffic on WAN interface
    - Apply iptables FORWARD rules for bidirectional traffic
    - _Requirements: 3.4, 4.2_

  - [ ] 5.3 Implement `tunnel.teardown` task handler
    - Execute `wg-quick down {interface_name}`
    - Remove WireGuard config file
    - Remove iptables rules (mangle PREROUTING mark, FORWARD, nat MASQUERADE, mangle POSTROUTING DSCP)
    - Remove ip rule and ip route entries for tunnel table
    - Remove tc qdisc if present
    - _Requirements: 8.2, 8.3, 10.4_

  - [ ] 5.4 Implement `tunnel.failover` task handler
    - Create new tunnel interface with backup config (new WireGuard keys, new outbound endpoint)
    - Switch ip rule/route to point to new tunnel interface
    - Bring down old tunnel interface
    - Update iptables FORWARD rules to reference new interface
    - _Requirements: 6.3_

  - [ ] 5.5 Implement `tunnel.failback` task handler
    - Verify primary tunnel is healthy (handshake check)
    - Switch ip rule/route back to primary tunnel interface
    - Bring down backup interface if no longer needed
    - _Requirements: 6.4_

  - [ ] 5.6 Extend node status push with tunnel health data
    - For each `wg-tunnel*` interface, run `wg show {iface} dump`
    - Parse latest handshake and transfer bytes
    - Classify health status using handshake age (healthy/degraded/down)
    - Add `tunnel_health` array and `tunnel_active_count` to push payload
    - _Requirements: 5.1, 5.2, 7.1, 7.2_

  - [ ] 5.7 Register tunnel task actions in `executeTask` switch
    - Add cases for: `tunnel.setup_entry`, `tunnel.setup_outbound`, `tunnel.teardown`, `tunnel.failover`, `tunnel.failback`
    - Route to handlers implemented in tunnel.go
    - _Requirements: 3.1, 3.2, 8.1_

  - [ ]* 5.8 Write property test for routing rules protocol coverage (Property 7)
    - **Property 7: Entry node routing rules are protocol-agnostic**
    - Generate random VPN interface name sets; verify all get marked with same fwmark
    - File: `node/cmd/node/tunnel_test.go`
    - **Validates: Requirements 4.1, 4.3**

- [ ] 6. Checkpoint - Ensure node agent tests pass
  - Run `go test ./node/cmd/node/...`
  - Ensure all tunnel-related tests pass
  - Ask the user if questions arise

- [ ] 7. Panel API endpoints for tunnel management
  - [ ] 7.1 Create tunnel API handler file `panel/internal/api/tunnel.go`
    - `POST /api/admin/tunnels` — Create tunnel pair (validate roles, call service.CreateTunnelPair)
    - `GET /api/admin/tunnels` — List tunnel pairs with optional filters (status, entry_node_id, outbound_node_id)
    - `GET /api/admin/tunnels/{id}` — Get tunnel pair detail with recent stats
    - `DELETE /api/admin/tunnels/{id}` — Remove tunnel pair (call service.RemoveTunnelPair)
    - `POST /api/admin/tunnels/{id}/failover` — Manual failover trigger
    - `GET /api/admin/tunnels/events` — List tunnel events (optional pair_id filter, paginated)
    - _Requirements: 12.1, 12.2, 12.4_

  - [ ] 7.2 Create node role API endpoints
    - `PUT /api/admin/nodes/{id}/tunnel-role` — Set node tunnel role (validate, call service.SetNodeTunnelRole)
    - `GET /api/admin/nodes/entry` — List entry nodes
    - `GET /api/admin/nodes/outbound` — List outbound nodes
    - _Requirements: 1.1, 1.2, 1.3_

  - [ ] 7.3 Register tunnel routes in API router
    - Add all tunnel routes to the existing `http.ServeMux` in `panel/internal/api/api.go`
    - Apply admin auth middleware to all tunnel endpoints
    - _Requirements: 12.1_

  - [ ] 7.4 Extend `/api/node/push` handler for tunnel health ingestion
    - Parse `tunnel_health` field from push payload
    - For each tunnel health report: update `tunnel_pairs.health_status` and `last_health_check_at`
    - Accumulate traffic bytes into `tunnel_stats` (upsert hourly bucket)
    - Trigger failover/failback checks on health transitions
    - _Requirements: 5.2, 5.4, 7.1, 7.2, 7.3_

- [ ] 8. Checkpoint - Full backend integration test
  - Run `go test ./...`
  - Verify tunnel API endpoints respond correctly
  - Ask the user if questions arise

- [ ] 9. Frontend admin — Tunnel management views
  - [ ] 9.1 Create `useTunnels` composable `panel/web/admin/src/composables/useTunnels.ts`
    - Functions: `fetchTunnels(filters)`, `createTunnel(data)`, `deleteTunnel(id)`, `getTunnelDetail(id)`, `triggerFailover(id)`, `fetchEvents(pairId?)`, `setNodeTunnelRole(nodeId, role)`, `fetchEntryNodes()`, `fetchOutboundNodes()`
    - _Requirements: 12.1, 12.2, 1.1_

  - [ ] 9.2 Create TunnelListView `panel/web/admin/src/views/tunnels/TunnelListView.vue`
    - Table columns: Entry Node, Outbound Node, Tunnel Subnet, Status, Priority, Health (colored indicator), Last Check, Actions
    - Health indicators: green (healthy), yellow (degraded), red (down), gray (unknown)
    - Create button opens TunnelCreateDialog
    - Delete button with confirmation
    - Manual failover button (visible when backup exists)
    - Register route: `/tunnels` in admin router
    - _Requirements: 12.1, 12.3_

  - [ ] 9.3 Create TunnelCreateDialog `panel/web/admin/src/views/tunnels/TunnelCreateDialog.vue`
    - Select Entry_Node from dropdown (filtered by role=entry)
    - Select Outbound_Node from dropdown (filtered by role=outbound)
    - Priority input (default 1)
    - Calls POST `/api/admin/tunnels` on submit
    - _Requirements: 12.2_

  - [ ] 9.4 Create TunnelDetailView `panel/web/admin/src/views/tunnels/TunnelDetailView.vue`
    - Show entry/outbound node info, IPs, tunnel subnet, WireGuard public keys
    - Traffic chart (entry TX, outbound RX) using chart.js or similar
    - Failover history for this pair
    - Register route: `/tunnels/:id`
    - _Requirements: 12.5, 7.4_

  - [ ] 9.5 Create TunnelEventsView `panel/web/admin/src/views/tunnels/TunnelEventsView.vue`
    - Table: timestamp, pair, event type, from/to outbound, reason
    - Filter by tunnel pair
    - Register route: `/tunnels/events`
    - _Requirements: 12.4_

  - [ ] 9.6 Create TunnelRoleSelector component `panel/web/admin/src/components/TunnelRoleSelector.vue`
    - Dropdown with options: None, Entry (Iran), Outbound (Foreign)
    - Integrate into existing node edit view/dialog
    - Calls PUT `/api/admin/nodes/{id}/tunnel-role` on change
    - _Requirements: 1.1_

  - [ ] 9.7 Add tunnel health indicators to node list view
    - Show tunnel role badge on node list (Entry/Outbound)
    - Show tunnel health summary for entry nodes (number of healthy/down tunnels)
    - _Requirements: 5.4, 12.3_

- [ ] 10. Final checkpoint - Ensure all tests pass
  - Run `go test ./...` for backend
  - Run `npm run test` in `panel/web/admin` for frontend
  - Verify complete tunnel lifecycle works: create pair → setup → active → health reporting → failover → teardown
  - Ask the user if questions arise

## Notes

- Tasks marked with `*` are property-based tests and can be skipped for faster MVP delivery
- The existing `node/cmd/node/outbound.go` handles protocol-level outbound proxying (SOCKS5/VLESS/VMess). Tunnel mode is a different concept — it operates at the IP layer, forwarding ALL VPN traffic through a WireGuard tunnel. These two features are complementary and don't conflict.
- The tunnel WireGuard interface (`wg-tunnel0`) is separate from any customer-facing WireGuard interface (`wg0`), avoiding configuration conflicts
- The tunnel port (default 51821) is distinct from the customer WireGuard port (default 51820) to prevent conflicts when both are running
- Key encryption uses the same `encryptToken` / `decryptToken` functions from `panel/internal/api/failover.go` (AES-256-GCM with PANEL_SECRET)
- The `/30` subnet allocation strategy supports up to 16,384 tunnel pairs which is far beyond expected usage
- Health monitoring piggybacks on the existing node agent push cycle (configurable interval, default 10s), no separate health-check goroutine needed
- Failover latency = push interval (10s) + panel detection + task dispatch + next poll (10s) ≈ 20-30 seconds worst case
- Property tests use `pgregory.net/rapid` for Go as specified in the tech stack

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1", "1.2", "1.3", "1.4"] },
    { "id": 1, "tasks": ["1.5", "1.6", "1.7", "1.8", "2.1", "2.2"] },
    { "id": 2, "tasks": ["2.3", "2.4", "2.5", "2.6", "2.7"] },
    { "id": 3, "tasks": ["3.1", "3.2"] },
    { "id": 4, "tasks": ["3.3", "3.4", "3.5"] },
    { "id": 5, "tasks": ["5.1", "5.2", "5.3", "5.4", "5.5"] },
    { "id": 6, "tasks": ["5.6", "5.7", "5.8"] },
    { "id": 7, "tasks": ["7.1", "7.2", "7.3", "7.4"] },
    { "id": 8, "tasks": ["9.1", "9.2", "9.3", "9.6"] },
    { "id": 9, "tasks": ["9.4", "9.5", "9.7"] }
  ]
}
```

