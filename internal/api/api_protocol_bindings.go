package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// validProtocols is the set of allowed protocol values for protocol bindings.
var validProtocols = map[string]bool{
	"openvpn-udp": true,
	"openvpn-tcp": true,
	"l2tp":        true,
	"ikev2":       true,
	"wireguard":   true,
	"ssh":         true,
	"mtproto":     true,
}

// protocolBindings dispatches /api/admin/nodes/{nodeId}/bindings requests.
func (s *Server) protocolBindings(w http.ResponseWriter, r *http.Request) {
	// Extract nodeId from path: /api/admin/nodes/{nodeId}/bindings[/{id}]
	rest := strings.TrimPrefix(r.URL.Path, "/api/admin/nodes/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) < 2 || parts[1] != "bindings" {
		writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found"})
		return
	}

	nodeID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || nodeID <= 0 {
		writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found"})
		return
	}

	// /api/admin/nodes/{nodeId}/bindings/reorder
	if len(parts) == 3 && parts[2] == "reorder" {
		if r.Method == http.MethodPatch {
			s.reorderProtocolBindings(w, r, nodeID)
			return
		}
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	// /api/admin/nodes/{nodeId}/bindings/{id}
	if len(parts) == 3 {
		bindingID, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil || bindingID <= 0 {
			writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found"})
			return
		}
		if r.Method == http.MethodDelete {
			s.deleteProtocolBinding(w, r, nodeID, bindingID)
			return
		}
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	// /api/admin/nodes/{nodeId}/bindings
	if len(parts) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.listProtocolBindings(w, r, nodeID)
		case http.MethodPost:
			s.createProtocolBinding(w, r, nodeID)
		default:
			http.Error(w, "method", http.StatusMethodNotAllowed)
		}
		return
	}

	writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found"})
}

// listProtocolBindings handles GET /api/admin/nodes/{nodeId}/bindings.
// Returns all bindings for a node grouped by protocol with domain details.
func (s *Server) listProtocolBindings(w http.ResponseWriter, r *http.Request, nodeID int64) {
	rows, err := s.DB.QueryContext(r.Context(), `
		SELECT pb.id, pb.node_id, pb.protocol, pb.domain_id, pb.position, pb.created_at,
		       d.name, d.ip_address, d.status
		FROM vpn_protocol_bindings pb
		JOIN vpn_domains d ON d.id = pb.domain_id
		WHERE pb.node_id = $1
		ORDER BY pb.protocol, pb.position
	`, nodeID)
	if err != nil {
		log.Printf("[protocol-bindings] list query error for node %d: %v", nodeID, err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "Failed to list bindings"})
		return
	}
	defer rows.Close()

	type bindingResponse struct {
		ID           int64  `json:"id"`
		NodeID       int64  `json:"node_id"`
		Protocol     string `json:"protocol"`
		DomainID     int64  `json:"domain_id"`
		Position     int    `json:"position"`
		CreatedAt    string `json:"created_at"`
		DomainName   string `json:"domain_name"`
		DomainIP     string `json:"domain_ip"`
		DomainStatus string `json:"domain_status"`
		Warning      bool   `json:"warning"`
	}

	var bindings []bindingResponse
	for rows.Next() {
		var b bindingResponse
		var createdAt sql.NullTime
		if err := rows.Scan(&b.ID, &b.NodeID, &b.Protocol, &b.DomainID, &b.Position, &createdAt,
			&b.DomainName, &b.DomainIP, &b.DomainStatus); err != nil {
			log.Printf("[protocol-bindings] scan error: %v", err)
			writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "Failed to scan bindings"})
			return
		}
		if createdAt.Valid {
			b.CreatedAt = createdAt.Time.Format("2006-01-02T15:04:05Z07:00")
		}
		if b.DomainStatus == "blocked" {
			b.Warning = true
		}
		bindings = append(bindings, b)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[protocol-bindings] rows iteration error: %v", err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "Failed to read bindings"})
		return
	}

	if bindings == nil {
		bindings = []bindingResponse{}
	}

	// Group by protocol
	grouped := make(map[string][]bindingResponse)
	for _, b := range bindings {
		grouped[b.Protocol] = append(grouped[b.Protocol], b)
	}

	writeJSON(w, map[string]any{"ok": true, "bindings": bindings, "grouped": grouped})
}

