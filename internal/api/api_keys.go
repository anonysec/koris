//go:build !lite

package api

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// apiKeys handles GET/POST/DELETE /api/settings/api-keys
func (s *Server) apiKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listAPIKeys(w, r)
	case http.MethodPost:
		s.createAPIKey(w, r)
	case http.MethodDelete:
		s.deleteAPIKey(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) listAPIKeys(w http.ResponseWriter, _ *http.Request) {
	rows, err := s.DB.Query(`SELECT id, name, key_prefix, scopes, last_used_at, created_at, COALESCE(created_by,'') FROM api_keys ORDER BY id DESC`)
	if err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error"})
		return
	}
	defer rows.Close()

	type Key struct {
		ID         int64  `json:"id"`
		Name       string `json:"name"`
		KeyPrefix  string `json:"key_prefix"`
		Scopes     string `json:"scopes"`
		LastUsedAt string `json:"last_used_at"`
		CreatedAt  string `json:"created_at"`
		CreatedBy  string `json:"created_by"`
	}
	var keys []Key
	for rows.Next() {
		var k Key
		var lastUsed, created sql.NullTime
		if err := rows.Scan(&k.ID, &k.Name, &k.KeyPrefix, &k.Scopes, &lastUsed, &created, &k.CreatedBy); err != nil {
			continue
		}
		if lastUsed.Valid {
			k.LastUsedAt = lastUsed.Time.Format(time.RFC3339)
		}
		if created.Valid {
			k.CreatedAt = created.Time.Format(time.RFC3339)
		}
		keys = append(keys, k)
	}
	if keys == nil {
		keys = []Key{}
	}
	writeJSON(w, map[string]any{"ok": true, "keys": keys})
}

func (s *Server) createAPIKey(w http.ResponseWriter, r *http.Request) {
	limitBody(w, r, maxJSONBody)
	var in struct {
		Name   string `json:"name"`
		Scopes string `json:"scopes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bad_json"})
		return
	}
	if in.Name == "" {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "name_required"})
		return
	}
	if in.Scopes == "" {
		in.Scopes = "read"
	}

	// Generate a 32-byte random key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "key_generation_failed"})
		return
	}
	fullKey := "koris_" + hex.EncodeToString(keyBytes)
	keyPrefix := fullKey[:12] + "..."

	actor, _, _ := s.currentAdmin(r)
	_, err := s.DB.Exec(
		`INSERT INTO api_keys (name, key_hash, key_prefix, scopes, created_by) VALUES ($1, $2, $3, $4, $5)`,
		in.Name, fullKey, keyPrefix, in.Scopes, actor,
	)
	if err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": fmt.Sprintf("insert failed: %v", err)})
		return
	}

	s.logAudit(actor, "api_key_created", "api_key", in.Name, nil, map[string]any{"name": in.Name, "scopes": in.Scopes}, r.RemoteAddr)

	// Return the full key ONLY on creation — it won't be shown again
	writeJSON(w, map[string]any{"ok": true, "key": fullKey, "name": in.Name, "prefix": keyPrefix})
}

func (s *Server) deleteAPIKey(w http.ResponseWriter, r *http.Request) {
	limitBody(w, r, maxJSONBody)
	var in struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.ID <= 0 {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bad_request"})
		return
	}
	result, err := s.DB.Exec(`DELETE FROM api_keys WHERE id=$1`, in.ID)
	if err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "delete_failed"})
		return
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found"})
		return
	}
	actor, _, _ := s.currentAdmin(r)
	s.logAudit(actor, "api_key_deleted", "api_key", strconv.FormatInt(in.ID, 10), nil, nil, r.RemoteAddr)
	writeJSON(w, map[string]any{"ok": true})
}
