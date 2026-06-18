# Requirements Document

## Introduction

Tunnel Mode enables KorisPanel to route customer VPN traffic through a two-hop architecture: customers connect to an entry node located in Iran, which forwards their encrypted traffic through an inter-node WireGuard tunnel to an outbound node in a foreign country. The outbound node performs NAT/masquerade and routes traffic to the internet. This architecture is essential for the Iran VPN market where direct connections to foreign servers are detected and blocked by Deep Packet Inspection (DPI). The panel manages node role designation, entry-outbound pairing, tunnel provisioning, failover, and traffic statistics.

## Glossary

- **Panel**: The central KorisPanel web application that manages nodes, customers, and VPN services
- **Node_Agent**: The lightweight agent process deployed on each VPN server node that executes tasks and reports status
- **Entry_Node**: A node located in Iran designated to accept customer VPN connections and forward traffic through the inter-node tunnel to an Outbound_Node
- **Outbound_Node**: A node located in a foreign country designated to receive tunneled traffic from Entry_Nodes and route it to the internet via NAT/masquerade
- **Tunnel**: An encrypted WireGuard point-to-point connection between an Entry_Node and an Outbound_Node used to forward customer traffic
- **Tunnel_Pair**: A configured association between one Entry_Node and one Outbound_Node defining a traffic forwarding path
- **Admin_UI**: The Vue 3 admin dashboard SPA for managing all panel resources
- **Task_System**: The existing mechanism where the Panel dispatches tasks to Node_Agent via /api/node/tasks/poll
- **DPI**: Deep Packet Inspection — network-level traffic analysis that can detect and block VPN protocols
- **NAT_Masquerade**: Network Address Translation on the Outbound_Node that rewrites tunneled traffic source addresses to the outbound node's public IP before forwarding to the internet
- **Failover**: Automatic switching of an Entry_Node's tunnel target from a failed Outbound_Node to a healthy backup Outbound_Node
- **Health_Check**: Periodic connectivity verification between an Entry_Node and its paired Outbound_Node via ICMP or WireGuard handshake status
- **Tunnel_Interface**: The WireGuard network interface (wg-tunnel0) created on both Entry_Node and Outbound_Node for the inter-node tunnel

## Requirements

### Requirement 1: Node Role Designation

**User Story:** As an administrator, I want to designate nodes as "entry" or "outbound" roles, so that the panel can manage tunnel mode traffic routing between Iran-based and foreign nodes.

#### Acceptance Criteria

1. THE Admin_UI SHALL provide a "Tunnel Role" selector on the node configuration page with options: "none", "entry", and "outbound"
2. WHEN an administrator sets a node's tunnel role to "entry", THE Panel SHALL store the role in the nodes table and mark the node as eligible for tunnel pairing as an Entry_Node
3. WHEN an administrator sets a node's tunnel role to "outbound", THE Panel SHALL store the role in the nodes table and mark the node as eligible for tunnel pairing as an Outbound_Node
4. THE Panel SHALL prevent assigning both "entry" and "outbound" roles to the same node simultaneously
5. IF a node with active Tunnel_Pairs has its role changed to "none", THEN THE Panel SHALL reject the change and return an error indicating active tunnels must be removed first

### Requirement 2: Tunnel Pair Configuration

**User Story:** As an administrator, I want to pair entry nodes with outbound nodes, so that customer traffic from Iran is forwarded through encrypted tunnels to foreign servers.

#### Acceptance Criteria

1. THE Admin_UI SHALL provide an interface to create Tunnel_Pairs by selecting one Entry_Node and one Outbound_Node
2. WHEN a Tunnel_Pair is created, THE Panel SHALL generate WireGuard key pairs for both the entry and outbound sides of the tunnel
3. WHEN a Tunnel_Pair is created, THE Panel SHALL assign a private tunnel subnet (from 10.200.0.0/16 range) for the point-to-point link between the two nodes
4. THE Panel SHALL support multiple Entry_Nodes paired with the same Outbound_Node (many-to-one mapping)
5. THE Panel SHALL store the Tunnel_Pair record with entry_node_id, outbound_node_id, tunnel subnet, WireGuard keys, status, and priority
6. IF the selected Entry_Node already has the maximum number of active tunnel pairs (configurable, default 3), THEN THE Panel SHALL reject the creation with a descriptive error

### Requirement 3: Tunnel Provisioning on Nodes

**User Story:** As an administrator, I want tunnels to be automatically set up on both entry and outbound nodes when a pair is created, so that traffic forwarding begins without manual server configuration.

#### Acceptance Criteria

