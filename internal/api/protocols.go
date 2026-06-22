package api

import (
	"encoding/json"
	"net/http"
)

// protocols handles GET (list protocol status per node) and POST (toggle protocol on a node).
func (s *Server) protocols(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listProtocols(w, r)
	case http.MethodPost:
		s.toggleProtocol(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, M{"ok": false, "error": "method"})
	}
}

func (s *Server) listProtocols(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.Query(`SELECT id, name, openvpn_enabled, l2tp_enabled FROM nodes ORDER BY id`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"ok": false, "error": "db_error"})
		return
	}
	defer rows.Close()

	type NodeProtocol struct {
		NodeID  int64  `json:"node_id"`
		Name    string `json:"name"`
		OpenVPN bool   `json:"openvpn"`
		L2TP    bool   `json:"l2tp"`
	}

	nodes := []NodeProtocol{}
	for rows.Next() {
		var n NodeProtocol
		if rows.Scan(&n.NodeID, &n.Name, &n.OpenVPN, &n.L2TP) == nil {
			nodes = append(nodes, n)
		}
	}

	ok(w, M{"ok": true, "protocols": M{"available": []string{"openvpn", "l2tp"}, "nodes": nodes}})
}

func (s *Server) toggleProtocol(w http.ResponseWriter, r *http.Request) {
	limitBody(r)
	var in struct {
		NodeID   int64  `json:"node_id"`
		Protocol string `json:"protocol"`
		Enabled  bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, M{"ok": false, "error": "bad_json"})
		return
	}
	if in.NodeID <= 0 {
		writeJSON(w, http.StatusBadRequest, M{"ok": false, "error": "node_id_required"})
		return
	}

	var column string
	switch in.Protocol {
	case "openvpn":
		column = "openvpn_enabled"
	case "l2tp":
		column = "l2tp_enabled"
	default:
		writeJSON(w, http.StatusBadRequest, M{"ok": false, "error": "invalid_protocol"})
		return
	}

	_, err := s.DB.Exec(`UPDATE nodes SET `+column+`=? WHERE id=?`, in.Enabled, in.NodeID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"ok": false, "error": "db_error"})
		return
	}

	// Create a task for the node agent to start/stop the service
	action := "start_" + in.Protocol
	if !in.Enabled {
		action = "stop_" + in.Protocol
	}
	s.DB.Exec(`INSERT INTO node_tasks (node_id, action, status, created_by) VALUES (?, ?, 'pending', ?)`,
		in.NodeID, action, s.currentAdmin(r))

	ok(w, M{"ok": true, "protocol": in.Protocol, "enabled": in.Enabled})
}
