//go:build !lite

package api

import "net/http"

// registerSettingsRoutes registers routes that are only available in the full build.
func (s *Server) registerSettingsRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/settings/api-keys", s.requireFullAdmin(s.apiKeys))
	mux.HandleFunc("/api/settings/maintenance-mode", s.requireFullAdmin(s.maintenanceMode))
	mux.HandleFunc("/api/settings/update-check", s.requireFullAdmin(s.updateCheck))
}
