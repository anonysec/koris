package proxyconfig

import (
	"strings"
	"testing"
)

var sslParams = ProxyParams{
	Domain:      "panel.example.com",
	BackendAddr: "127.0.0.1:8080",
	SSLCertPath: "/etc/letsencrypt/live/panel.example.com/fullchain.pem",
	SSLKeyPath:  "/etc/letsencrypt/live/panel.example.com/privkey.pem",
	SSLEnabled:  true,
}

var noSSLParams = ProxyParams{
	Domain:      "panel.example.com",
	BackendAddr: "127.0.0.1:8080",
	SSLEnabled:  false,
}

func TestGenerateCaddy_SSL(t *testing.T) {
	result := GenerateCaddy(sslParams)

	checks := []struct {
		name    string
		pattern string
	}{
		{"header comment", "# KorisPanel Caddy Configuration"},
		{"domain block", "panel.example.com {"},
		{"tls directive", "tls /etc/letsencrypt/live/panel.example.com/fullchain.pem /etc/letsencrypt/live/panel.example.com/privkey.pem"},
		{"reverse_proxy", "reverse_proxy 127.0.0.1:8080"},
		{"websocket", "handle /api/ws/*"},
		{"health check", "handle /api/health"},
		{"x-real-ip", "header_up X-Real-IP"},
		{"x-forwarded-for", "header_up X-Forwarded-For"},
		{"x-forwarded-proto", "header_up X-Forwarded-Proto"},
	}

	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if !strings.Contains(result, c.pattern) {
				t.Errorf("caddy config missing %q:\n%s", c.pattern, result)
			}
		})
	}
}

func TestGenerateCaddy_NoSSL(t *testing.T) {
	result := GenerateCaddy(noSSLParams)

	if !strings.Contains(result, "http://panel.example.com {") {
		t.Error("non-SSL caddy config should use http:// prefix")
	}
	if strings.Contains(result, "tls ") {
		t.Error("non-SSL caddy config should not have tls directive")
	}
}

func TestGenerateTraefik_SSL(t *testing.T) {
	result := GenerateTraefik(sslParams)

	checks := []struct {
		name    string
		pattern string
	}{
		{"header comment", "# KorisPanel Traefik Configuration"},
		{"http routers", "routers:"},
		{"host rule", "Host(`panel.example.com`)"},
		{"https entrypoint", "websecure"},
		{"tls cert", "certFile: /etc/letsencrypt/live/panel.example.com/fullchain.pem"},
		{"tls key", "keyFile: /etc/letsencrypt/live/panel.example.com/privkey.pem"},
		{"redirect middleware", "korispanel-https-redirect"},
		{"redirect scheme", "scheme: https"},
		{"service backend", `url: "http://127.0.0.1:8080"`},
		{"health check path", "path: /api/health"},
		{"pass host header", "passHostHeader: true"},
	}

	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if !strings.Contains(result, c.pattern) {
				t.Errorf("traefik config missing %q:\n%s", c.pattern, result)
			}
		})
	}
}

func TestGenerateTraefik_NoSSL(t *testing.T) {
	result := GenerateTraefik(noSSLParams)

	if strings.Contains(result, "websecure") {
		t.Error("non-SSL traefik config should not reference websecure entrypoint")
	}
	if strings.Contains(result, "certFile") {
		t.Error("non-SSL traefik config should not have certFile")
	}
	if !strings.Contains(result, "web") {
		t.Error("non-SSL traefik config should use web entrypoint")
	}
}

func TestGenerateHAProxy_SSL(t *testing.T) {
	result := GenerateHAProxy(sslParams)

	checks := []struct {
		name    string
		pattern string
	}{
		{"header comment", "# KorisPanel HAProxy Configuration"},
		{"http frontend", "frontend korispanel_http"},
		{"https frontend", "frontend korispanel_https"},
		{"ssl bind", "bind *:443 ssl crt"},
		{"host acl", "acl is_panel hdr(host) -i panel.example.com"},
		{"websocket acl", "acl is_websocket path_beg /api/ws/"},
		{"http redirect", "http-request redirect scheme https"},
		{"x-real-ip", "http-request set-header X-Real-IP"},
		{"x-forwarded-for", "http-request set-header X-Forwarded-For"},
		{"x-forwarded-proto", "http-request set-header X-Forwarded-Proto https"},
		{"ws backend", "backend korispanel_ws"},
		{"main backend", "backend korispanel_backend"},
		{"server line", "server panel 127.0.0.1:8080 check"},
		{"health check", "option httpchk GET /api/health"},
		{"ws timeout", "timeout server 86400s"},
	}

	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if !strings.Contains(result, c.pattern) {
				t.Errorf("haproxy config missing %q:\n%s", c.pattern, result)
			}
		})
	}
}

