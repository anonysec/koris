package proxyconfig

import (
	"strings"
	"testing"
)

func TestDetectNginxVersion(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "standard output",
			output:   "nginx version: nginx/1.24.0",
			expected: "1.24.0",
		},
		{
			name:     "with additional info",
			output:   "nginx version: nginx/1.25.1 (Ubuntu)",
			expected: "1.25.1",
		},
		{
			name:     "older version",
			output:   "nginx version: nginx/1.18.0",
			expected: "1.18.0",
		},
		{
			name:     "development version output with extra lines",
			output:   "nginx version: nginx/1.27.0\nbuilt with OpenSSL 3.0.2",
			expected: "1.27.0",
		},
		{
			name:     "empty output",
			output:   "",
			expected: "",
		},
		{
			name:     "malformed output",
			output:   "some unrelated text",
			expected: "",
		},
		{
			name:     "version without patch",
			output:   "nginx version: nginx/1.24",
			expected: "",
		},
		{
			name:     "stderr style output",
			output:   "nginx version: nginx/1.22.1\n",
			expected: "1.22.1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := DetectNginxVersion(tc.output)
			if result != tc.expected {
				t.Errorf("DetectNginxVersion(%q) = %q, want %q", tc.output, result, tc.expected)
			}
		})
	}
}

func TestIsHTTP2Supported(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected bool
	}{
		{"exactly 1.9.5", "1.9.5", true},
		{"above 1.9.5", "1.10.0", true},
		{"modern version", "1.24.0", true},
		{"latest version", "1.25.1", true},
		{"below 1.9.5", "1.9.4", false},
		{"very old version", "1.8.0", false},
		{"ancient version", "0.9.0", false},
		{"empty version", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsHTTP2Supported(tc.version)
			if result != tc.expected {
				t.Errorf("IsHTTP2Supported(%q) = %v, want %v", tc.version, result, tc.expected)
			}
		})
	}
}

func TestIsHTTP3Supported(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected bool
	}{
		{"exactly 1.25.0", "1.25.0", true},
		{"above 1.25.0", "1.25.1", true},
		{"future version", "1.27.0", true},
		{"below 1.25.0", "1.24.0", false},
		{"much older", "1.18.0", false},
		{"just below", "1.24.9", false},
		{"empty version", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsHTTP3Supported(tc.version)
			if result != tc.expected {
				t.Errorf("IsHTTP3Supported(%q) = %v, want %v", tc.version, result, tc.expected)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		{"equal", "1.24.0", "1.24.0", 0},
		{"a greater major", "2.0.0", "1.24.0", 1},
		{"a lesser major", "1.0.0", "2.0.0", -1},
		{"a greater minor", "1.25.0", "1.24.0", 1},
		{"a lesser minor", "1.23.0", "1.24.0", -1},
		{"a greater patch", "1.24.1", "1.24.0", 1},
		{"a lesser patch", "1.24.0", "1.24.1", -1},
		{"empty a", "", "1.24.0", -1},
		{"empty b", "1.24.0", "", -1},
		{"both empty", "", "", -1},
		{"minor vs major diff", "1.10.0", "1.9.5", 1},
		{"minor crossover", "1.9.5", "1.10.0", -1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := compareVersions(tc.a, tc.b)
			if result != tc.expected {
				t.Errorf("compareVersions(%q, %q) = %d, want %d", tc.a, tc.b, result, tc.expected)
			}
		})
	}
}

func TestGenerateCompatibleNginx_ModernHTTP2(t *testing.T) {
	params := ProxyParams{
		Domain:      "panel.example.com",
		BackendAddr: "127.0.0.1:8080",
		SSLCertPath: "/etc/letsencrypt/live/panel.example.com/fullchain.pem",
		SSLKeyPath:  "/etc/letsencrypt/live/panel.example.com/privkey.pem",
		SSLEnabled:  true,
	}
	info := NginxInfo{
		Installed:  true,
		Version:    "1.25.1",
		ConfigPath: "/etc/nginx/nginx.conf",
		SitesPath:  "/etc/nginx/sites-enabled/",
	}

	result := GenerateCompatibleNginx(params, info)

	checks := []struct {
		name    string
		pattern string
		present bool
	}{
		{"version comment", "# Detected nginx version: 1.25.1", true},
		{"modern http2 on", "http2 on;", true},
		{"no legacy http2 in listen", "listen 443 ssl http2;", false},
		{"listen 443 ssl", "listen 443 ssl;", true},
		{"ssl_certificate", "ssl_certificate /etc/letsencrypt/live/panel.example.com/fullchain.pem", true},
		{"ssl_certificate_key", "ssl_certificate_key /etc/letsencrypt/live/panel.example.com/privkey.pem", true},
		{"proxy_pass", "proxy_pass http://127.0.0.1:8080", true},
		{"websocket upgrade", "proxy_set_header Upgrade $http_upgrade", true},
		{"health check", "location /api/health", true},
		{"http redirect", "return 301 https://$server_name$request_uri", true},
	}

	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			contains := strings.Contains(result, c.pattern)
			if c.present && !contains {
				t.Errorf("expected config to contain %q but it doesn't:\n%s", c.pattern, result)
			}
			if !c.present && contains {
				t.Errorf("expected config NOT to contain %q but it does:\n%s", c.pattern, result)
			}
		})
	}
}

