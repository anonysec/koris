package api

import (
	"encoding/json"
	"log"
	"net/http"
)

// handleNodeAgentUpdate handles POST /api/admin/nodes/update.
// Pushes an update_agent task to a single node via the task polling system.
func (s *Server) handleNodeAgentUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	limitBody(w, r, maxJSONBody)
	var in struct {
		NodeID   int64  `json:"node_id"`
		Version  string `json:"version"`
		URL      string `json:"url"`
		Checksum string `json:"checksum"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bad_json"})
		return
	}

	if in.NodeID == 0 || in.Version == "" || in.URL == "" || in.Checksum == "" {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "missing_fields"})
		return
	}

	// Build the task payload
	payload, _ := json.Marshal(map[string]any{
		"version":  in.Version,
		"url":      in.URL,
		"checksum": in.Checksum,
	})

	actor, _, _ := s.currentAdmin(r)

	_, err := s.DB.Exec(
		`INSERT INTO node_tasks(node_id, action, payload_json, status, created_by) VALUES(?, 'update_agent', ?, 'pending', ?)`,
		in.NodeID, string(payload), actor,
	)
	if err != nil {
		log.Printf("[update] failed to queue update_agent task for node %d: %v", in.NodeID, err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error"})
		return
	}

	log.Printf("[update] queued update_agent task for node %d to version %s by %s", in.NodeID, in.Version, actor)
	writeJSON(w, map[string]any{"ok": true, "message": "update task queued"})
}

// handleNodeBulkAgentUpdate handles POST /api/admin/nodes/update/bulk.
// Pushes update_agent tasks to multiple nodes with a 30-second staggered interval.
func (s *Server) handleNodeBulkAgentUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	limitBody(w, r, maxJSONBody)
	var in struct {
		NodeIDs  []int64 `json:"node_ids"`
		Version  string  `json:"version"`
		URL      string  `json:"url"`
		Checksum string  `json:"checksum"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bad_json"})
		return
	}

	if len(in.NodeIDs) == 0 || in.Version == "" || in.URL == "" || in.Checksum == "" {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "missing_fields"})
		return
	}

	actor, _, _ := s.currentAdmin(r)

	for i, nodeID := range in.NodeIDs {
		delaySeconds := i * 30

		payload, _ := json.Marshal(map[string]any{
			"version":       in.Version,
			"url":           in.URL,
			"checksum":      in.Checksum,
			"delay_seconds": delaySeconds,
		})

		_, err := s.DB.Exec(
			`INSERT INTO node_tasks(node_id, action, payload_json, status, created_by) VALUES(?, 'update_agent', ?, 'pending', ?)`,
			nodeID, string(payload), actor,
		)
		if err != nil {
			log.Printf("[update] failed to queue bulk update_agent task for node %d: %v", nodeID, err)
			// Continue with remaining nodes
			continue
		}
	}

	log.Printf("[update] queued bulk update_agent tasks for %d nodes to version %s by %s", len(in.NodeIDs), in.Version, actor)
	writeJSON(w, map[string]any{"ok": true, "queued": len(in.NodeIDs)})
}
