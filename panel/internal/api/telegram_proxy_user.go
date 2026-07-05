//go:build !lite

package api

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
)

// userTelegramProxy is a per-customer MTProto proxy credential with limits.
type userTelegramProxy struct {
	ID                  int64  `json:"id"`
	UserID              int64  `json:"user_id"`
	NodeID              int64  `json:"node_id"`
	Port                int    `json:"port"`
	Secret              string `json:"secret"`
	Token               string `json:"token"`
	MaxConnections      int    `json:"max_connections"`
	BandwidthLimitBytes int64  `json:"bandwidth_limit_bytes"`
	UsedConnections     int    `json:"used_connections"`
	Status              string `json:"status"`
	NodeName            string `json:"node_name,omitempty"`
	ShareLink           string `json:"share_link,omitempty"`
	CreatedAt           string `json:"created_at"`
}

func genHexToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// adminUserTelegramProxy handles /api/admin/user-telegram-proxy/{userID} by method.
func (s *Server) adminUserTelegramProxy(w http.ResponseWriter, r *http.Request) {
	id, _, ok := pathID(r.URL.Path, "/api/admin/user-telegram-proxy/")
	if !ok {
		writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.getUserTelegramProxy(w, r, id)
	case http.MethodPost:
		s.createUserTelegramProxy(w, r, id)
	case http.MethodPut:
		s.updateUserTelegramProxy(w, r, id)
	case http.MethodDelete:
		s.deleteUserTelegramProxy(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getUserTelegramProxy(w http.ResponseWriter, r *http.Request, userID int64) {
	ctx := context.Background()
	var p userTelegramProxy
	var nodeName string
	err := s.DB.QueryRowContext(ctx, `
		SELECT utp.id, utp.user_id, utp.node_id, utp.port, utp.secret, utp.token,
		       utp.max_connections, utp.bandwidth_limit_bytes, utp.used_connections,
		       utp.status, utp.created_at, COALESCE(n.name, '')
		FROM user_telegram_proxies utp
		LEFT JOIN nodes n ON n.id = utp.node_id
		WHERE utp.user_id = $1`, userID).
		Scan(&p.ID, &p.UserID, &p.NodeID, &p.Port, &p.Secret, &p.Token,
			&p.MaxConnections, &p.BandwidthLimitBytes, &p.UsedConnections,
			&p.Status, &p.CreatedAt, &nodeName)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, map[string]any{"ok": true, "proxy": nil})
			return
		}
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	p.NodeName = nodeName
	if nodeIP := s.getNodeIP(p.NodeID); nodeIP != "" {
		p.ShareLink = fmt.Sprintf("https://t.me/proxy?server=%s&port=%d&secret=%s", nodeIP, p.Port, p.Secret)
	}
	writeJSON(w, map[string]any{"ok": true, "proxy": p})
}

func (s *Server) createUserTelegramProxy(w http.ResponseWriter, r *http.Request, userID int64) {
	limitBody(w, r, maxJSONBody)
	var in struct {
		NodeID         int64 `json:"node_id"`
		Port           int   `json:"port"`
		MaxConnections int   `json:"max_connections"`
		BandwidthLimit int64 `json:"bandwidth_limit_bytes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bad_json"})
		return
	}
	if in.NodeID == 0 {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "node_id required"})
		return
	}
	if in.Port == 0 {
		in.Port = 443
	}
	secret, err := genHexToken()
	if err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "token_gen"})
		return
	}
	token, err := genHexToken()
	if err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "token_gen"})
		return
	}
	// One credential per user: replace any existing one.
	if _, err := s.DB.ExecContext(context.Background(), `DELETE FROM user_telegram_proxies WHERE user_id=$1`, userID); err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	var p userTelegramProxy
	err = s.DB.QueryRowContext(context.Background(), `
		INSERT INTO user_telegram_proxies (user_id, node_id, port, secret, token, max_connections, bandwidth_limit_bytes, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,'active')
		RETURNING id, user_id, node_id, port, secret, token, max_connections, bandwidth_limit_bytes, used_connections, status, created_at`,
		userID, in.NodeID, in.Port, secret, token, in.MaxConnections, in.BandwidthLimit).
		Scan(&p.ID, &p.UserID, &p.NodeID, &p.Port, &p.Secret, &p.Token, &p.MaxConnections, &p.BandwidthLimitBytes, &p.UsedConnections, &p.Status, &p.CreatedAt)
	if err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if nodeIP := s.getNodeIP(p.NodeID); nodeIP != "" {
		p.ShareLink = fmt.Sprintf("https://t.me/proxy?server=%s&port=%d&secret=%s", nodeIP, p.Port, p.Secret)
	}
	writeJSON(w, map[string]any{"ok": true, "proxy": p})
}

func (s *Server) updateUserTelegramProxy(w http.ResponseWriter, r *http.Request, userID int64) {
	limitBody(w, r, maxJSONBody)
	var in struct {
		MaxConnections int   `json:"max_connections"`
		BandwidthLimit int64 `json:"bandwidth_limit_bytes"`
		Regenerate     bool  `json:"regenerate"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bad_json"})
		return
	}
	ctx := context.Background()
	if in.Regenerate {
		secret, _ := genHexToken()
		token, _ := genHexToken()
		if _, err := s.DB.ExecContext(ctx, `UPDATE user_telegram_proxies SET secret=$1, token=$2, updated_at=NOW() WHERE user_id=$3`, secret, token, userID); err != nil {
			writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
			return
		}
	}
	if _, err := s.DB.ExecContext(ctx, `UPDATE user_telegram_proxies SET max_connections=$1, bandwidth_limit_bytes=$2, updated_at=NOW() WHERE user_id=$3`, in.MaxConnections, in.BandwidthLimit, userID); err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	s.getUserTelegramProxy(w, r, userID)
}

func (s *Server) deleteUserTelegramProxy(w http.ResponseWriter, r *http.Request, userID int64) {
	if _, err := s.DB.ExecContext(context.Background(), `DELETE FROM user_telegram_proxies WHERE user_id=$1`, userID); err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

// portalUserTelegramProxy returns the current customer's own proxy credential.
func (s *Server) portalUserTelegramProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	username, _ := s.currentCustomer(r)
	if username == "" {
		writeJSONCode(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "unauthorized"})
		return
	}
	var uid int64
	if err := s.DB.QueryRowContext(context.Background(), `SELECT COALESCE(id,0) FROM customers WHERE username=$1 AND deleted_at IS NULL LIMIT 1`, username).Scan(&uid); err != nil || uid == 0 {
		writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "user_not_found"})
		return
	}
	s.getUserTelegramProxy(w, r, uid)
}
