package api

import (
	"context"
	"github.com/anonysec/koris/internal/grpcclient"
	"github.com/anonysec/koris/internal/knodepb"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// adminCustomerMTProtoSecret dispatches requests for /api/admin/customers/{id}/mtproto-secret[/regenerate|enable|disable].
func (s *Server) adminCustomerMTProtoSecret(w http.ResponseWriter, r *http.Request) {
	// Parse path: /api/admin/customers/{id}/mtproto-secret[/action]
	rest := strings.TrimPrefix(r.URL.Path, "/api/admin/customers/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")

	// Must have at least ["{id}", "mtproto-secret"]
	if len(parts) < 2 || parts[1] != "mtproto-secret" {
		writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found"})
		return
	}

	customerID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || customerID <= 0 {
		writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found", "message": "invalid customer ID"})
		return
	}

	// /api/admin/customers/{id}/mtproto-secret/regenerate
	if len(parts) == 3 && parts[2] == "regenerate" {
		if r.Method == http.MethodPost {
			s.regenerateMTProtoSecret(w, r, customerID)
			return
		}
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	// /api/admin/customers/{id}/mtproto-secret/enable
	if len(parts) == 3 && parts[2] == "enable" {
		if r.Method == http.MethodPost {
			s.handleMTProtoEnable(w, r, customerID)
			return
		}
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	// /api/admin/customers/{id}/mtproto-secret/disable
	if len(parts) == 3 && parts[2] == "disable" {
		if r.Method == http.MethodPost {
			s.handleMTProtoDisable(w, r, customerID)
			return
		}
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	// /api/admin/customers/{id}/mtproto-secret
	if len(parts) == 2 {
		if r.Method == http.MethodGet {
			s.getMTProtoSecret(w, r, customerID)
			return
		}
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found"})
}

// getMTProtoSecret handles GET /api/admin/customers/{id}/mtproto-secret.
// Returns the customer's MTProto secret, enabled status, connection count, and connection limit.
func (s *Server) getMTProtoSecret(w http.ResponseWriter, r *http.Request, customerID int64) {
	var secret sql.NullString
	var enabled bool
	var connLimit int

	err := s.DB.QueryRowContext(r.Context(), `
		SELECT mtproto_secret, mtproto_enabled, mtproto_conn_limit
		FROM customers
		WHERE id = $1 AND deleted_at IS NULL
	`, customerID).Scan(&secret, &enabled, &connLimit)
	if err == sql.ErrNoRows {
		writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found", "message": "customer not found"})
		return
	}
	if err != nil {
		log.Printf("[mtproto-secrets] get secret query error for customer %d: %v", customerID, err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "failed to retrieve MTProto secret"})
		return
	}

	// Connection count — return 0 for now until gRPC sync provides real-time counts
	connectionCount := 0

	secretValue := ""
	if secret.Valid {
		secretValue = secret.String
	}

	writeJSON(w, map[string]any{
		"ok": true,
		"mtproto": map[string]any{
			"secret":           secretValue,
			"enabled":          enabled,
			"connection_count": connectionCount,
			"connection_limit": connLimit,
		},
	})
}

// regenerateMTProtoSecret handles POST /api/admin/customers/{id}/mtproto-secret/regenerate.
// Generates a new 32-byte random secret (hex-encoded, 64 chars), invalidates the old secret,
// and pushes the updated secret list to knode via gRPC.
func (s *Server) regenerateMTProtoSecret(w http.ResponseWriter, r *http.Request, customerID int64) {
	limitBody(w, r, maxJSONBody)

	// Verify customer exists
	var username string
	err := s.DB.QueryRowContext(r.Context(), `
		SELECT username FROM customers WHERE id = $1 AND deleted_at IS NULL
	`, customerID).Scan(&username)
	if err == sql.ErrNoRows {
		writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found", "message": "customer not found"})
		return
	}
	if err != nil {
		log.Printf("[mtproto-secrets] check customer existence error for %d: %v", customerID, err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "failed to verify customer"})
		return
	}

	// Generate new 32-byte random secret
	newSecret, err := generateMTProtoSecret()
	if err != nil {
		log.Printf("[mtproto-secrets] failed to generate secret: %v", err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "failed to generate secret"})
		return
	}

	// Store the new secret and enable MTProto
	_, err = s.DB.ExecContext(r.Context(), `
		UPDATE customers
		SET mtproto_secret = $2, mtproto_enabled = true, updated_at = NOW()
		WHERE id = $1
	`, customerID, newSecret)
	if err != nil {
		log.Printf("[mtproto-secrets] update secret error for customer %d: %v", customerID, err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "failed to store new secret"})
		return
	}

	// Push updated secrets to knode via gRPC (best-effort; log if unavailable)
	s.syncMTProtoSecretsToKnode(r.Context())

	actor, _, _ := s.currentAdmin(r)
	s.logAudit(actor, "mtproto.secret_regenerated", "customer", strconv.FormatInt(customerID, 10),
		nil, map[string]any{"username": username}, clientIP(r))

	writeJSON(w, map[string]any{
		"ok": true,
		"mtproto": map[string]any{
			"secret":  newSecret,
			"enabled": true,
		},
	})
}

// enableCustomerMTProtoByID generates a new secret and sets mtproto_enabled=true for a customer.
// Called when a customer is created or enabled for MTProto access.
func (s *Server) enableCustomerMTProtoByID(customerID int64) error {
	secret, err := generateMTProtoSecret()
	if err != nil {
		return err
	}

	_, err = s.DB.Exec(`
		UPDATE customers
		SET mtproto_secret = $2, mtproto_enabled = true, updated_at = NOW()
		WHERE id = $1
	`, customerID, secret)
	if err != nil {
		log.Printf("[mtproto-secrets] enableCustomerMTProtoByID error for customer %d: %v", customerID, err)
		return err
	}

	// Push updated secrets to knode (best-effort)
	s.syncMTProtoSecretsToKnode(nil)
	return nil
}

// disableCustomerMTProtoByID sets mtproto_enabled=false and pushes the updated list to knode.
// Called when a customer is disabled or deleted.
func (s *Server) disableCustomerMTProtoByID(customerID int64) error {
	_, err := s.DB.Exec(`
		UPDATE customers
		SET mtproto_enabled = false, updated_at = NOW()
		WHERE id = $1
	`, customerID)
	if err != nil {
		log.Printf("[mtproto-secrets] disableCustomerMTProto error for customer %d: %v", customerID, err)
		return err
	}

	// Push updated secrets to knode (best-effort)
	s.syncMTProtoSecretsToKnode(nil)
	return nil
}

// generateMTProtoSecret generates a cryptographically random 32-byte hex-encoded secret (64 chars).
func generateMTProtoSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// syncMTProtoSecretsToKnode pushes the current list of enabled MTProto secrets to all connected knodes.
// This is a best-effort operation — if the gRPC call is unavailable, it logs and returns.
func (s *Server) syncMTProtoSecretsToKnode(_ any) {
	// Collect all enabled MTProto secrets
	rows, err := s.DB.Query(`
		SELECT username, mtproto_secret, mtproto_conn_limit
		FROM customers
		WHERE mtproto_enabled = true AND mtproto_secret IS NOT NULL AND deleted_at IS NULL
	`)
	if err != nil {
		log.Printf("[mtproto-secrets] syncMTProtoSecretsToKnode: failed to query secrets: %v", err)
		return
	}
	defer rows.Close()

	type secretEntry struct {
		Username  string `json:"username"`
		Secret    string `json:"secret"`
		ConnLimit int    `json:"max_connections"`
	}

	var secrets []secretEntry
	for rows.Next() {
		var entry secretEntry
		if err := rows.Scan(&entry.Username, &entry.Secret, &entry.ConnLimit); err != nil {
			log.Printf("[mtproto-secrets] syncMTProtoSecretsToKnode: scan error: %v", err)
			return
		}
		secrets = append(secrets, entry)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[mtproto-secrets] syncMTProtoSecretsToKnode: rows error: %v", err)
		return
	}

	// Build proto secrets list
	protoSecrets := make([]*knodepb.MTProtoUserSecret, 0, len(secrets))
	for _, s := range secrets {
		protoSecrets = append(protoSecrets, &knodepb.MTProtoUserSecret{
			Username:       s.Username,
			Secret:         s.Secret,
			Enabled:        true,
			MaxConnections: int32(s.ConnLimit),
		})
	}

	// Push to all connected knodes
	for _, node := range s.GRPCPool.All() {
		if node.Status == grpcclient.StatusOffline {
			continue
		}
		client := knodepb.NewKnodeServiceClient(node.Conn)
		_, err := client.SyncMTProtoSecrets(context.Background(), &knodepb.SyncMTProtoSecretsRequest{
			Secrets: protoSecrets,
		})
		if err != nil {
			log.Printf("[mtproto-secrets] SyncMTProtoSecrets failed on node %q (id=%d): %v", node.NodeName, node.NodeID, err)
		} else {
			log.Printf("[mtproto-secrets] SyncMTProtoSecrets succeeded on node %q (id=%d): %d secrets", node.NodeName, node.NodeID, len(protoSecrets))
		}
	}
}

// handleMTProtoEnable handles POST /api/admin/customers/{id}/mtproto-secret/enable.
// Generates a secret and enables MTProto for the customer.
func (s *Server) handleMTProtoEnable(w http.ResponseWriter, r *http.Request, customerID int64) {
	// Verify customer exists
	var username string
	err := s.DB.QueryRowContext(r.Context(), `
		SELECT username FROM customers WHERE id = $1 AND deleted_at IS NULL
	`, customerID).Scan(&username)
	if err == sql.ErrNoRows {
		writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found", "message": "customer not found"})
		return
	}
	if err != nil {
		log.Printf("[mtproto-secrets] handleMTProtoEnable: check customer error for %d: %v", customerID, err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "failed to verify customer"})
		return
	}

	// Generate and store secret
	secret, err := generateMTProtoSecret()
	if err != nil {
		log.Printf("[mtproto-secrets] handleMTProtoEnable: generate secret error: %v", err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "failed to generate secret"})
		return
	}

	_, err = s.DB.ExecContext(r.Context(), `
		UPDATE customers
		SET mtproto_secret = $2, mtproto_enabled = true, updated_at = NOW()
		WHERE id = $1
	`, customerID, secret)
	if err != nil {
		log.Printf("[mtproto-secrets] handleMTProtoEnable: update error for customer %d: %v", customerID, err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "failed to enable MTProto"})
		return
	}

	// Push updated secrets to knode
	s.syncMTProtoSecretsToKnode(r.Context())

	actor, _, _ := s.currentAdmin(r)
	s.logAudit(actor, "mtproto.enabled", "customer", strconv.FormatInt(customerID, 10),
		nil, map[string]any{"username": username}, clientIP(r))

	writeJSON(w, map[string]any{
		"ok": true,
		"mtproto": map[string]any{
			"secret":  secret,
			"enabled": true,
		},
	})
}