1. WHEN a Tunnel_Pair is created, THE Panel SHALL dispatch a "tunnel.setup_entry" task to the Entry_Node containing the outbound node's public IP, tunnel WireGuard keys, and tunnel subnet
2. WHEN a Tunnel_Pair is created, THE Panel SHALL dispatch a "tunnel.setup_outbound" task to the Outbound_Node containing the entry node's public IP, tunnel WireGuard keys, and tunnel subnet
3. WHEN the Entry_Node receives a "tunnel.setup_entry" task, THE Node_Agent SHALL create a WireGuard Tunnel_Interface (wg-tunnel0 or wg-tunnelN), configure routing to forward all customer VPN traffic through the tunnel, and apply iptables FORWARD rules
4. WHEN the Outbound_Node receives a "tunnel.setup_outbound" task, THE Node_Agent SHALL create a WireGuard Tunnel_Interface, enable IP forwarding, and apply NAT_Masquerade rules (iptables MASQUERADE on the outbound network interface) for tunneled traffic
5. WHEN both nodes complete their setup tasks, THE Panel SHALL update the Tunnel_Pair status to "active"
6. IF either node fails to set up the tunnel, THEN THE Panel SHALL set the Tunnel_Pair status to "error" and store the failure reason

### Requirement 4: Customer Traffic Routing

**User Story:** As a customer, I want to connect to an Iran-based entry node with any supported VPN protocol and have my traffic routed to the internet through the outbound server, without needing to know about the tunnel architecture.

#### Acceptance Criteria

1. THE Entry_Node SHALL route all customer VPN traffic (from VPN interfaces: tun0, ppp0, ipsec, wg0) through the active Tunnel_Interface to the Outbound_Node
2. THE Outbound_Node SHALL apply NAT_Masquerade to tunneled traffic so that internet-bound packets carry the Outbound_Node's public IP as source address
3. THE Entry_Node SHALL support forwarding traffic from all VPN protocols (OpenVPN, L2TP, IKEv2, SSH tunnel, WireGuard) through the same tunnel without protocol-specific configuration changes
4. THE customer VPN connection process SHALL remain unchanged — customers connect to the Entry_Node using standard VPN client configuration as if it were a direct server

### Requirement 5: Tunnel Health Monitoring

**User Story:** As an administrator, I want the system to monitor tunnel health between entry and outbound nodes, so that I can identify connectivity problems before customers are affected.

#### Acceptance Criteria

1. WHILE a Tunnel_Pair is active, THE Entry_Node SHALL perform Health_Checks every 30 seconds by verifying the WireGuard handshake timestamp on the Tunnel_Interface
2. THE Entry_Node SHALL report tunnel health status (healthy/degraded/down) in each status push to the Panel
3. WHEN the Entry_Node detects no WireGuard handshake on the Tunnel_Interface for more than 90 seconds, THE Entry_Node SHALL report tunnel status as "down"
4. THE Panel SHALL display tunnel health status for each active Tunnel_Pair in the Admin_UI
5. THE Panel SHALL record tunnel status history for diagnostic purposes

### Requirement 6: Outbound Node Failover

**User Story:** As an administrator, I want entry nodes to automatically switch to a backup outbound node when the primary outbound goes down, so that customer service continuity is maintained.

#### Acceptance Criteria

1. THE Panel SHALL support designating Tunnel_Pairs with a priority value (1 = primary, 2+ = backup) for each Entry_Node
2. WHEN the Panel detects that an Entry_Node's primary Outbound_Node is down (based on tunnel health reports), THE Panel SHALL dispatch a "tunnel.failover" task to the Entry_Node with the backup Outbound_Node's tunnel configuration
3. WHEN the Entry_Node receives a "tunnel.failover" task, THE Node_Agent SHALL reconfigure routing to forward customer traffic through the backup tunnel and report completion
4. WHEN the primary Outbound_Node recovers (tunnel health returns to "healthy"), THE Panel SHALL dispatch a "tunnel.failback" task to switch the Entry_Node back to the primary tunnel
5. THE Panel SHALL log all failover and failback events with timestamps, affected nodes, and trigger reason
6. IF no backup Tunnel_Pair is configured for an Entry_Node with a failed primary, THEN THE Panel SHALL send an alert notification to the administrator

### Requirement 7: Tunnel Traffic Statistics

**User Story:** As an administrator, I want to see traffic statistics for both entry and outbound nodes in tunnel mode, so that I can monitor bandwidth usage and plan capacity.

#### Acceptance Criteria

1. THE Entry_Node SHALL report tunnel TX bytes (traffic sent to outbound) and tunnel RX bytes (traffic received from outbound) in each status push
2. THE Outbound_Node SHALL report tunnel RX bytes (traffic received from entry) and tunnel TX bytes (traffic sent back to entry) in each status push
3. THE Panel SHALL store per-Tunnel_Pair traffic statistics with hourly granularity
4. THE Admin_UI SHALL display tunnel traffic charts showing entry TX, outbound RX, and total throughput per Tunnel_Pair
5. THE Panel SHALL aggregate tunnel traffic statistics per Entry_Node and per Outbound_Node for capacity planning views

