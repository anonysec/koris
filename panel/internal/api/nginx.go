package api

import (
	"log"
	"net/http"

	"KorisPanel/panel/internal/proxyconfig"
)

// handleNginxStatus handles GET /api/admin/nginx/status
// This is a deprecated endpoint — clients should use /api/admin/proxy-configs instead.
// Returns nginx installation status with deprecation notices.
func (s *Server) handleNginxStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONCode(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
		return
	}

	log.Printf("[nginx] deprecated API called: %s", r.URL.Path)

	// Detect nginx installation safely (won't crash if nginx is not installed).
	nginxInfo := proxyconfig.DetectNginx()

	// Set deprecation headers.
	w.Header().Set("Deprecated", "true")
	w.Header().Set("X-Deprecation-Notice", "Use /api/admin/proxy-configs for reverse proxy configuration")

	resp := map[string]any{
		"ok":                 true,
		"deprecated":         true,
		"deprecation_notice": "Nginx is now optional. Use /api/admin/proxy-configs for reverse proxy configuration.",
		"nginx_installed":    nginxInfo.Installed,
		"nginx_version":      nginxInfo.Version,
	}

	writeJSON(w, resp)
}
