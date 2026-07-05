package grpcclient

import (
	"context"
	"github.com/anonysec/koris/internal/knodepb"
	"encoding/json"
	"fmt"
	"log"

	"github.com/anonysec/koris/internal/dbstore"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TunnelConfig holds the configuration for setting up an outbound tunnel.
type TunnelConfig struct {
	Protocol    string            `json:"protocol"`     // "vless_reality", "wireguard", "ssh", "rathole", etc.
	ExitAddress string            `json:"exit_address"` // exit node address (IP or hostname)
	ExitPort    int               `json:"exit_port"`    // exit node port
	ExtraConfig map[string]string `json:"extra_config"` // protocol-specific configuration
}

// TunnelInfo represents a single active tunnel reported by a knode instance.
type TunnelInfo struct {
	TunnelID string `json:"tunnel_id"`
	Protocol string `json:"protocol"`
	ExitAddr string `json:"exit_addr"`
	ExitPort int    `json:"exit_port"`
	State    string `json:"state"` // "active", "stopped", "error"
}

// TunnelManager provides outbound tunnel management operations on knode instances
// via gRPC. It wraps the connection pool and database store, exposing SetupTunnel,
// TeardownTunnel, and TunnelStatus as high-level methods for the admin API.
//
// Multi-panel compatibility (Requirements 14.1, 14.3):
// - TunnelStatus always fetches live state from knode (no caching).
// - TeardownTunnel handles NotFound gracefully (tunnel already torn down by another panel).
type TunnelManager struct {
	pool  Pool
	store dbstore.Store
}

// NewTunnelManager creates a new TunnelManager using the given connection pool and store.
func NewTunnelManager(pool Pool, store dbstore.Store) *TunnelManager {
	return &TunnelManager{pool: pool, store: store}
}

// SetupTunnel requests a knode to set up an outbound tunnel with the given configuration.
// On success, it stores the tunnel_id in the outbound_tunnels table for future reference.
//
// Satisfies Requirement 11.1: WHEN an admin requests setting up a tunnel, THE Panel SHALL
// call SetupTunnel on the target knode with the protocol, exit address, exit port, and
// protocol-specific configuration.
//
// Satisfies Requirement 11.4: WHEN the SetupTunnel RPC returns success, THE Panel SHALL
// store the tunnel_id in the database for future reference.
//
// Satisfies Requirement 11.5: IF the SetupTunnel RPC returns an error, THEN THE Panel
// SHALL display the error message to the admin.
func (tm *TunnelManager) SetupTunnel(ctx context.Context, nodeID int64, config TunnelConfig) (string, error) {
	node, err := tm.pool.Get(nodeID)
	if err != nil {
		return "", fmt.Errorf("node %d not found in pool: %w", nodeID, err)
	}

	if node.Status == StatusOffline {
		return "", fmt.Errorf("node %q is offline, cannot setup tunnel", node.NodeName)
	}

	// Map string protocol to proto enum
	var protoEnum knodepb.TunnelProtocol
	switch config.Protocol {
	case "wireguard":
		protoEnum = knodepb.TunnelProtocol_TUNNEL_PROTOCOL_WIREGUARD
	case "ssh":
		protoEnum = knodepb.TunnelProtocol_TUNNEL_PROTOCOL_SSH
	case "vless_reality":
		protoEnum = knodepb.TunnelProtocol_TUNNEL_PROTOCOL_VLESS_REALITY
	case "rathole":
		protoEnum = knodepb.TunnelProtocol_TUNNEL_PROTOCOL_RATHOLE
	case "gre":
		protoEnum = knodepb.TunnelProtocol_TUNNEL_PROTOCOL_GRE
	case "ipip":
		protoEnum = knodepb.TunnelProtocol_TUNNEL_PROTOCOL_IPIP
	default:
		protoEnum = knodepb.TunnelProtocol_TUNNEL_PROTOCOL_UNSPECIFIED
	}

	configBytes, _ := json.Marshal(config.ExtraConfig)

	client := knodepb.NewKnodeServiceClient(node.Conn)
	resp, rpcErr := client.SetupTunnel(ctx, &knodepb.SetupTunnelRequest{
		Protocol: protoEnum,
		ExitAddr: config.ExitAddress,
		ExitPort: int32(config.ExitPort),
		Config:   configBytes,
	})
	var tunnelID string
	if rpcErr == nil {
		tunnelID = resp.GetTunnelId()
	} else {
		log.Printf("[knode] SetupTunnel failed on node %q (id=%d): %v", node.NodeName, nodeID, rpcErr)
		return "", fmt.Errorf("SetupTunnel RPC on node %q: %w", node.NodeName, rpcErr)
	}

	// Store tunnel_id in outbound_tunnels table on successful setup
	if err := tm.storeTunnel(ctx, nodeID, tunnelID, config); err != nil {
		log.Printf("[knode] Failed to store tunnel record for node %d: %v", nodeID, err)
		return tunnelID, fmt.Errorf("tunnel setup succeeded but failed to store record: %w", err)
	}

	log.Printf("[knode] SetupTunnel: stored tunnel %q for node %d", tunnelID, nodeID)
	return tunnelID, nil
}

// TeardownTunnel requests a knode to tear down an active outbound tunnel.
// On success, it updates the tunnel state in the outbound_tunnels table to "stopped".
//
// Satisfies Requirement 11.2: WHEN an admin requests tearing down a tunnel, THE Panel
// SHALL call TeardownTunnel on the target knode with the tunnel ID.
//
// Satisfies Requirement 11.5: IF the TeardownTunnel RPC returns an error, THEN THE Panel
// SHALL display the error message to the admin.
//
// Multi-panel compatibility (Requirement 14.2): If NotFound is returned (tunnel already
// torn down by another panel), treat as success and update local state.
func (tm *TunnelManager) TeardownTunnel(ctx context.Context, nodeID int64, tunnelID string) error {
	node, err := tm.pool.Get(nodeID)
	if err != nil {
		return fmt.Errorf("node %d not found in pool: %w", nodeID, err)
	}

	if node.Status == StatusOffline {
		return fmt.Errorf("node %q is offline, cannot teardown tunnel", node.NodeName)
	}

	client := knodepb.NewKnodeServiceClient(node.Conn)
	_, rpcErr := client.TeardownTunnel(ctx, &knodepb.TeardownTunnelRequest{
		TunnelId: tunnelID,
	})

	if rpcErr != nil {
		if status.Code(rpcErr) == codes.NotFound {
			// Tunnel already torn down (likely by another panel instance).
			// Not an error — the desired state is already achieved.
			log.Printf("[knode] TeardownTunnel: tunnel %q not found on node %q (id=%d), already torn down", tunnelID, node.NodeName, nodeID)
		} else {
			log.Printf("[knode] TeardownTunnel failed on node %q (id=%d): %v", node.NodeName, nodeID, rpcErr)
			return rpcErr
		}
	} else {
		log.Printf("[knode] TeardownTunnel: torn down tunnel %q on node %q (id=%d)",
			tunnelID, node.NodeName, nodeID)
	}

	// Update tunnel state to "stopped" in the database
	if err := tm.updateTunnelState(ctx, nodeID, tunnelID, "stopped"); err != nil {
		log.Printf("[knode] Failed to update tunnel state for %q on node %d: %v", tunnelID, nodeID, err)
		return fmt.Errorf("tunnel teardown succeeded but failed to update record: %w", err)
	}

	log.Printf("[knode] TeardownTunnel: marked tunnel %q as stopped for node %d", tunnelID, nodeID)
	return nil
}

// TunnelStatus retrieves the current active tunnels from a knode instance.
// Called when the admin views a node's tunnel status page.
//
// Satisfies Requirement 11.3: THE Panel SHALL call TunnelStatus on a node to display
// active tunnels and their states.
//
// Multi-panel compatibility (Requirements 14.1, 14.3): This always fetches live state
// from knode rather than relying on any cached/local data, ensuring consistent display
// even when multiple panels modify the same node simultaneously.
func (tm *TunnelManager) TunnelStatus(ctx context.Context, nodeID int64) ([]TunnelInfo, error) {
	node, err := tm.pool.Get(nodeID)
	if err != nil {
		return nil, fmt.Errorf("node %d not found in pool: %w", nodeID, err)
	}

	if node.Status == StatusOffline {
		return nil, fmt.Errorf("node %q is offline, cannot query tunnel status", node.NodeName)
	}

	client := knodepb.NewKnodeServiceClient(node.Conn)
	resp, err := client.TunnelStatus(ctx, &knodepb.TunnelStatusRequest{})
	if err != nil {
		log.Printf("[knode] TunnelStatus failed on node %q (id=%d): %v", node.NodeName, nodeID, err)
		return nil, err
	}
	tunnels := make([]TunnelInfo, 0, len(resp.GetTunnels()))
	for _, t := range resp.GetTunnels() {
		if t == nil {
			continue
		}
		tunnels = append(tunnels, TunnelInfo{
			TunnelID: t.GetTunnelId(),
			Protocol: t.GetProtocol().String(),
			ExitAddr: t.GetExitAddr(),
			ExitPort: int(t.GetExitPort()),
			State:    t.GetState().String(),
		})
	}
	return tunnels, nil
}

// storeTunnel inserts a new tunnel record into the outbound_tunnels table.
func (tm *TunnelManager) storeTunnel(ctx context.Context, nodeID int64, tunnelID string, config TunnelConfig) error {
	db := tm.store.DB()
	_, err := db.ExecContext(ctx,
		`INSERT INTO outbound_tunnels (node_id, tunnel_id, protocol, exit_addr, exit_port, state, created_at)
		 VALUES ($1, $2, $3, $4, $5, 'active', NOW())`,
		nodeID, tunnelID, config.Protocol, config.ExitAddress, config.ExitPort,
	)
	return err
}

// updateTunnelState updates the state of an existing tunnel record in the database.
func (tm *TunnelManager) updateTunnelState(ctx context.Context, nodeID int64, tunnelID string, state string) error {
	db := tm.store.DB()
	_, err := db.ExecContext(ctx,
		`UPDATE outbound_tunnels SET state = $1 WHERE node_id = $2 AND tunnel_id = $3`,
		state, nodeID, tunnelID,
	)
	return err
}