// createProtocolBinding handles POST /api/admin/nodes/{nodeId}/bindings.
// Validates protocol, checks domain exists and is active, validates position, checks for duplicates.
func (s *Server) createProtocolBinding(w http.ResponseWriter, r *http.Request, nodeID int64) {
	limitBody(w, r, maxJSONBody)

	var in struct {
		Protocol string `json:"protocol"`
		DomainID int64  `json:"domain_id"`
		Position int    `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bad_json", "message": "Invalid JSON body"})
		return
	}

	// Validate protocol enum
	if !validProtocols[in.Protocol] {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_protocol", "message": "Protocol must be one of: openvpn-udp, openvpn-tcp, l2tp, ikev2, wireguard, ssh, mtproto"})
		return
	}

	// Validate position (1-10)
	if in.Position < 1 || in.Position > 10 {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "position_limit_exceeded", "message": "Position must be between 1 and 10"})
		return
	}

	// Check domain exists and is active
	var domainStatus string
	err := s.DB.QueryRowContext(r.Context(), `SELECT status FROM vpn_domains WHERE id = $1`, in.DomainID).Scan(&domainStatus)
	if err == sql.ErrNoRows {
		writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found", "message": "Domain not found"})
		return
	}
	if err != nil {
		log.Printf("[protocol-bindings] GetDomain query error: %v", err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "Failed to check domain"})
		return
	}
	if domainStatus != "active" {
		writeJSONCode(w, http.StatusUnprocessableEntity, map[string]any{"ok": false, "error": "domain_not_active", "message": "Domain must be active to create a binding"})
		return
	}

	// Check for duplicate (node_id, protocol, domain_id)
	var existingID int64
	err = s.DB.QueryRowContext(r.Context(), `SELECT id FROM vpn_protocol_bindings WHERE node_id = $1 AND protocol = $2 AND domain_id = $3`,
		nodeID, in.Protocol, in.DomainID).Scan(&existingID)
	if err == nil {
		writeJSONCode(w, http.StatusConflict, map[string]any{"ok": false, "error": "duplicate_binding", "message": "This domain is already bound to this protocol on this node"})
		return
	}
	if err != sql.ErrNoRows {
		log.Printf("[protocol-bindings] duplicate check error: %v", err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "Failed to check for duplicates"})
		return
	}

	// Check position limit (max 10 bindings per node+protocol)
	var bindingCount int
	err = s.DB.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM vpn_protocol_bindings WHERE node_id = $1 AND protocol = $2`,
		nodeID, in.Protocol).Scan(&bindingCount)
	if err != nil {
		log.Printf("[protocol-bindings] count error: %v", err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "Failed to count bindings"})
		return
	}
	if bindingCount >= 10 {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "position_limit_exceeded", "message": "Maximum 10 bindings per protocol per node"})
		return
	}

	// Create the binding
	var id int64
	var createdAt sql.NullTime
	err = s.DB.QueryRowContext(r.Context(), `
		INSERT INTO vpn_protocol_bindings (node_id, protocol, domain_id, position, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id, created_at
	`, nodeID, in.Protocol, in.DomainID, in.Position).Scan(&id, &createdAt)
	if err != nil {
		// Check for constraint violations (position conflict)
		if strings.Contains(err.Error(), "uq_binding_node_proto_position") || strings.Contains(err.Error(), "duplicate key") {
			writeJSONCode(w, http.StatusConflict, map[string]any{"ok": false, "error": "duplicate_binding", "message": "Position already occupied for this protocol on this node"})
			return
		}
		log.Printf("[protocol-bindings] create error: %v", err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "Failed to create binding"})
		return
	}

	createdAtStr := ""
	if createdAt.Valid {
		createdAtStr = createdAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}

	writeJSONCode(w, http.StatusCreated, map[string]any{
		"ok": true,
		"binding": map[string]any{
			"id":         id,
			"node_id":    nodeID,
			"protocol":   in.Protocol,
			"domain_id":  in.DomainID,
			"position":   in.Position,
			"created_at": createdAtStr,
		},
	})
}

