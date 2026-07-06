package proxyconfig

import (
	"fmt"
	"strings"
)

// ProxyType identifies a supported reverse proxy.
type ProxyType string

const (
	ProxyCaddy   ProxyType = "caddy"
	ProxyTraefik ProxyType = "traefik"
	ProxyHAProxy ProxyType = "haproxy"
)

// ProxyParams holds all parameters needed to generate a reverse proxy config.
type ProxyParams struct {
	Domain      string // e.g. "panel.example.com"
	BackendAddr string // e.g. "127.0.0.1:8080"
	SSLCertPath string // e.g. "/etc/letsencrypt/live/panel.example.com/fullchain.pem"
	SSLKeyPath  string // e.g. "/etc/letsencrypt/live/panel.example.com/privkey.pem"
	SSLEnabled  bool
}

// GenerateConfig dispatches to the correct generator based on proxyType.
func GenerateConfig(proxyType ProxyType, params ProxyParams) (string, error) {
	switch proxyType {
	case ProxyCaddy:
		return GenerateCaddy(params), nil
	case ProxyTraefik:
		return GenerateTraefik(params), nil
	case ProxyHAProxy:
		return GenerateHAProxy(params), nil
	default:
		return "", fmt.Errorf("unsupported proxy type: %q", proxyType)
	}
}

// GenerateCaddy generates a complete Caddyfile configuration.
func GenerateCaddy(params ProxyParams) string {
	var b strings.Builder

	b.WriteString("# KorisPanel Caddy Configuration\n")
	b.WriteString(fmt.Sprintf("# Generated for: %s\n\n", params.Domain))

	if params.SSLEnabled {
		b.WriteString(fmt.Sprintf("%s {\n", params.Domain))
		b.WriteString(fmt.Sprintf("    tls %s %s\n\n", params.SSLCertPath, params.SSLKeyPath))
	} else {
		b.WriteString(fmt.Sprintf("http://%s {\n", params.Domain))
		b.WriteString("\n")
	}

	// Health check
	b.WriteString("    # Health check endpoint\n")
	b.WriteString("    handle /api/health {\n")
	b.WriteString(fmt.Sprintf("        reverse_proxy %s\n", params.BackendAddr))
	b.WriteString("    }\n\n")

	// WebSocket for /api/ws/*
	b.WriteString("    # WebSocket support\n")
	b.WriteString("    handle /api/ws/* {\n")
	b.WriteString(fmt.Sprintf("        reverse_proxy %s\n", params.BackendAddr))
	b.WriteString("    }\n\n")

	// Default reverse proxy
	b.WriteString("    # Reverse proxy to panel backend\n")
	b.WriteString("    handle {\n")
	b.WriteString(fmt.Sprintf("        reverse_proxy %s {\n", params.BackendAddr))
	b.WriteString("            header_up X-Real-IP {remote_host}\n")
	b.WriteString("            header_up X-Forwarded-For {remote_host}\n")
	b.WriteString("            header_up X-Forwarded-Proto {scheme}\n")
	b.WriteString("        }\n")
	b.WriteString("    }\n")
	b.WriteString("}\n")

	return b.String()
}

// GenerateTraefik generates a Traefik dynamic file provider YAML configuration.
func GenerateTraefik(params ProxyParams) string {
	var b strings.Builder

	b.WriteString("# KorisPanel Traefik Configuration (Dynamic File Provider)\n")
	b.WriteString(fmt.Sprintf("# Generated for: %s\n", params.Domain))
	b.WriteString("# Place this file in your Traefik dynamic config directory\n\n")

	// HTTP routers
	b.WriteString("http:\n")
	b.WriteString("  routers:\n")

	if params.SSLEnabled {
		// HTTP → HTTPS redirect router
		b.WriteString("    korispanel-http:\n")
		b.WriteString(fmt.Sprintf("      rule: \"Host(`%s`)\"\n", params.Domain))
		b.WriteString("      entryPoints:\n")
		b.WriteString("        - web\n")
		b.WriteString("      middlewares:\n")
		b.WriteString("        - korispanel-https-redirect\n")
		b.WriteString("      service: korispanel\n\n")

		// HTTPS router
		b.WriteString("    korispanel-https:\n")
		b.WriteString(fmt.Sprintf("      rule: \"Host(`%s`)\"\n", params.Domain))
		b.WriteString("      entryPoints:\n")
		b.WriteString("        - websecure\n")
		b.WriteString("      tls:\n")
		b.WriteString("        certificates:\n")
		b.WriteString(fmt.Sprintf("          - certFile: %s\n", params.SSLCertPath))
		b.WriteString(fmt.Sprintf("            keyFile: %s\n", params.SSLKeyPath))
		b.WriteString("      service: korispanel\n\n")
	} else {
		b.WriteString("    korispanel:\n")
		b.WriteString(fmt.Sprintf("      rule: \"Host(`%s`)\"\n", params.Domain))
		b.WriteString("      entryPoints:\n")
		b.WriteString("        - web\n")
		b.WriteString("      service: korispanel\n\n")
	}

	// Middlewares
	if params.SSLEnabled {
		b.WriteString("  middlewares:\n")
		b.WriteString("    # HTTP to HTTPS redirect\n")
		b.WriteString("    korispanel-https-redirect:\n")
		b.WriteString("      redirectScheme:\n")
		b.WriteString("        scheme: https\n")
		b.WriteString("        permanent: true\n\n")
	}

	// Services
	b.WriteString("  services:\n")
	b.WriteString("    # Panel backend service\n")
	b.WriteString("    korispanel:\n")
	b.WriteString("      loadBalancer:\n")
	b.WriteString("        servers:\n")
	b.WriteString(fmt.Sprintf("          - url: \"http://%s\"\n", params.BackendAddr))
	b.WriteString("        passHostHeader: true\n")
	b.WriteString("        # Health check\n")
	b.WriteString("        healthCheck:\n")
	b.WriteString("          path: /api/health\n")
	b.WriteString("          interval: 10s\n")
	b.WriteString("          timeout: 3s\n")

	return b.String()
}