// handleMTProtoDisable handles POST /api/admin/customers/{id}/mtproto-secret/disable.
// Disables MTProto for the customer and pushes the updated list to knode.
func (s *Server) handleMTProtoDisable(w http.ResponseWriter, r *http.Request, customerID int64) {
	// Verify customer exists
	var username string
	err := s.DB.QueryRowContext(r.Context(), `
		SELECT username FROM customers WHERE id = $1 AND deleted_at IS NULL
	`, customerID).Scan(&username)
	if err == sql.ErrNoRows {
		writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found", "message": "customer not found"})
		return
	}
	if err != nil {
		log.Printf("[mtproto-secrets] handleMTProtoDisable: check customer error for %d: %v", customerID, err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "failed to verify customer"})
		return
	}

	_, err = s.DB.ExecContext(r.Context(), `
		UPDATE customers
		SET mtproto_enabled = false, updated_at = NOW()
		WHERE id = $1
	`, customerID)
	if err != nil {
		log.Printf("[mtproto-secrets] handleMTProtoDisable: update error for customer %d: %v", customerID, err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error", "message": "failed to disable MTProto"})
		return
	}

	// Push updated secrets to knode
	s.syncMTProtoSecretsToKnode(r.Context())

	actor, _, _ := s.currentAdmin(r)
	s.logAudit(actor, "mtproto.disabled", "customer", strconv.FormatInt(customerID, 10),
		nil, map[string]any{"username": username}, clientIP(r))

	writeJSON(w, map[string]any{"ok": true})
}
