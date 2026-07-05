package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// domainRotateIP handles POST /api/admin/domains/{id}/rotate-ip.
// Validates the new IP, updates the domain's IP in a transaction,
// and inserts an audit trail record into vpn_domain_ip_history.
func (s *Server) domainRotateIP(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	// Get admin username from session
	adminUsername, _, _ := s.currentAdmin(r)

	limitBody(w, r, maxJSONBody)
	var in struct {
		NewIP string `json:"new_ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{
			"ok":      false,
			"error":   "bad_json",
			"message": "Invalid JSON request body",
		})
		return
	}

	in.NewIP = strings.TrimSpace(in.NewIP)

	// Validate new IP
	if net.ParseIP(in.NewIP) == nil {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{
			"ok":      false,
			"error":   "invalid_ip_address",
			"message": "The provided IP address is not a valid IPv4 or IPv6 address",
		})
		return
	}

	// Get current domain
	var currentIP, name, status string
	err := s.DB.QueryRow(`SELECT name, ip_address, status FROM vpn_domains WHERE id = $1`, id).Scan(&name, &currentIP, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSONCode(w, http.StatusNotFound, map[string]any{
				"ok":      false,
				"error":   "not_found",
				"message": "Domain not found",
			})
			return
		}
		log.Printf("[domain-rotation] GetDomain failed for id %d: %v", id, err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{
			"ok":      false,
			"error":   "db_error",
			"message": "Failed to retrieve domain",
		})
		return
	}

	// Reject if same as current IP
	if in.NewIP == currentIP {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{
			"ok":      false,
			"error":   "ip_unchanged",
			"message": "The new IP address is the same as the current IP address",
		})
		return
	}

	// Execute rotation in a transaction
	tx, err := s.DB.Begin()
	if err != nil {
		log.Printf("[domain-rotation] Begin tx failed: %v", err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{
			"ok":      false,
			"error":   "db_error",
			"message": "Failed to start transaction",
		})
		return
	}
	defer tx.Rollback()

	// Update domain IP and updated_at
	now := time.Now().UTC()
	_, err = tx.Exec(`UPDATE vpn_domains SET ip_address = $1, updated_at = $2 WHERE id = $3`, in.NewIP, now, id)
	if err != nil {
		log.Printf("[domain-rotation] Update domain IP failed for id %d: %v", id, err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{
			"ok":      false,
			"error":   "db_error",
			"message": "Failed to update domain IP",
		})
		return
	}

	// Insert audit trail record
	_, err = tx.Exec(`
		INSERT INTO vpn_domain_ip_history (domain_id, previous_ip, new_ip, admin_username, rotated_at)
		VALUES ($1, $2, $3, $4, $5)
	`, id, currentIP, in.NewIP, adminUsername, now)
	if err != nil {
		log.Printf("[domain-rotation] Insert IP history failed for domain %d: %v", id, err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{
			"ok":      false,
			"error":   "db_error",
			"message": "Failed to record IP rotation history",
		})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[domain-rotation] Commit failed for domain %d: %v", id, err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{
			"ok":      false,
			"error":   "db_error",
			"message": "Failed to commit rotation",
		})
		return
	}

	writeJSON(w, map[string]any{
		"ok": true,
		"domain": map[string]any{
			"id":         id,
			"name":       name,
			"ip_address": in.NewIP,
			"status":     status,
			"updated_at": now.Format(time.RFC3339),
		},
		"rotation": map[string]any{
			"previous_ip":    currentIP,
			"new_ip":         in.NewIP,
			"admin_username": adminUsername,
			"rotated_at":     now.Format(time.RFC3339),
		},
	})
}

// domainIPHistory handles GET /api/admin/domains/{id}/history.
// Returns the IP rotation history for a domain ordered by rotated_at DESC.
func (s *Server) domainIPHistory(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	// Verify domain exists
	var exists bool
	err := s.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM vpn_domains WHERE id = $1)`, id).Scan(&exists)
	if err != nil || !exists {
		writeJSONCode(w, http.StatusNotFound, map[string]any{
			"ok":      false,
			"error":   "not_found",
			"message": "Domain not found",
		})
		return
	}

	rows, err := s.DB.Query(`
		SELECT id, domain_id, previous_ip, new_ip, admin_username, rotated_at
		FROM vpn_domain_ip_history
		WHERE domain_id = $1
		ORDER BY rotated_at DESC
	`, id)
	if err != nil {
		log.Printf("[domain-rotation] ListHistory failed for domain %d: %v", id, err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{
			"ok":      false,
			"error":   "db_error",
			"message": "Failed to retrieve rotation history",
		})
		return
	}
	defer rows.Close()

	type historyEntry struct {
		ID            int64  `json:"id"`
		DomainID      int64  `json:"domain_id"`
		PreviousIP    string `json:"previous_ip"`
		NewIP         string `json:"new_ip"`
		AdminUsername string `json:"admin_username"`
		RotatedAt     string `json:"rotated_at"`
	}

	history := []historyEntry{}
	for rows.Next() {
		var h historyEntry
		var rotatedAt time.Time
		if err := rows.Scan(&h.ID, &h.DomainID, &h.PreviousIP, &h.NewIP, &h.AdminUsername, &rotatedAt); err != nil {
			log.Printf("[domain-rotation] Scan history row failed: %v", err)
			writeJSONCode(w, http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"error":   "db_error",
				"message": "Failed to parse rotation history",
			})
			return
		}
		h.RotatedAt = rotatedAt.Format(time.RFC3339)
		history = append(history, h)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[domain-rotation] Iterate history rows failed: %v", err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{
			"ok":      false,
			"error":   "db_error",
			"message": "Failed to iterate rotation history",
		})
		return
	}

	writeJSON(w, map[string]any{
		"ok":      true,
		"history": history,
	})
}
