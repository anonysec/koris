package api

import (
	"database/sql"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
)

// --- Domain CRUD Handlers ---

// adminDomains dispatches GET (list) and POST (create) for /api/admin/domains.
func (s *Server) adminDomains(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listDomains(w, r)
	case http.MethodPost:
		s.createDomain(w, r)
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

// adminDomainByID dispatches GET, PATCH, DELETE for /api/admin/domains/{id}.
func (s *Server) adminDomainByID(w http.ResponseWriter, r *http.Request) {
	id, action, ok := pathID(r.URL.Path, "/api/admin/domains/")
	if !ok {
		writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found", "message": "invalid domain ID"})
		return
	}

	if action != "" {
		writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found", "message": "unknown action"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getDomain(w, r, id)
	case http.MethodPatch:
		s.updateDomain(w, r, id)
	case http.MethodDelete:
		s.deleteDomain(w, r, id)
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

// listDomains returns all domains with binding count and cert status, ordered by created_at DESC.
func (s *Server) listDomains(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.Query(`
		SELECT
			d.id, d.name, d.ip_address, d.status, d.created_at, d.updated_at,
			COALESCE(b.binding_count, 0),
			COALESCE(
				CASE
					WHEN c.expires_at IS NULL THEN 'none'
					WHEN c.expires_at <= NOW() THEN 'expired'
					WHEN c.expires_at <= NOW() + INTERVAL '30 days' THEN 'expiring_soon'
					ELSE 'valid'
				END,
				'none'
			)
		FROM vpn_domains d
		LEFT JOIN (
			SELECT domain_id, COUNT(*) AS binding_count
			FROM vpn_protocol_bindings
			GROUP BY domain_id
		) b ON b.domain_id = d.id
		LEFT JOIN LATERAL (
			SELECT expires_at
			FROM vpn_certificates
			WHERE domain_id = d.id AND status IN ('active', 'pending')
			ORDER BY expires_at DESC
			LIMIT 1
		) c ON true
		ORDER BY d.created_at DESC
	`)
	if err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "failed to list domains"})
		return
	}
	defer rows.Close()

	type domainRow struct {
		ID           int64  `json:"id"`
		Name         string `json:"name"`
		IPAddress    string `json:"ip_address"`
		Status       string `json:"status"`
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at"`
		BindingCount int    `json:"binding_count"`
		CertStatus   string `json:"cert_status"`
	}

	domains := []domainRow{}
	for rows.Next() {
		var d domainRow
		var createdAt, updatedAt sql.NullTime
		if err := rows.Scan(&d.ID, &d.Name, &d.IPAddress, &d.Status, &createdAt, &updatedAt, &d.BindingCount, &d.CertStatus); err != nil {
			writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "failed to scan domain"})
			return
		}
		if createdAt.Valid {
			d.CreatedAt = createdAt.Time.UTC().Format("2006-01-02T15:04:05Z")
		}
		if updatedAt.Valid {
			d.UpdatedAt = updatedAt.Time.UTC().Format("2006-01-02T15:04:05Z")
		}
		domains = append(domains, d)
	}
	if err := rows.Err(); err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "failed to iterate domains"})
		return
	}

	writeJSON(w, map[string]any{"ok": true, "domains": domains})
}

