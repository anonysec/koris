# Requirements Document

## Introduction

Add WireGuard VPN protocol support to KorisPanel, enabling administrators to configure and manage WireGuard VPN servers on nodes, manage peers (clients), generate downloadable configuration files, and provide a gaming-optimized mode for low-latency connections. This feature integrates with the existing node agent task system, plan system, and customer portal.

## Glossary

- **Panel**: The central KorisPanel web application that manages nodes, customers, and VPN services
- **Node_Agent**: The lightweight agent process deployed on each VPN server node that executes tasks and reports status
- **WireGuard_Server**: The WireGuard VPN service instance running on a node (wg-quick@wg0)
- **Peer**: A WireGuard client configuration representing a single VPN connection endpoint with its own key pair
- **Admin_UI**: The Vue 3 admin dashboard SPA for managing all panel resources
- **Customer_Portal**: The Vue 3 customer-facing SPA for self-service VPN profile downloads and account management
- **Node_VPN_Config**: A record in the node_vpn_configs table storing per-node protocol settings
- **Gaming_Optimize**: A configuration mode that tunes WireGuard parameters for low-latency, high-throughput gaming traffic
- **Task_System**: The existing mechanism where the Panel dispatches tasks to Node_Agent via /api/node/tasks/poll
- **Config_File**: A downloadable .conf file containing all WireGuard client connection parameters
- **Key_Pair**: A Curve25519 private/public key pair used for WireGuard cryptographic identity
- **Preshared_Key**: An additional symmetric key shared between server and peer for post-quantum resistance

## Requirements

### Requirement 1: WireGuard Server Setup via Node Agent

**User Story:** As an administrator, I want to enable and configure a WireGuard VPN server on a node, so that the node can accept WireGuard client connections.

#### Acceptance Criteria

1. WHEN an administrator enables WireGuard on a node, THE Panel SHALL dispatch a "wireguard.setup" task to the Node_Agent containing the server configuration parameters (port, network CIDR, DNS servers, MTU)
2. WHEN the Node_Agent receives a "wireguard.setup" task, THE Node_Agent SHALL generate a server Key_Pair, create the /etc/wireguard/wg0.conf file, and start the wg-quick@wg0 service
3. WHEN the Node_Agent completes WireGuard setup, THE Node_Agent SHALL report the server public key back to the Panel via the task completion response
4. THE Panel SHALL store the WireGuard server configuration in the Node_VPN_Config table with protocol value "wireguard"
5. IF the Node_Agent fails to set up WireGuard, THEN THE Node_Agent SHALL report the failure reason in the task completion response with status "failed"
6. THE Panel SHALL validate that the configured port is between 1024 and 65535 before dispatching the setup task
7. THE Panel SHALL validate that the network CIDR is a valid IPv4 or IPv6 subnet before dispatching the setup task

### Requirement 2: WireGuard Server Configuration Management

**User Story:** As an administrator, I want to configure per-node WireGuard settings (port, network CIDR, DNS, MTU), so that each node can be tuned for its specific deployment environment.

#### Acceptance Criteria

1. THE Admin_UI SHALL provide a WireGuard configuration form for each node with fields for listen port, network CIDR, primary DNS, secondary DNS, and MTU
2. WHEN an administrator updates WireGuard settings on a node, THE Panel SHALL dispatch a "wireguard.update_config" task to the Node_Agent with the updated parameters
3. WHEN the Node_Agent receives a "wireguard.update_config" task, THE Node_Agent SHALL update the /etc/wireguard/wg0.conf [Interface] section and reload the WireGuard interface
4. THE Panel SHALL use default values of port 51820, network 10.66.66.0/24, DNS 1.1.1.1, and MTU 1420 when no custom values are specified
5. THE Admin_UI SHALL allow administrators to enable or disable WireGuard on a node without removing existing peer configurations

### Requirement 3: Peer Creation

**User Story:** As an administrator, I want to create WireGuard peers for customers, so that customers can connect to the VPN using WireGuard.

#### Acceptance Criteria