// deleteProtocolBinding handles DELETE /api/admin/nodes/{nodeId}/bindings/{id}.
// Deletes the binding and re-sequences remaining positions to be contiguous starting at 1.
func (s *Server) deleteProtocolBinding(w http.ResponseWriter, r *http.Request, nodeID int64, bindingID int64) {
	ctx := r.Context()

	// Verify the binding exists and belongs to this node
	var protocol string
	err := s.DB.QueryRowContext(ctx, `SELECT protocol FROM vpn_protocol_bindings WHERE id = $1 AND node_id = $2`,
		bindingID, nodeID).Scan(&protocol)
	if err == sql.ErrNoRows {
		writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found", "message": "Binding not found"})
		return
	}
	if err != nil {
		log.Printf("[protocol-bindings] delete lookup error: %v", err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "Failed to find binding"})
		return
	}

	// Delete the binding
	_, err = s.DB.ExecContext(ctx, `DELETE FROM vpn_protocol_bindings WHERE id = $1`, bindingID)
	if err != nil {
		log.Printf("[protocol-bindings] delete exec error: %v", err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "Failed to delete binding"})
		return
	}

	// Re-sequence remaining positions for same (node_id, protocol) to be contiguous from 1
	_, err = s.DB.ExecContext(ctx, `
		WITH ranked AS (
			SELECT id, ROW_NUMBER() OVER (ORDER BY position) AS new_pos
			FROM vpn_protocol_bindings
			WHERE node_id = $1 AND protocol = $2
		)
		UPDATE vpn_protocol_bindings
		SET position = ranked.new_pos
		FROM ranked
		WHERE vpn_protocol_bindings.id = ranked.id
	`, nodeID, protocol)
	if err != nil {
		log.Printf("[protocol-bindings] re-sequence error: %v", err)
		// Non-fatal: binding is deleted, just log the re-sequence failure
	}

	writeJSON(w, map[string]any{"ok": true})
}

// reorderProtocolBindings handles PATCH /api/admin/nodes/{nodeId}/bindings/reorder.
// Accepts an array of binding IDs in desired order and updates positions atomically.
func (s *Server) reorderProtocolBindings(w http.ResponseWriter, r *http.Request, nodeID int64) {
	limitBody(w, r, maxJSONBody)

	var in struct {
		BindingIDs []int64 `json:"binding_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bad_json", "message": "Invalid JSON body"})
		return
	}

	if len(in.BindingIDs) == 0 {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bad_json", "message": "binding_ids must not be empty"})
		return
	}

	if len(in.BindingIDs) > 10 {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "position_limit_exceeded", "message": "Maximum 10 bindings allowed"})
		return
	}

	ctx := r.Context()

	// Verify all binding IDs belong to this node and the same protocol
	var protocol string
	err := s.DB.QueryRowContext(ctx, `SELECT protocol FROM vpn_protocol_bindings WHERE id = $1 AND node_id = $2`,
		in.BindingIDs[0], nodeID).Scan(&protocol)
	if err == sql.ErrNoRows {
		writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found", "message": "Binding not found"})
		return
	}
	if err != nil {
		log.Printf("[protocol-bindings] reorder lookup error: %v", err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "Failed to verify bindings"})
		return
	}

	// Verify all IDs belong to same node and protocol
	for _, id := range in.BindingIDs[1:] {
		var p string
		err := s.DB.QueryRowContext(ctx, `SELECT protocol FROM vpn_protocol_bindings WHERE id = $1 AND node_id = $2`, id, nodeID).Scan(&p)
		if err == sql.ErrNoRows {
			writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found", "message": "Binding not found: " + strconv.FormatInt(id, 10)})
			return
		}
		if err != nil {
			log.Printf("[protocol-bindings] reorder verify error: %v", err)
			writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "Failed to verify bindings"})
			return
		}
		if p != protocol {
			writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bad_json", "message": "All bindings must be for the same protocol"})
			return
		}
	}

	// Update positions atomically: first set to negative (avoid unique constraint conflicts),
	// then set to final positions
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("[protocol-bindings] reorder begin tx error: %v", err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Set to negative positions first
	for i, id := range in.BindingIDs {
		_, err := tx.ExecContext(ctx, `UPDATE vpn_protocol_bindings SET position = $1 WHERE id = $2 AND node_id = $3`,
			-(i + 1), id, nodeID)
		if err != nil {
			log.Printf("[protocol-bindings] reorder negative set error: %v", err)
			writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "Failed to reorder bindings"})
			return
		}
	}

	// Set final positions
	for i, id := range in.BindingIDs {
		_, err := tx.ExecContext(ctx, `UPDATE vpn_protocol_bindings SET position = $1 WHERE id = $2 AND node_id = $3`,
			i+1, id, nodeID)
		if err != nil {
			log.Printf("[protocol-bindings] reorder final set error: %v", err)
			writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "Failed to reorder bindings"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[protocol-bindings] reorder commit error: %v", err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "Failed to commit reorder"})
		return
	}

	writeJSON(w, map[string]any{"ok": true})
}