// createDomain validates and creates a new domain with status 'active'.
func (s *Server) createDomain(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name      string `json:"name"`
		IPAddress string `json:"ip_address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bad_json", "message": "invalid request body"})
		return
	}

	in.Name = strings.TrimSpace(in.Name)
	in.IPAddress = strings.TrimSpace(in.IPAddress)

	// Validate domain name (RFC 1123)
	if !isValidDomainName(in.Name) {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_domain_name", "message": "domain name must be RFC 1123 compliant, ≤253 characters"})
		return
	}

	// Validate IP address
	if net.ParseIP(in.IPAddress) == nil {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_ip_address", "message": "IP address must be a valid IPv4 or IPv6 address"})
		return
	}

	// Check uniqueness
	var exists bool
	err := s.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM vpn_domains WHERE name = $1)`, in.Name).Scan(&exists)
	if err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "failed to check domain uniqueness"})
		return
	}
	if exists {
		writeJSONCode(w, http.StatusConflict, map[string]any{"ok": false, "error": "domain_exists", "message": "a domain with this name already exists"})
		return
	}

	// Create the domain
	var id int64
	var createdAt, updatedAt sql.NullTime
	err = s.DB.QueryRow(`
		INSERT INTO vpn_domains (name, ip_address, status, created_at, updated_at)
		VALUES ($1, $2, 'active', NOW(), NOW())
		RETURNING id, created_at, updated_at
	`, in.Name, in.IPAddress).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "failed to create domain"})
		return
	}

	domain := map[string]any{
		"id":         id,
		"name":       in.Name,
		"ip_address": in.IPAddress,
		"status":     "active",
	}
	if createdAt.Valid {
		domain["created_at"] = createdAt.Time.UTC().Format("2006-01-02T15:04:05Z")
	}
	if updatedAt.Valid {
		domain["updated_at"] = updatedAt.Time.UTC().Format("2006-01-02T15:04:05Z")
	}

	actor, _, _ := s.currentAdmin(r)
	s.logAudit(actor, "domain.created", "domain", strconv.FormatInt(id, 10), nil, map[string]any{"name": in.Name, "ip_address": in.IPAddress}, clientIP(r))

	writeJSONCode(w, http.StatusCreated, map[string]any{"ok": true, "domain": domain})
}

// getDomain returns a single domain by ID.
func (s *Server) getDomain(w http.ResponseWriter, r *http.Request, id int64) {
	var name, ipAddress, status string
	var createdAt, updatedAt sql.NullTime
	err := s.DB.QueryRow(`
		SELECT name, ip_address, status, created_at, updated_at
		FROM vpn_domains WHERE id = $1
	`, id).Scan(&name, &ipAddress, &status, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found", "message": "domain not found"})
		return
	}
	if err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "failed to get domain"})
		return
	}

	domain := map[string]any{
		"id":         id,
		"name":       name,
		"ip_address": ipAddress,
		"status":     status,
	}
	if createdAt.Valid {
		domain["created_at"] = createdAt.Time.UTC().Format("2006-01-02T15:04:05Z")
	}
	if updatedAt.Valid {
		domain["updated_at"] = updatedAt.Time.UTC().Format("2006-01-02T15:04:05Z")
	}

	writeJSON(w, map[string]any{"ok": true, "domain": domain})
}

// updateDomain updates IP or status fields for a domain.
func (s *Server) updateDomain(w http.ResponseWriter, r *http.Request, id int64) {
	var in struct {
		IPAddress *string `json:"ip_address"`
		Status    *string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bad_json", "message": "invalid request body"})
		return
	}

	// Validate IP if provided
	if in.IPAddress != nil {
		ip := strings.TrimSpace(*in.IPAddress)
		if net.ParseIP(ip) == nil {
			writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_ip_address", "message": "IP address must be a valid IPv4 or IPv6 address"})
			return
		}
		in.IPAddress = &ip
	}

	// Validate status if provided
	validStatuses := map[string]bool{"active": true, "blocked": true, "retired": true}
	if in.Status != nil {
		s := strings.TrimSpace(*in.Status)
		if !validStatuses[s] {
			writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_status", "message": "status must be one of: active, blocked, retired"})
			return
		}
		in.Status = &s
	}

	// Check domain exists
	var currentIP, currentStatus string
	err := s.DB.QueryRow(`SELECT ip_address, status FROM vpn_domains WHERE id = $1`, id).Scan(&currentIP, &currentStatus)
	if err == sql.ErrNoRows {
		writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found", "message": "domain not found"})
		return
	}
	if err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "failed to get domain"})
		return
	}

	// Build update
	newIP := currentIP
	newStatus := currentStatus
	if in.IPAddress != nil {
		newIP = *in.IPAddress
	}
	if in.Status != nil {
		newStatus = *in.Status
	}

	var name string
	var createdAt, updatedAt sql.NullTime
	err = s.DB.QueryRow(`
		UPDATE vpn_domains
		SET ip_address = $2, status = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING name, ip_address, status, created_at, updated_at
	`, id, newIP, newStatus).Scan(&name, &newIP, &newStatus, &createdAt, &updatedAt)
	if err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "failed to update domain"})
		return
	}

	domain := map[string]any{
		"id":         id,
		"name":       name,
		"ip_address": newIP,
		"status":     newStatus,
	}
	if createdAt.Valid {
		domain["created_at"] = createdAt.Time.UTC().Format("2006-01-02T15:04:05Z")
	}
	if updatedAt.Valid {
		domain["updated_at"] = updatedAt.Time.UTC().Format("2006-01-02T15:04:05Z")
	}

	actor, _, _ := s.currentAdmin(r)
	s.logAudit(actor, "domain.updated", "domain", strconv.FormatInt(id, 10),
		map[string]any{"ip_address": currentIP, "status": currentStatus},
		map[string]any{"ip_address": newIP, "status": newStatus},
		clientIP(r))

	writeJSON(w, map[string]any{"ok": true, "domain": domain})
}

