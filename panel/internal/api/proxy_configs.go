package api

import (
	"net/http"

	"KorisPanel/panel/internal/proxyconfig"
)

// proxyConfigEntry represents a single proxy type configuration in the response.
type proxyConfigEntry struct {
	Type     string `json:"type"`
	Config   string `json:"config"`
	Detected bool   `json:"detected"`
	Version  string `json:"version,omitempty"`
}

// nginxInfoResponse is the nginx-specific detection info included in the response.
type nginxInfoResponse struct {
	Installed  bool   `json:"installed"`
	Version    string `json:"version,omitempty"`
	ConfigPath string `json:"config_path,omitempty"`
	SitesPath  string `json:"sites_path,omitempty"`
}

// handleProxyConfigs handles GET /api/admin/proxy-configs
// Returns example reverse proxy configurations for all supported proxy types.
// Optional query param: ?type=nginx|caddy|traefik|haproxy to filter to one type.
func (s *Server) handleProxyConfigs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONCode(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
		return
	}

	// Build ProxyParams from panel config.
	domain := s.Config.TLSDomain
	if domain == "" {
		// Fallback: check panel_settings table for panel_domain.
		_ = s.DB.QueryRow(`SELECT setting_value FROM panel_settings WHERE setting_key='panel_domain'`).Scan(&domain)
	}
	if domain == "" {
		domain = "panel.example.com"
	}

	backendAddr := s.Config.Addr
	if backendAddr == "" {
		backendAddr = ":8080"
	}
	// Normalize for proxy configs — if listening on all interfaces, use 127.0.0.1.
	if len(backendAddr) > 0 && backendAddr[0] == ':' {
		backendAddr = "127.0.0.1" + backendAddr
	}

	sslEnabled := s.Config.TLSEnabled
	sslCertPath := s.Config.TLSCert
	sslKeyPath := s.Config.TLSKey

	// If TLS is not explicitly enabled but Let's Encrypt paths exist, use those.
	if !sslEnabled && domain != "panel.example.com" {
		leCert := "/etc/letsencrypt/live/" + domain + "/fullchain.pem"
		leKey := "/etc/letsencrypt/live/" + domain + "/privkey.pem"
		if _, ok := fileExists(leCert); ok {
			if _, ok2 := fileExists(leKey); ok2 {
				sslEnabled = true
				sslCertPath = leCert
				sslKeyPath = leKey
			}
		}
	}

	params := proxyconfig.ProxyParams{
		Domain:      domain,
		BackendAddr: backendAddr,
		SSLCertPath: sslCertPath,
		SSLKeyPath:  sslKeyPath,
		SSLEnabled:  sslEnabled,
	}

	// Detect nginx installation.
	nginxInfo := proxyconfig.DetectNginx()

	// Check if a specific type is requested.
	filterType := r.URL.Query().Get("type")

	// Generate configs for each proxy type.
	var configs []proxyConfigEntry

	allTypes := []proxyconfig.ProxyType{
		proxyconfig.ProxyNginx,
		proxyconfig.ProxyCaddy,
		proxyconfig.ProxyTraefik,
		proxyconfig.ProxyHAProxy,
	}

	for _, pt := range allTypes {
		if filterType != "" && string(pt) != filterType {
			continue
		}

		entry := proxyConfigEntry{
			Type:     string(pt),
			Detected: false,
		}

		switch pt {
		case proxyconfig.ProxyNginx:
			entry.Detected = nginxInfo.Installed
			if nginxInfo.Version != "" {
				entry.Version = nginxInfo.Version
			}
			// Use compatible config if nginx is detected; otherwise use generic.
			if nginxInfo.Installed {
				entry.Config = proxyconfig.GenerateCompatibleNginx(params, nginxInfo)
			} else {
				entry.Config = proxyconfig.GenerateNginx(params)
			}
		case proxyconfig.ProxyCaddy:
			entry.Config = proxyconfig.GenerateCaddy(params)
		case proxyconfig.ProxyTraefik:
			entry.Config = proxyconfig.GenerateTraefik(params)
		case proxyconfig.ProxyHAProxy:
			entry.Config = proxyconfig.GenerateHAProxy(params)
		}

		configs = append(configs, entry)
	}

	resp := map[string]any{
		"ok":      true,
		"configs": configs,
		"nginx_info": nginxInfoResponse{
			Installed:  nginxInfo.Installed,
			Version:    nginxInfo.Version,
			ConfigPath: nginxInfo.ConfigPath,
			SitesPath:  nginxInfo.SitesPath,
		},
	}

	writeJSON(w, resp)
}
