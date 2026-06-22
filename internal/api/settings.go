package api

import (
	"encoding/json"
	"net/http"
)

func (s *Server) settings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getSettings(w, r)
	case http.MethodPost:
		s.updateSettings(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, M{"ok": false, "error": "method"})
	}
}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.Query(`SELECT key_name, value FROM panel_settings ORDER BY key_name`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"ok": false, "error": "db_error"})
		return
	}
	defer rows.Close()

	settings := map[string]string{}
	for rows.Next() {
		var k, v string
		if rows.Scan(&k, &v) == nil {
			settings[k] = v
		}
	}
	ok(w, M{"ok": true, "settings": settings})
}

func (s *Server) updateSettings(w http.ResponseWriter, r *http.Request) {
	limitBody(r)
	var in map[string]string
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, M{"ok": false, "error": "bad_json"})
		return
	}

	for key, val := range in {
		_, err := s.DB.Exec(`INSERT INTO panel_settings (key_name, value) VALUES (?, ?) ON DUPLICATE KEY UPDATE value=?`, key, val, val)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, M{"ok": false, "error": "db_error"})
			return
		}
	}

	ok(w, M{"ok": true})
}