// deleteDomain checks for active bindings and deletes a domain.
func (s *Server) deleteDomain(w http.ResponseWriter, r *http.Request, id int64) {
	// Check domain exists
	var name string
	err := s.DB.QueryRow(`SELECT name FROM vpn_domains WHERE id = $1`, id).Scan(&name)
	if err == sql.ErrNoRows {
		writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found", "message": "domain not found"})
		return
	}
	if err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "failed to get domain"})
		return
	}

	// Check for active protocol bindings
	rows, err := s.DB.Query(`
		SELECT id, node_id, protocol, position
		FROM vpn_protocol_bindings
		WHERE domain_id = $1
		ORDER BY node_id, protocol, position
	`, id)
	if err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "failed to check bindings"})
		return
	}
	defer rows.Close()

	type bindingRef struct {
		ID       int64  `json:"id"`
		NodeID   int64  `json:"node_id"`
		Protocol string `json:"protocol"`
		Position int    `json:"position"`
	}
	var bindings []bindingRef
	for rows.Next() {
		var b bindingRef
		if err := rows.Scan(&b.ID, &b.NodeID, &b.Protocol, &b.Position); err != nil {
			writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "failed to scan binding"})
			return
		}
		bindings = append(bindings, b)
	}
	if err := rows.Err(); err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "failed to iterate bindings"})
		return
	}

	if len(bindings) > 0 {
		writeJSONCode(w, http.StatusConflict, map[string]any{
			"ok":       false,
			"error":    "domain_in_use",
			"message":  "cannot delete domain with active protocol bindings",
			"bindings": bindings,
		})
		return
	}

	// Delete the domain
	_, err = s.DB.Exec(`DELETE FROM vpn_domains WHERE id = $1`, id)
	if err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "failed to delete domain"})
		return
	}

	actor, _, _ := s.currentAdmin(r)
	s.logAudit(actor, "domain.deleted", "domain", strconv.FormatInt(id, 10), map[string]any{"name": name}, nil, clientIP(r))

	writeJSON(w, map[string]any{"ok": true})
}

// --- Domain Name Validation (RFC 1123) ---

// isValidDomainName validates a domain name per RFC 1123:
// - Total length ≤ 253 characters
// - Labels separated by dots
// - Each label: 1-63 chars, lowercase a-z, 0-9, hyphens
// - No leading or trailing hyphens per label
func isValidDomainName(name string) bool {
	if name == "" || len(name) > 253 {
		return false
	}

	labels := strings.Split(name, ".")
	if len(labels) < 2 {
		return false
	}

	for _, label := range labels {
		if !isValidLabel(label) {
			return false
		}
	}
	return true
}

// isValidLabel checks a single DNS label per RFC 1123.
func isValidLabel(label string) bool {
	n := len(label)
	if n == 0 || n > 63 {
		return false
	}
	// No leading or trailing hyphens
	if label[0] == '-' || label[n-1] == '-' {
		return false
	}
	for _, ch := range label {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-') {
			return false
		}
	}
	return true
}
