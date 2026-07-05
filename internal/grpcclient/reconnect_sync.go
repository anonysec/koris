package grpcclient

import (
	"context"
	"log"

	"github.com/anonysec/koris/internal/dbstore"
	"github.com/anonysec/koris/internal/noderegistry"
)

// RegisterReconnectSync registers an OnStatusChange callback on the pool that
// triggers a full sync whenever a node transitions from "offline" to "online".
// This ensures that:
//  1. The Panel calls Health + AllCoreStatuses to learn the node's capabilities
//     (satisfying the requirement that these are fetched within 30s of reconnection).
//  2. Non-running cores are re-enabled with default configurations.
//  3. A full user sync (FullSyncForNode) pushes all assigned users to the node,
//     detecting any drift that occurred during the disconnect.
func RegisterReconnectSync(pool Pool, syncService *UserSyncService, store dbstore.Store, domain string) {
	pool.OnStatusChange(func(nodeID int64, old, new NodeStatus) {
		if old == StatusOffline && new == StatusOnline {
			log.Printf("[knode] Node %d reconnected (offline → online), triggering Health + ReenableCores + full user sync", nodeID)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[knode] RegisterReconnectSync: recovered panic for node %d: %v", nodeID, r)
					}
				}()

				ctx := context.Background()

				// 1. Call Health RPC to verify the node is responsive and get its status.
				if err := checkNodeHealth(ctx, pool, nodeID); err != nil {
					log.Printf("[knode] Health check failed for reconnected node %d: %v (continuing with sync)", nodeID, err)
				}

				// 2. Call AllCoreStatuses to refresh the node's capabilities in the DB.
				if err := RefreshNodeState(ctx, pool, store, nodeID); err != nil {
					log.Printf("[knode] RefreshNodeState failed for reconnected node %d: %v (continuing with re-enable)", nodeID, err)
				}

				// 3. Re-enable any stopped/crashed cores with default configurations.
				cm := NewCoreManager(pool, store)
				enabler := NewCoreEnablerAdapter(cm)
				results := noderegistry.ReenableStoppedCores(ctx, enabler, nodeID, domain)
				for _, r := range results {
					if !r.Success {
						log.Printf("[knode] ReenableStoppedCores: core %q failed on node %d: %s", r.Core, nodeID, r.Error)
					}
				}

				// 4. Full user sync for all cores on this node.
				if err := syncService.FullSyncForNode(ctx, nodeID); err != nil {
					log.Printf("[knode] Full sync failed for reconnected node %d: %v", nodeID, err)
				}
			}()
		}
	})
}

// checkNodeHealth performs a Health RPC on a single node to verify it is responsive.
// This is called during reconnection to confirm the node is reachable and to record
// its health status (HEALTHY, DEGRADED, or UNHEALTHY).
func checkNodeHealth(ctx context.Context, pool Pool, nodeID int64) error {
	node, err := pool.Get(nodeID)
	if err != nil {
		return err
	}
	if node.Conn == nil {
		return nil // No connection yet, skip
	}

	hc := &HealthChecker{pool: pool.(*connPool)}
	status, err := hc.callHealthRPC(ctx, nodeID)
	if err != nil {
		return err
	}

	log.Printf("[knode] Health check for reconnected node %d: %s", nodeID, status)
	return nil
}

// InitialNodeSync performs a Health check and AllCoreStatuses refresh for a node
// that has just connected successfully during Panel startup. This satisfies the
// requirement that Health + AllCoreStatuses are called within 30s of a node
// becoming reachable (Requirement 10.4).
func InitialNodeSync(ctx context.Context, pool Pool, store dbstore.Store, nodeID int64) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[knode] InitialNodeSync: recovered panic for node %d: %v", nodeID, r)
			}
		}()

		// Call Health RPC
		if err := checkNodeHealth(ctx, pool, nodeID); err != nil {
			log.Printf("[knode] Initial health check failed for node %d: %v", nodeID, err)
		}

		// Call AllCoreStatuses to populate node_services
		if err := RefreshNodeState(ctx, pool, store, nodeID); err != nil {
			log.Printf("[knode] Initial RefreshNodeState failed for node %d: %v", nodeID, err)
		}
	}()
}
