package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"KorisLite/internal/auth"
)

func (s *Server) nodes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listNodes(w, r)
	case http.MethodPost:
		s.createNode(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, M{"ok": false, "error": "method"})
	}
}

func (s *Server) listNodes(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.Query(`SELECT id, name, public_ip, status, COALESCE(last_seen_at, created_at), openvpn_enabled, l2tp_enabled FROM nodes ORDER BY id`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"ok": false, "error": "db_error"})
		return
	}
	defer rows.Close()

	type Node struct {
		ID             int64  `json:"id"`
		Name           string `json:"name"`
		IP             string `json:"ip"`
		Status         string `json:"status"`
		LastSeen       string `json:"last_seen"`
		OpenVPNEnabled bool   `json:"openvpn_enabled"`
		L2TPEnabled    bool   `json:"l2tp_enabled"`
	}

	nodes := []Node{}
	for rows.Next() {
		var n Node
		var t time.Time
		if rows.Scan(&n.ID, &n.Name, &n.IP, &n.Status, &t, &n.OpenVPNEnabled, &n.L2TPEnabled) == nil {
			n.LastSeen = t.Format(time.RFC3339)
			nodes = append(nodes, n)
		}
	}

	ok(w, M{"ok": true, "nodes": nodes})
}

func (s *Server) createNode(w http.ResponseWriter, r *http.Request) {
	limitBody(r)
	var in struct {
		Name     string `json:"name"`
		PublicIP string `json:"public_ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, M{"ok": false, "error": "bad_json"})
		return
	}
	if in.Name == "" || in.PublicIP == "" {
		writeJSON(w, http.StatusBadRequest, M{"ok": false, "error": "name_and_ip_required"})
		return
	}

	// Generate API token for node agent
	token := auth.GenerateToken()
	tokenHash := auth.HashToken(token)

	res, err := s.DB.Exec(`INSERT INTO nodes (name, public_ip, api_token_hash, status, openvpn_enabled, l2tp_enabled, created_at)
		VALUES (?, ?, ?, 'offline', 1, 1, NOW())`,
		in.Name, in.PublicIP, tokenHash)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"ok": false, "error": "db_error"})
		return
	}

	id, _ := res.LastInsertId()
	ok(w, M{"ok": true, "id": id, "token": token, "name": in.Name})
}

func (s *Server) nodeByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/admin/nodes/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, M{"ok": false, "error": "invalid_id"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getNode(w, id)
	case http.MethodPut:
		s.updateNode(w, r, id)
	case http.MethodDelete:
		s.deleteNode(w, id)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, M{"ok": false, "error": "method"})
	}
}

func (s *Server) getNode(w http.ResponseWriter, id int64) {
	var name, ip, status string
	var lastSeen time.Time
	var ovpn, l2tp bool
	err := s.DB.QueryRow(`SELECT name, public_ip, status, COALESCE(last_seen_at, created_at), openvpn_enabled, l2tp_enabled FROM nodes WHERE id=?`, id).
		Scan(&name, &ip, &status, &lastSeen, &ovpn, &l2tp)
	if err != nil {
		writeJSON(w, http.StatusNotFound, M{"ok": false, "error": "not_found"})
		return
	}

	ok(w, M{"ok": true, "node": M{
		"id": id, "name": name, "ip": ip, "status": status,
		"last_seen":       lastSeen.Format(time.RFC3339),
		"openvpn_enabled": ovpn, "l2tp_enabled": l2tp,
	}})
}

func (s *Server) updateNode(w http.ResponseWriter, r *http.Request, id int64) {
	limitBody(r)
	var in struct {
		Name           *string `json:"name"`
		PublicIP       *string `json:"public_ip"`
		OpenVPNEnabled *bool   `json:"openvpn_enabled"`
		L2TPEnabled    *bool   `json:"l2tp_enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, M{"ok": false, "error": "bad_json"})
		return
	}

	if in.Name != nil {
		s.DB.Exec(`UPDATE nodes SET name=? WHERE id=?`, *in.Name, id)
	}
	if in.PublicIP != nil {
		s.DB.Exec(`UPDATE nodes SET public_ip=? WHERE id=?`, *in.PublicIP, id)
	}
	if in.OpenVPNEnabled != nil {
		s.DB.Exec(`UPDATE nodes SET openvpn_enabled=? WHERE id=?`, *in.OpenVPNEnabled, id)
	}
	if in.L2TPEnabled != nil {
		s.DB.Exec(`UPDATE nodes SET l2tp_enabled=? WHERE id=?`, *in.L2TPEnabled, id)
	}

	ok(w, M{"ok": true})
}

