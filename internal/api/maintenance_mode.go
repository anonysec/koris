//go:build !lite

package api

import (
	"encoding/json"
	"net/http"
	"os"
	"time"
)

// maintenanceMode handles GET/POST /api/settings/maintenance-mode
func (s *Server) maintenanceMode(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getMaintenanceMode(w)
	case http.MethodPost:
		s.setMaintenanceMode(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getMaintenanceMode(w http.ResponseWriter) {
	var enabled bool
	var reason, enabledBy, enabledAt string
	row := s.DB.QueryRow(`SELECT COALESCE(value,'') FROM panel_settings WHERE key='maintenance_mode'`)
	var val string
	_ = row.Scan(&val)
	enabled = val == "true"

	row2 := s.DB.QueryRow(`SELECT COALESCE(value,'') FROM panel_settings WHERE key='maintenance_reason'`)
	_ = row2.Scan(&reason)

	row3 := s.DB.QueryRow(`SELECT COALESCE(value,'') FROM panel_settings WHERE key='maintenance_enabled_by'`)
	_ = row3.Scan(&enabledBy)

	row4 := s.DB.QueryRow(`SELECT COALESCE(value,'') FROM panel_settings WHERE key='maintenance_enabled_at'`)
	_ = row4.Scan(&enabledAt)

	writeJSON(w, map[string]any{
		"ok":          true,
		"enabled":     enabled,
		"reason":      reason,
		"enabled_by":  enabledBy,
		"enabled_at":  enabledAt,
	})
}

func (s *Server) setMaintenanceMode(w http.ResponseWriter, r *http.Request) {
	limitBody(w, r, maxJSONBody)
	var in struct {
		Enabled bool   `json:"enabled"`
		Reason  string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bad_json"})
		return
	}

	actor, _, _ := s.currentAdmin(r)
	now := time.Now().Format(time.RFC3339)
	val := "false"
	if in.Enabled {
		val = "true"
	}

	// Upsert settings
	for _, kv := range []struct{ key, value string }{
		{"maintenance_mode", val},
		{"maintenance_reason", in.Reason},
		{"maintenance_enabled_by", actor},
		{"maintenance_enabled_at", now},
	} {
		_, _ = s.DB.Exec(
			`INSERT INTO panel_settings (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value=$2`,
			kv.key, kv.value,
		)
	}

	// Write maintenance flag file for the middleware to check without DB
	flagPath := "/tmp/koris_maintenance"
	if in.Enabled {
		_ = os.WriteFile(flagPath, []byte(in.Reason), 0644)
	} else {
		_ = os.Remove(flagPath)
	}

	s.logAudit(actor, "maintenance_mode_changed", "system", "",
		map[string]any{"enabled": !in.Enabled},
		map[string]any{"enabled": in.Enabled, "reason": in.Reason},
		r.RemoteAddr)

	writeJSON(w, map[string]any{"ok": true, "enabled": in.Enabled})
}