func TestGenerateCompatibleNginx_LegacyHTTP2(t *testing.T) {
	params := ProxyParams{
		Domain:      "panel.example.com",
		BackendAddr: "127.0.0.1:8080",
		SSLCertPath: "/etc/letsencrypt/live/panel.example.com/fullchain.pem",
		SSLKeyPath:  "/etc/letsencrypt/live/panel.example.com/privkey.pem",
		SSLEnabled:  true,
	}
	info := NginxInfo{
		Installed:  true,
		Version:    "1.24.0",
		ConfigPath: "/etc/nginx/nginx.conf",
		SitesPath:  "/etc/nginx/sites-enabled/",
	}

	result := GenerateCompatibleNginx(params, info)

	checks := []struct {
		name    string
		pattern string
		present bool
	}{
		{"version comment", "# Detected nginx version: 1.24.0", true},
		{"legacy http2 in listen", "listen 443 ssl http2;", true},
		{"no modern http2 on", "http2 on;", false},
		{"ssl_certificate", "ssl_certificate /etc/letsencrypt/live/panel.example.com/fullchain.pem", true},
		{"proxy_pass", "proxy_pass http://127.0.0.1:8080", true},
	}

	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			contains := strings.Contains(result, c.pattern)
			if c.present && !contains {
				t.Errorf("expected config to contain %q but it doesn't:\n%s", c.pattern, result)
			}
			if !c.present && contains {
				t.Errorf("expected config NOT to contain %q but it does:\n%s", c.pattern, result)
			}
		})
	}
}

func TestGenerateCompatibleNginx_NoHTTP2(t *testing.T) {
	params := ProxyParams{
		Domain:      "panel.example.com",
		BackendAddr: "127.0.0.1:8080",
		SSLCertPath: "/etc/letsencrypt/live/panel.example.com/fullchain.pem",
		SSLKeyPath:  "/etc/letsencrypt/live/panel.example.com/privkey.pem",
		SSLEnabled:  true,
	}
	info := NginxInfo{
		Installed:  true,
		Version:    "1.8.0",
		ConfigPath: "/etc/nginx/nginx.conf",
		SitesPath:  "/etc/nginx/sites-enabled/",
	}

	result := GenerateCompatibleNginx(params, info)

	checks := []struct {
		name    string
		pattern string
		present bool
	}{
		{"version comment", "# Detected nginx version: 1.8.0", true},
		{"no http2 in listen", "listen 443 ssl http2;", false},
		{"no http2 on", "http2 on;", false},
		{"plain listen 443 ssl", "listen 443 ssl;", true},
		{"ssl_certificate", "ssl_certificate /etc/letsencrypt/live/panel.example.com/fullchain.pem", true},
		{"proxy_pass", "proxy_pass http://127.0.0.1:8080", true},
	}

	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			contains := strings.Contains(result, c.pattern)
			if c.present && !contains {
				t.Errorf("expected config to contain %q but it doesn't:\n%s", c.pattern, result)
			}
			if !c.present && contains {
				t.Errorf("expected config NOT to contain %q but it does:\n%s", c.pattern, result)
			}
		})
	}
}

func TestGenerateCompatibleNginx_NoSSL(t *testing.T) {
	params := ProxyParams{
		Domain:      "panel.example.com",
		BackendAddr: "127.0.0.1:8080",
		SSLEnabled:  false,
	}
	info := NginxInfo{
		Installed:  true,
		Version:    "1.24.0",
		ConfigPath: "/etc/nginx/nginx.conf",
		SitesPath:  "/etc/nginx/sites-enabled/",
	}

	result := GenerateCompatibleNginx(params, info)

	if strings.Contains(result, "listen 443") {
		t.Error("non-SSL config should not contain listen 443")
	}
	if strings.Contains(result, "ssl_certificate") {
		t.Error("non-SSL config should not contain ssl_certificate")
	}
	if strings.Contains(result, "http2") {
		t.Error("non-SSL config should not contain http2 directives")
	}
	if !strings.Contains(result, "listen 80") {
		t.Error("non-SSL config should listen on port 80")
	}
	if !strings.Contains(result, "proxy_pass http://127.0.0.1:8080") {
		t.Error("non-SSL config should still proxy to backend")
	}
}

func TestGenerateCompatibleNginx_UnknownVersion(t *testing.T) {
	params := ProxyParams{
		Domain:      "panel.example.com",
		BackendAddr: "127.0.0.1:8080",
		SSLCertPath: "/etc/letsencrypt/live/panel.example.com/fullchain.pem",
		SSLKeyPath:  "/etc/letsencrypt/live/panel.example.com/privkey.pem",
		SSLEnabled:  true,
	}
	info := NginxInfo{
		Installed:  true,
		Version:    "",
		ConfigPath: "",
		SitesPath:  "",
	}

	result := GenerateCompatibleNginx(params, info)

	// With unknown version, should fall back to plain ssl (no http2)
	if strings.Contains(result, "http2") {
		t.Error("unknown version should not include http2 directives")
	}
	if !strings.Contains(result, "listen 443 ssl;") {
		t.Error("unknown version should use plain listen 443 ssl")
	}
}

func TestNginxInfo_Defaults(t *testing.T) {
	info := NginxInfo{}
	if info.Installed {
		t.Error("default NginxInfo should have Installed = false")
	}
	if info.Version != "" {
		t.Error("default NginxInfo should have empty Version")
	}
	if info.ConfigPath != "" {
		t.Error("default NginxInfo should have empty ConfigPath")
	}
	if info.SitesPath != "" {
		t.Error("default NginxInfo should have empty SitesPath")
	}
}
