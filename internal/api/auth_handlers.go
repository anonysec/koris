package api

import (
	"encoding/json"
	"net/http"
	"time"

	"KorisLite/internal/auth"
)

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ok(w, M{"ok": true, "version": s.Config.Version, "time": time.Now().UTC().Format(time.RFC3339)})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, M{"ok": false, "error": "method"})
		return
	}
	limitBody(r)
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, M{"ok": false, "error": "bad_json"})
		return
	}
	if in.Username == "" || in.Password == "" {
		writeJSON(w, http.StatusBadRequest, M{"ok": false, "error": "username_and_password_required"})
		return
	}

	// Verify credentials
	var storedHash, role string
	err := s.DB.QueryRow(`SELECT password_hash, role FROM admins WHERE username=? AND is_active=1 LIMIT 1`,
		in.Username).Scan(&storedHash, &role)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, M{"ok": false, "error": "invalid_credentials"})
		return
	}
	if !auth.CheckPassword(storedHash, in.Password) {
		writeJSON(w, http.StatusUnauthorized, M{"ok": false, "error": "invalid_credentials"})
		return
	}

	// Create session
	sess, err := s.Auth.CreateSession(in.Username, role)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"ok": false, "error": "session_error"})
		return
	}

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sess.Token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})

	ok(w, M{"ok": true, "token": sess.Token, "username": in.Username, "role": role})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	token := auth.GetSessionFromRequest(r)
	if token != "" {
		s.Auth.DeleteSession(token)
	}
	http.SetCookie(w, &http.Cookie{Name: "session", Path: "/", MaxAge: -1})
	ok(w, M{"ok": true})
}

func (s *Server) adminMe(w http.ResponseWriter, r *http.Request) {
	username := s.currentAdmin(r)
	role := r.Header.Get("X-Admin-Role")
	ok(w, M{"ok": true, "username": username, "role": role})
}