### Requirement 8: Tunnel Removal

**User Story:** As an administrator, I want to remove tunnel pairs, so that I can decommission nodes or reconfigure the tunnel topology.

#### Acceptance Criteria

1. WHEN an administrator removes a Tunnel_Pair, THE Panel SHALL dispatch a "tunnel.teardown" task to both the Entry_Node and Outbound_Node
2. WHEN the Entry_Node receives a "tunnel.teardown" task, THE Node_Agent SHALL remove the Tunnel_Interface, remove associated routing rules, and remove iptables FORWARD rules
3. WHEN the Outbound_Node receives a "tunnel.teardown" task, THE Node_Agent SHALL remove the Tunnel_Interface and remove associated NAT_Masquerade rules
4. THE Panel SHALL set the Tunnel_Pair status to "removed" after both nodes confirm teardown
5. IF the Tunnel_Pair being removed is the active primary for an Entry_Node with a backup, THEN THE Panel SHALL trigger failover to the backup before removing the primary

### Requirement 9: Resource Constraints Compliance

**User Story:** As a system operator, I want tunnel mode to operate within the resource limits of a 1GB RAM single-core VPS, so that both entry and outbound nodes remain stable under load.

#### Acceptance Criteria

1. THE Tunnel_Interface SHALL use WireGuard (kernel-mode) for the inter-node tunnel to minimize CPU and memory overhead
2. THE Node_Agent SHALL not consume more than 30MB of additional RAM when managing tunnel operations
3. THE Entry_Node SHALL handle a minimum of 100 concurrent customer VPN connections routed through the tunnel on a node with 1GB RAM and a single CPU core
4. THE inter-node tunnel MTU SHALL be configured to avoid fragmentation (default 1400, accounting for WireGuard overhead on customer VPN packets)

### Requirement 10: Low Latency Priority for Inter-Node Tunnel

**User Story:** As a system operator, I want the inter-node tunnel traffic to be prioritized on the network, so that customer VPN experience is not degraded by other traffic on the same node.

#### Acceptance Criteria

1. WHEN tunnel provisioning is complete, THE Entry_Node SHALL apply tc (traffic control) qdisc rules to prioritize traffic on the Tunnel_Interface
2. THE Entry_Node SHALL mark tunnel traffic with DSCP EF (Expedited Forwarding) for network-level priority
3. THE Node_Agent SHALL use fq_codel qdisc on the Tunnel_Interface for fair queuing with low latency
4. WHEN a tunnel is removed, THE Node_Agent SHALL remove the associated tc qdisc rules

### Requirement 11: Database Schema for Tunnel Mode

**User Story:** As a developer, I want the database schema to support tunnel mode storage, so that all tunnel configuration and statistics are properly persisted and queryable.

#### Acceptance Criteria

1. THE Panel SHALL add a "tunnel_role" ENUM column (values: 'none', 'entry', 'outbound') to the nodes table via a numbered SQL migration
2. THE Panel SHALL create a tunnel_pairs table storing: id, entry_node_id, outbound_node_id, tunnel_subnet, entry_wg_public_key, entry_wg_private_key_encrypted, outbound_wg_public_key, outbound_wg_private_key_encrypted, status (pending/active/error/removed), priority, created_at, updated_at
3. THE Panel SHALL create a tunnel_stats table storing: id, tunnel_pair_id, entry_tx_bytes, entry_rx_bytes, outbound_tx_bytes, outbound_rx_bytes, recorded_at
4. THE Panel SHALL create a tunnel_events table storing: id, tunnel_pair_id, event_type (failover/failback/error/recovery), from_outbound_id, to_outbound_id, reason, created_at
5. THE Panel SHALL create indexes on tunnel_pairs(entry_node_id, status) and tunnel_pairs(outbound_node_id, status) for efficient lookups

### Requirement 12: Admin UI for Tunnel Management

**User Story:** As an administrator, I want a dedicated tunnel management interface, so that I can view, configure, and monitor all tunnel pairs from a single page.

#### Acceptance Criteria

1. THE Admin_UI SHALL provide a "Tunnel Mode" page listing all Tunnel_Pairs with columns: Entry_Node name, Outbound_Node name, tunnel subnet, status, priority, health, and last traffic stats
2. THE Admin_UI SHALL provide a "Create Tunnel Pair" dialog allowing selection of Entry_Node and Outbound_Node from eligible nodes
3. THE Admin_UI SHALL display real-time tunnel health indicators (green/yellow/red) based on the last reported Health_Check status
4. THE Admin_UI SHALL provide a "Failover History" section showing recent failover and failback events with timestamps and reasons
5. WHEN an administrator clicks a Tunnel_Pair row, THE Admin_UI SHALL show detailed tunnel information including both nodes' IPs, tunnel subnet, WireGuard public keys, creation date, and traffic charts

