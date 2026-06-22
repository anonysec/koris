package api

import (
	"encoding/json"
	"net/http"

	"KorisLite/internal/auth"
)

// SetupHandler returns a handler for initial admin account creation.
// Only works when no admins exist in the database.
func (s *Server) SetupHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, M{"ok": false, "error": "method"})
			return
		}

		// Check if any admin exists
		var count int
		s.DB.QueryRow(`SELECT COUNT(*) FROM admins`).Scan(&count)
		if count > 0 {
			writeJSON(w, http.StatusForbidden, M{"ok": false, "error": "setup_already_complete"})
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
		if len(in.Password) < 6 {
			writeJSON(w, http.StatusBadRequest, M{"ok": false, "error": "password_too_short"})
			return
		}

		hash, err := auth.HashPassword(in.Password)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, M{"ok": false, "error": "hash_error"})
			return
		}

		_, err = s.DB.Exec(`INSERT INTO admins (username, password_hash, role, is_active) VALUES (?, ?, 'owner', 1)`,
			in.Username, hash)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, M{"ok": false, "error": "db_error"})
			return
		}

		ok(w, M{"ok": true, "message": "admin_created", "username": in.Username})
	}
}