// GenerateHAProxy generates HAProxy frontend and backend configuration sections.
func GenerateHAProxy(params ProxyParams) string {
	var b strings.Builder

	b.WriteString("# KorisPanel HAProxy Configuration\n")
	b.WriteString(fmt.Sprintf("# Generated for: %s\n", params.Domain))
	b.WriteString("# Add these sections to your haproxy.cfg\n\n")

	if params.SSLEnabled {
		// HTTP frontend — redirect to HTTPS
		b.WriteString("# HTTP to HTTPS redirect\n")
		b.WriteString("frontend korispanel_http\n")
		b.WriteString("    bind *:80\n")
		b.WriteString(fmt.Sprintf("    acl is_panel hdr(host) -i %s\n", params.Domain))
		b.WriteString("    http-request redirect scheme https if is_panel\n\n")

		// HTTPS frontend
		b.WriteString("# HTTPS frontend with TLS termination\n")
		b.WriteString("frontend korispanel_https\n")
		b.WriteString(fmt.Sprintf("    bind *:443 ssl crt %s\n", params.SSLCertPath))
		b.WriteString(fmt.Sprintf("    acl is_panel hdr(host) -i %s\n", params.Domain))
		b.WriteString("    acl is_websocket path_beg /api/ws/\n\n")
		b.WriteString("    # Forward headers\n")
		b.WriteString("    http-request set-header X-Real-IP %%[src]\n")
		b.WriteString("    http-request set-header X-Forwarded-For %%[src]\n")
		b.WriteString("    http-request set-header X-Forwarded-Proto https\n\n")
		b.WriteString("    use_backend korispanel_ws if is_websocket\n")
		b.WriteString("    default_backend korispanel_backend\n\n")
	} else {
		b.WriteString("# HTTP frontend\n")
		b.WriteString("frontend korispanel_http\n")
		b.WriteString("    bind *:80\n")
		b.WriteString(fmt.Sprintf("    acl is_panel hdr(host) -i %s\n", params.Domain))
		b.WriteString("    acl is_websocket path_beg /api/ws/\n\n")
		b.WriteString("    # Forward headers\n")
		b.WriteString("    http-request set-header X-Real-IP %%[src]\n")
		b.WriteString("    http-request set-header X-Forwarded-For %%[src]\n")
		b.WriteString("    http-request set-header X-Forwarded-Proto http\n\n")
		b.WriteString("    use_backend korispanel_ws if is_websocket\n")
		b.WriteString("    default_backend korispanel_backend\n\n")
	}

	// WebSocket backend
	b.WriteString("# WebSocket backend with long timeout\n")
	b.WriteString("backend korispanel_ws\n")
	b.WriteString(fmt.Sprintf("    server panel %s check\n", params.BackendAddr))
	b.WriteString("    timeout server 86400s\n")
	b.WriteString("    timeout tunnel 86400s\n\n")

	// Main backend
	b.WriteString("# Panel backend\n")
	b.WriteString("backend korispanel_backend\n")
	b.WriteString(fmt.Sprintf("    server panel %s check\n", params.BackendAddr))
	b.WriteString("    # Health check\n")
	b.WriteString("    option httpchk GET /api/health\n")
	b.WriteString("    http-check expect status 200\n")

	return b.String()
}