1. WHEN an administrator creates a new Peer, THE Panel SHALL generate a client Key_Pair and Preshared_Key using cryptographically secure random generation
2. WHEN a Peer is created, THE Panel SHALL assign the next available IP address from the node's configured network CIDR
3. WHEN a Peer is created, THE Panel SHALL dispatch a "wireguard.add_peer" task to the Node_Agent containing the peer public key, preshared key, and allowed IPs
4. THE Panel SHALL store the peer record with the encrypted private key, public key, preshared key, assigned IP, associated customer ID, and node ID
5. IF the assigned IP address conflicts with an existing active Peer on the same node, THEN THE Panel SHALL reject the creation and return an error
6. THE Panel SHALL support creating peers both with and without a customer association (unassigned peers)

### Requirement 4: Peer Removal

**User Story:** As an administrator, I want to remove WireGuard peers, so that revoked users can no longer connect.

#### Acceptance Criteria

1. WHEN an administrator removes a Peer, THE Panel SHALL set the peer status to "revoked" in the database
2. WHEN a Peer is removed, THE Panel SHALL dispatch a "wireguard.remove_peer" task to the Node_Agent containing the peer public key
3. WHEN the Node_Agent receives a "wireguard.remove_peer" task, THE Node_Agent SHALL remove the peer from the live WireGuard interface and from the wg0.conf file
4. THE Panel SHALL release the peer's assigned IP address back to the available pool after revocation

### Requirement 5: Peer Listing and Status

**User Story:** As an administrator, I want to view all WireGuard peers with their connection status, so that I can monitor active connections.

#### Acceptance Criteria

1. THE Admin_UI SHALL display a list of all WireGuard peers with columns for ID, customer username, node, public key, allowed IPs, status, last handshake time, and transferred bytes (RX/TX)
2. THE Admin_UI SHALL support filtering peers by node, status (active/revoked), and customer
3. WHEN the Node_Agent reports status, THE Node_Agent SHALL include WireGuard peer handshake times and transfer statistics from the "wg show" command output
4. THE Panel SHALL update peer last_handshake_at, rx_bytes, and tx_bytes fields when receiving status data from the Node_Agent

### Requirement 6: Client Configuration Generation

**User Story:** As an administrator or customer, I want to download a WireGuard .conf file for a peer, so that the client can import it directly into a WireGuard application.

#### Acceptance Criteria

1. WHEN a Config_File is requested, THE Panel SHALL generate a valid WireGuard client configuration containing the client private key, assigned address, DNS servers, server public key, preshared key, server endpoint, and allowed IPs (0.0.0.0/0, ::/0)
2. THE Panel SHALL serve the Config_File with Content-Type "text/plain" and Content-Disposition header for download with filename format "wg-peer-{id}.conf"
3. THE Panel SHALL include PersistentKeepalive = 25 in the generated Config_File
4. IF the peer's private key is not available, THEN THE Panel SHALL return an error indicating the configuration cannot be generated
5. FOR ALL valid Peer records with stored private keys, generating a Config_File then parsing the result SHALL produce a valid WireGuard configuration containing the correct keys and addresses (round-trip property)

### Requirement 7: Gaming Optimize Mode

**User Story:** As an administrator, I want to enable a gaming optimization mode on WireGuard nodes, so that gaming traffic benefits from lower latency and higher throughput.

#### Acceptance Criteria

1. THE Admin_UI SHALL provide a "Gaming Optimize" toggle per node within the WireGuard configuration section
2. WHEN Gaming_Optimize is enabled, THE Panel SHALL configure MTU to 1280 to reduce fragmentation overhead
3. WHEN Gaming_Optimize is enabled, THE Panel SHALL set PersistentKeepalive to 15 seconds for faster dead-peer detection
4. WHEN Gaming_Optimize is enabled, THE Node_Agent SHALL apply fwmark-based priority routing rules using iptables/ip rule to prioritize WireGuard traffic
5. WHEN Gaming_Optimize is disabled, THE Node_Agent SHALL remove the priority routing rules and revert MTU and keepalive to standard values
6. THE Panel SHALL store the Gaming_Optimize setting in the Node_VPN_Config extra_json field

### Requirement 8: Customer Portal WireGuard Config Download