func (s *Server) deleteNode(w http.ResponseWriter, id int64) {
	s.DB.Exec(`DELETE FROM nodes WHERE id=?`, id)
	ok(w, M{"ok": true})
}

// ── Node Agent Endpoints ──

func (s *Server) nodePush(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, M{"ok": false, "error": "method"})
		return
	}

	// Authenticate by bearer token
	tokenHeader := r.Header.Get("Authorization")
	if len(tokenHeader) < 8 || tokenHeader[:7] != "Bearer " {
		writeJSON(w, http.StatusUnauthorized, M{"ok": false, "error": "bad_token"})
		return
	}
	token := tokenHeader[7:]
	tokenHash := auth.HashToken(token)

	var nodeID int64
	err := s.DB.QueryRow(`SELECT id FROM nodes WHERE api_token_hash=? LIMIT 1`, tokenHash).Scan(&nodeID)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, M{"ok": false, "error": "invalid_token"})
		return
	}

	// Parse push body (simplified — just update last_seen and status)
	limitBody(r)
	var push struct {
		CPUPercent  float64 `json:"cpu_percent"`
		RAMPercent  float64 `json:"ram_percent"`
		DiskPercent float64 `json:"disk_percent"`
		RxBps       int64   `json:"rx_bps"`
		TxBps       int64   `json:"tx_bps"`
		OnlineUsers int     `json:"online_users"`
	}
	json.NewDecoder(r.Body).Decode(&push)

	s.DB.Exec(`UPDATE nodes SET status='online', last_seen_at=NOW() WHERE id=?`, nodeID)

	ok(w, M{"ok": true})
}

func (s *Server) nodeTaskPoll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, M{"ok": false, "error": "method"})
		return
	}

	tokenHeader := r.Header.Get("Authorization")
	if len(tokenHeader) < 8 || tokenHeader[:7] != "Bearer " {
		writeJSON(w, http.StatusUnauthorized, M{"ok": false, "error": "bad_token"})
		return
	}
	token := tokenHeader[7:]
	tokenHash := auth.HashToken(token)

	var nodeID int64
	err := s.DB.QueryRow(`SELECT id FROM nodes WHERE api_token_hash=? LIMIT 1`, tokenHash).Scan(&nodeID)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, M{"ok": false, "error": "invalid_token"})
		return
	}

	// Fetch pending tasks
	rows, err := s.DB.Query(`SELECT id, action, COALESCE(payload_json,'{}') FROM node_tasks WHERE node_id=? AND status='pending' ORDER BY id LIMIT 5`, nodeID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"ok": false, "error": "db_error"})
		return
	}
	defer rows.Close()

	type Task struct {
		ID      int64  `json:"id"`
		Action  string `json:"action"`
		Payload string `json:"payload"`
	}
	tasks := []Task{}
	for rows.Next() {
		var t Task
		if rows.Scan(&t.ID, &t.Action, &t.Payload) == nil {
			tasks = append(tasks, t)
			s.DB.Exec(`UPDATE node_tasks SET status='running' WHERE id=?`, t.ID)
		}
	}

	ok(w, M{"ok": true, "tasks": tasks})
}
