package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// NodeBulkRequest represents a request to perform bulk operations on nodes.
type NodeBulkRequest struct {
	Action  string         `json:"action"`
	NodeIDs []int64        `json:"node_ids"`
	Params  map[string]any `json:"params"`
}

// NodeBulkResult represents the result for a single node in a bulk operation.
type NodeBulkResult struct {
	NodeID  int64  `json:"node_id"`
	Success bool   `json:"success"`
	TaskID  *int64 `json:"task_id,omitempty"`
	Error   string `json:"error,omitempty"`
}

// validNodeBulkActions defines the set of supported bulk actions.
var validNodeBulkActions = map[string]bool{
	"restart_openvpn":  true,
	"restart_all":      true,
	"push_config":      true,
	"enable_protocol":  true,
	"disable_protocol": true,
	"run_command":      true,
	"maintenance_on":   true,
	"maintenance_off":  true,
}

// nodeBulk handles POST /api/admin/nodes/bulk
func (s *Server) nodeBulk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	limitBody(w, r, maxJSONBody)
	var req NodeBulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bad_json"})
		return
	}

	// Validate action
	if !validNodeBulkActions[req.Action] {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_action"})
		return
	}

	// Validate node_ids
	if len(req.NodeIDs) == 0 {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "node_ids_required"})
		return
	}
	if len(req.NodeIDs) > 50 {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "too_many_nodes"})
		return
	}

	// Validate params for actions that require them
	if req.Action == "enable_protocol" || req.Action == "disable_protocol" {
		proto, _ := req.Params["protocol"].(string)
		if proto == "" {
			writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "protocol_required"})
			return
		}
	}
	if req.Action == "run_command" {
		cmd, _ := req.Params["command"].(string)
		if cmd == "" {
			writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "command_required"})
			return
		}
	}

	actor, _, _ := s.currentAdmin(r)
	ip := clientIP(r)

	results := make([]NodeBulkResult, 0, len(req.NodeIDs))

	for _, nodeID := range req.NodeIDs {
		result := NodeBulkResult{NodeID: nodeID}

		// Check node exists
		var exists int
		err := s.DB.QueryRow(`SELECT 1 FROM nodes WHERE id=? LIMIT 1`, nodeID).Scan(&exists)
		if err != nil {
			result.Error = "node not found"
			results = append(results, result)
			continue
		}

		switch req.Action {
		case "maintenance_on":
			err = s.nodeBulkSetMaintenance(nodeID, true)
		case "maintenance_off":
			err = s.nodeBulkSetMaintenance(nodeID, false)
		default:
			var taskID int64
			taskID, err = s.nodeBulkCreateTask(nodeID, req.Action, req.Params, actor)
			if err == nil {
				result.TaskID = &taskID
			}
		}

		if err != nil {
			result.Error = err.Error()
		} else {
			result.Success = true
		}
		results = append(results, result)
	}

	// Log audit trail
	s.logAudit(actor, "nodes.bulk_action", "node", "", nil, map[string]any{
		"action":   req.Action,
		"node_ids": req.NodeIDs,
		"params":   req.Params,
	}, ip)

	log.Printf("[nodes] bulk action=%s nodes=%d by=%s", req.Action, len(req.NodeIDs), actor)

	writeJSON(w, map[string]any{
		"ok":      true,
		"results": results,
	})
}

// nodeBulkCreateTask inserts a node_task for the given action and returns the task ID.
func (s *Server) nodeBulkCreateTask(nodeID int64, action string, params map[string]any, actor string) (int64, error) {
	taskAction, payload, err := s.nodeBulkResolveTask(action, params)
	if err != nil {
		return 0, err
	}

	payloadJSON, _ := json.Marshal(payload)

	res, err := s.DB.Exec(
		`INSERT INTO node_tasks(node_id, action, payload_json, status, created_by) VALUES(?, ?, ?, 'pending', ?)`,
		nodeID, taskAction, string(payloadJSON), actor,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create task: %v", err)
	}

	taskID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get task id: %v", err)
	}
	return taskID, nil
}

// nodeBulkResolveTask maps a bulk action to the node_tasks action name and payload.
func (s *Server) nodeBulkResolveTask(action string, params map[string]any) (string, map[string]any, error) {
	switch action {
	case "restart_openvpn":
		return "restart", map[string]any{"service": "openvpn"}, nil
	case "restart_all":
		return "restart_all", map[string]any{}, nil
	case "push_config":
		return "sync_config", map[string]any{}, nil
	case "enable_protocol":
		proto, _ := params["protocol"].(string)
		return "enable_protocol", map[string]any{"protocol": proto}, nil
	case "disable_protocol":
		proto, _ := params["protocol"].(string)
		return "disable_protocol", map[string]any{"protocol": proto}, nil
	case "run_command":
		cmd, _ := params["command"].(string)
		return "run_command", map[string]any{"command": cmd}, nil
	default:
		return "", nil, fmt.Errorf("unsupported task action: %s", action)
	}
}

// nodeBulkSetMaintenance updates the maintenance_mode flag on a node.
func (s *Server) nodeBulkSetMaintenance(nodeID int64, enabled bool) error {
	_, err := s.DB.Exec(`UPDATE nodes SET maintenance_mode=? WHERE id=?`, enabled, nodeID)
	if err != nil {
		return fmt.Errorf("failed to update maintenance mode: %v", err)
	}
	return nil
}