**User Story:** As a customer, I want to download my WireGuard configuration files from the portal, so that I can set up the VPN client on my devices.

#### Acceptance Criteria

1. THE Customer_Portal SHALL display a list of WireGuard peers assigned to the authenticated customer with their status and node information
2. WHEN a customer requests a Config_File download, THE Customer_Portal SHALL serve the configuration file for peers owned by that customer
3. IF a customer attempts to download a Config_File for a peer not assigned to the customer, THEN THE Panel SHALL return a 403 forbidden response
4. THE Customer_Portal SHALL display a QR code representation of the WireGuard configuration for easy mobile device setup

### Requirement 9: Plan System Integration

**User Story:** As an administrator, I want to add WireGuard as a protocol option in plans, so that customer subscriptions can include WireGuard access.

#### Acceptance Criteria

1. THE Panel SHALL add "wireguard" to the protocol ENUM in the node_vpn_configs table
2. THE Admin_UI SHALL include "WireGuard" as a selectable protocol when configuring plans and node VPN settings
3. WHEN a plan includes WireGuard protocol, THE Panel SHALL automatically provision a WireGuard Peer when a customer subscription is activated on a WireGuard-enabled node
4. WHEN a customer subscription is terminated, THE Panel SHALL revoke the associated WireGuard Peer

### Requirement 10: Connection Status Reporting

**User Story:** As an administrator, I want to see real-time WireGuard connection status on nodes, so that I can monitor service health.

#### Acceptance Criteria

1. THE Node_Agent SHALL report the WireGuard service status (running/stopped/error) in each status push to the Panel
2. THE Node_Agent SHALL report the number of active WireGuard peers (peers with handshake within the last 3 minutes) in the status push
3. THE Panel SHALL display the WireGuard service status on the node detail page in the Admin_UI
4. IF the WireGuard service is not running on a node where it is configured, THEN THE Panel SHALL display a warning indicator on the node list

### Requirement 11: IPv4 and IPv6 Dual-Stack Support

**User Story:** As an administrator, I want WireGuard to support both IPv4 and IPv6 tunnel addresses, so that clients can route both address families through the VPN.

#### Acceptance Criteria

1. THE Panel SHALL accept both IPv4 and IPv6 CIDR notation for the WireGuard network configuration
2. WHEN both IPv4 and IPv6 addresses are configured, THE Node_Agent SHALL assign dual-stack addresses to the WireGuard interface
3. THE Panel SHALL generate Config_Files with both IPv4 and IPv6 addresses in the Address field when dual-stack is configured
4. THE Panel SHALL validate IPv6 CIDR notation using standard net.ParseCIDR validation

### Requirement 12: Resource Constraints Compliance

**User Story:** As a system operator, I want WireGuard to operate within the resource limits of a 1GB RAM single-core VPS, so that the node remains stable under load.

#### Acceptance Criteria

1. THE WireGuard_Server SHALL support a minimum of 50 concurrent peers on a node with 1GB RAM and a single CPU core
2. THE Node_Agent SHALL not consume more than 50MB of additional RAM when managing WireGuard peer operations
3. WHEN the Node_Agent adds or removes peers, THE Node_Agent SHALL use the "wg" command-line tool rather than spawning additional long-running processes
4. THE Node_Agent SHALL use "wg syncconf" for bulk configuration updates to minimize service interruption

### Requirement 13: Database Schema Extension

**User Story:** As a developer, I want the database schema to support WireGuard protocol storage, so that all WireGuard data is properly persisted and queryable.

#### Acceptance Criteria

1. THE Panel SHALL add "wireguard" to the protocol ENUM column in the node_vpn_configs table via a numbered SQL migration
2. THE Panel SHALL create a wg_peers table storing peer ID, customer_id, node_id, public_key, preshared_key, private_key_encrypted, allowed_ips, endpoint, status, last_handshake_at, rx_bytes, tx_bytes, created_at, and updated_at
3. THE Panel SHALL create an index on wg_peers(node_id, status) for efficient per-node active peer lookups
4. THE Panel SHALL create an index on wg_peers(customer_id) for efficient per-customer peer lookups
