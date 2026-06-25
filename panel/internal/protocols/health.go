package protocols

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"time"
)

// HealthResult holds the result of a protocol health check for a single node+protocol.
type HealthResult struct {
	NodeID   int64
	NodeIP   string
	Protocol string
	Port     int
	Healthy  bool
	Latency  time.Duration
	Error    string
}

// CheckProtocolHealth performs TCP connect tests for each enabled protocol on each node.
// It queries the node_vpn_configs table for enabled protocols and attempts a TCP dial
// to the node's IP on the configured port. Results are written to the node_services table.
// This function is intended to be called from the background worker tick.
func CheckProtocolHealth(db *sql.DB) {
	type target struct {
		NodeID   int64
		NodeIP   string
		Protocol string
		Port     int
	}

	rows, err := db.Query(`
		SELECT nvc.node_id, n.public_ip, nvc.protocol, nvc.port
		FROM node_vpn_configs nvc
		JOIN nodes n ON n.id = nvc.node_id
		WHERE nvc.enabled = TRUE AND n.status IN ('online', 'stale')
		  AND nvc.port > 0`)
	if err != nil {
		log.Printf("[protocols] health check query failed: %v", err)
		return
	}
	defer rows.Close()

	var targets []target
	for rows.Next() {
		var t target
		if err := rows.Scan(&t.NodeID, &t.NodeIP, &t.Protocol, &t.Port); err != nil {
			log.Printf("[protocols] health check scan error: %v", err)
			continue
		}
		targets = append(targets, t)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[protocols] health check rows error: %v", err)
		return
	}

	for _, t := range targets {
		addr := net.JoinHostPort(t.NodeIP, fmt.Sprintf("%d", t.Port))
		start := time.Now()
		conn, err := net.DialTimeout("tcp", addr, 5*time.Second)

		status := "running"
		if err != nil {
			status = "stopped"
			log.Printf("[protocols] health check failed for %s on node %d (%s): %v", t.Protocol, t.NodeID, addr, err)
		} else {
			conn.Close()
			_ = time.Since(start)
		}

		// Upsert into node_services table (service name matches protocol name)
		_, _ = db.Exec(`
			INSERT INTO node_services (node_id, service, status, updated_at)
			VALUES ($1, $2, $3, NOW())
			ON CONFLICT (node_id, service) DO UPDATE SET status = EXCLUDED.status, updated_at = NOW()`,
			t.NodeID, t.Protocol, status)
	}
}
