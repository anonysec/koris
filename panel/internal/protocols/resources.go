package protocols

import (
	"database/sql"
	"encoding/json"
	"log"
)

// ProtocolResourceStats holds per-protocol CPU/memory metrics parsed from the
// node push payload.
type ProtocolResourceStats struct {
	Protocol string  `json:"protocol"`
	CPUPct   float64 `json:"cpu_pct"`
	MemoryMB float64 `json:"memory_mb"`
	Status   string  `json:"status"`
}

// ParseProtocolResources extracts per-protocol resource usage from the node_status
// payload_json field. The node agent reports service-level metrics in the services
// map and optionally in a "protocol_resources" key within the push payload.
// If the agent hasn't reported protocol_resources, we fall back to service status only.
func ParseProtocolResources(db *sql.DB, nodeID int64) []ProtocolResourceStats {
	var payloadRaw []byte
	err := db.QueryRow(`SELECT payload_json FROM node_status WHERE node_id = ?`, nodeID).Scan(&payloadRaw)
	if err != nil || len(payloadRaw) == 0 {
		return nil
	}

	var payload struct {
		Services          map[string]string          `json:"services"`
		ProtocolResources map[string]json.RawMessage `json:"protocol_resources"`
	}
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		log.Printf("[protocols] failed to parse payload for node %d: %v", nodeID, err)
		return nil
	}

	var results []ProtocolResourceStats

	// If the node agent reports protocol_resources, use that
	if payload.ProtocolResources != nil {
		for proto, raw := range payload.ProtocolResources {
			var stats struct {
				CPUPct   float64 `json:"cpu_pct"`
				MemoryMB float64 `json:"memory_mb"`
			}
			if err := json.Unmarshal(raw, &stats); err != nil {
				continue
			}
			status := "unknown"
			if s, ok := payload.Services[proto]; ok {
				status = s
			}
			results = append(results, ProtocolResourceStats{
				Protocol: proto,
				CPUPct:   stats.CPUPct,
				MemoryMB: stats.MemoryMB,
				Status:   status,
			})
		}
		return results
	}

	// Fallback: just return service status (no CPU/memory data)
	for proto, status := range payload.Services {
		results = append(results, ProtocolResourceStats{
			Protocol: proto,
			CPUPct:   0,
			MemoryMB: 0,
			Status:   status,
		})
	}
	return results
}
