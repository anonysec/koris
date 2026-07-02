package api

import (
	"crypto/subtle"
	"database/sql"
	"strconv"
	"sync"
	"time"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"KorisPanel/panel/internal/auth"
	"golang.org/x/crypto/bcrypt"
)


var loginAttempts = make(map[string]int)
var loginMu sync.Mutex

func checkLoginRate(ip string) bool {
	loginMu.Lock()
	defer loginMu.Unlock()
	now := time.Now()
	for k, v := range loginAttempts {
		if v < now.Hour()*100+now.Minute() {
			delete(loginAttempts, k)
		}
	}
	key := ip + ":" + strconv.Itoa(now.Hour()*100+now.Minute())
	loginAttempts[key]++
	return loginAttempts[key] <= 10
}

func (s *Server) adminLogin(w http.ResponseWriter, r *http.Request) {
	if !checkLoginRate(clientIP(r)) {
		writeJSONCode(w, http.StatusTooManyRequests, map[string]any{"ok": false, "error": "rate_limited"})
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bad_json"})
		return
	}
	in.Username = strings.TrimSpace(in.Username)
	if in.Username == "" || in.Password == "" {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "required"})
		return
	}
	ok, err := s.Auth.LoginAdmin(in.Username, in.Password)
	if err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if !ok {
		writeJSONCode(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "invalid"})
		return
	}
	auth.SetSession(w, auth.AdminCookieName, in.Username, s.Config.SessionSecret, s.Config.SecureCookies)
	var role string
	_ = s.DB.QueryRow(`SELECT role FROM admins WHERE username=$1`, in.Username).Scan(&role)
	writeJSON(w, map[string]any{"ok": true, "username": in.Username, "role": role})
}

func (s *Server) adminMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	username, role, ok := s.currentAdmin(r)
	if !ok {
		writeJSONCode(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "unauthorized"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "username": username, "role": role})
}

func (s *Server) adminLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	auth.ClearSession(w, auth.AdminCookieName, s.Config.SecureCookies)
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) customerLogin(w http.ResponseWriter, r *http.Request) {
	if !checkLoginRate(clientIP(r)) {
		writeJSONCode(w, http.StatusTooManyRequests, map[string]any{"ok": false, "error": "rate_limited"})
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bad_json"})
		return
	}
	in.Username = strings.TrimSpace(in.Username)
	if in.Username == "" || in.Password == "" {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "required"})
		return
	}

	// FIX: Verify password against radcheck
	var storedPassword string
	err := s.DB.QueryRow(`SELECT value FROM radcheck WHERE username=$1 AND attribute='User-Password' LIMIT 1`, in.Username).Scan(&storedPassword)
	if err == sql.ErrNoRows {
		writeJSONCode(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "invalid"})
		return
	}
	if err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error"})
		return
	}

	// Plaintext comparison (FreeRADIUS stores plaintext by default)
	if subtle.ConstantTimeCompare([]byte(in.Password), []byte(storedPassword)) != 1 {
		writeJSONCode(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "invalid"})
		return
	}

	_, _ = s.DB.Exec(`INSERT INTO customers(username,sub_token) VALUES($1,$2) ON CONFLICT (username) DO NOTHING`, in.Username, auth.RandomToken(24))
	_, _ = s.DB.Exec(`INSERT INTO wallets(username,credit) VALUES($1,0) ON CONFLICT (username) DO NOTHING`, in.Username)
	auth.SetSession(w, auth.CustomerCookieName, in.Username, s.Config.SessionSecret, s.Config.SecureCookies)
	writeJSON(w, map[string]any{"ok": true, "username": in.Username})
}

func (s *Server) customerLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	auth.ClearSession(w, auth.CustomerCookieName, s.Config.SecureCookies)
	writeJSON(w, map[string]any{"ok": true})
}