func TestGenerateHAProxy_NoSSL(t *testing.T) {
	result := GenerateHAProxy(noSSLParams)

	if strings.Contains(result, "bind *:443") {
		t.Error("non-SSL haproxy config should not bind on 443")
	}
	if strings.Contains(result, "frontend korispanel_https") {
		t.Error("non-SSL haproxy config should not have https frontend")
	}
	if !strings.Contains(result, "frontend korispanel_http") {
		t.Error("non-SSL haproxy config should have http frontend")
	}
}

func TestGenerateConfig_ValidTypes(t *testing.T) {
	types := []ProxyType{ProxyCaddy, ProxyTraefik, ProxyHAProxy}

	for _, pt := range types {
		t.Run(string(pt), func(t *testing.T) {
			result, err := GenerateConfig(pt, sslParams)
			if err != nil {
				t.Fatalf("GenerateConfig(%q) returned error: %v", pt, err)
			}
			if result == "" {
				t.Fatalf("GenerateConfig(%q) returned empty string", pt)
			}
			if !strings.Contains(result, "panel.example.com") {
				t.Errorf("GenerateConfig(%q) missing domain in output", pt)
			}
		})
	}
}

func TestGenerateConfig_InvalidType(t *testing.T) {
	_, err := GenerateConfig("apache", sslParams)
	if err == nil {
		t.Error("GenerateConfig with invalid type should return error")
	}
	if !strings.Contains(err.Error(), "unsupported proxy type") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// --- Additional coverage tests (task 11.6) ---

func TestAllGenerators_MinimalParams_NonEmpty(t *testing.T) {
	// Minimal params: domain only, no SSL, empty backend addr
	params := ProxyParams{
		Domain: "minimal.example.com",
	}

	generators := []struct {
		name string
		fn   func(ProxyParams) string
	}{
		{"Caddy", GenerateCaddy},
		{"Traefik", GenerateTraefik},
		{"HAProxy", GenerateHAProxy},
	}

	for _, g := range generators {
		t.Run(g.name, func(t *testing.T) {
			result := g.fn(params)
			if result == "" {
				t.Errorf("%s generator returned empty output with minimal params", g.name)
			}
			if !strings.Contains(result, "minimal.example.com") {
				t.Errorf("%s generator output missing domain 'minimal.example.com'", g.name)
			}
		})
	}
}

func TestAllGenerators_SSLDisabled_NoSSLDirectives(t *testing.T) {
	params := ProxyParams{
		Domain:      "nossl.example.com",
		BackendAddr: "127.0.0.1:9090",
		SSLEnabled:  false,
	}

	tests := []struct {
		name    string
		fn      func(ProxyParams) string
		absent  []string // patterns that must NOT appear
		present []string // patterns that MUST appear
	}{
		{
			name: "Caddy",
			fn:   GenerateCaddy,
			absent: []string{
				"tls ",
			},
			present: []string{"http://nossl.example.com", "reverse_proxy"},
		},
		{
			name: "Traefik",
			fn:   GenerateTraefik,
			absent: []string{
				"certFile:",
				"keyFile:",
				"websecure",
				"redirectScheme",
				"scheme: https",
			},
			present: []string{"nossl.example.com", "web"},
		},
		{
			name: "HAProxy",
			fn:   GenerateHAProxy,
			absent: []string{
				"bind *:443",
				"frontend korispanel_https",
				"redirect scheme https",
				"ssl crt",
			},
			present: []string{"nossl.example.com", "frontend korispanel_http", "bind *:80"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.fn(params)
			for _, pattern := range tc.absent {
				if strings.Contains(result, pattern) {
					t.Errorf("%s: SSL-disabled config should NOT contain %q", tc.name, pattern)
				}
			}
			for _, pattern := range tc.present {
				if !strings.Contains(result, pattern) {
					t.Errorf("%s: SSL-disabled config should contain %q", tc.name, pattern)
				}
			}
		})
	}
}

func TestAllGenerators_DomainAppearsInOutput(t *testing.T) {
	domains := []string{
		"panel.example.com",
		"my-vpn.io",
		"sub.domain.deep.example.org",
	}

	types := []ProxyType{ProxyCaddy, ProxyTraefik, ProxyHAProxy}

	for _, domain := range domains {
		for _, pt := range types {
			name := domain + "/" + string(pt)
			t.Run(name, func(t *testing.T) {
				params := ProxyParams{
					Domain:      domain,
					BackendAddr: "127.0.0.1:8080",
					SSLEnabled:  false,
				}
				result, err := GenerateConfig(pt, params)
				if err != nil {
					t.Fatalf("GenerateConfig(%s, %s) error: %v", pt, domain, err)
				}
				if !strings.Contains(result, domain) {
					t.Errorf("GenerateConfig(%s) output missing domain %q", pt, domain)
				}
			})
		}
	}
}

