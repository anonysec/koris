package proxyconfig

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// NginxInfo holds information about the detected nginx installation.
type NginxInfo struct {
	Installed  bool
	Version    string
	ConfigPath string
	SitesPath  string
}

// DetectNginx runs nginx -v to detect if nginx is installed and gathers
// version, config path, and sites directory information.
func DetectNginx() NginxInfo {
	info := NginxInfo{}

	cmd := exec.Command("nginx", "-v")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return info
	}

	info.Installed = true
	info.Version = DetectNginxVersion(string(output))
	info.ConfigPath = FindNginxConfigPath()
	info.SitesPath = FindNginxSitesDir()

	return info
}

// DetectNginxVersion parses the nginx version from the output of `nginx -v`.
// Expected format: "nginx version: nginx/1.24.0"
func DetectNginxVersion(output string) string {
	re := regexp.MustCompile(`nginx/(\d+\.\d+\.\d+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

// IsHTTP2Supported returns true if the nginx version supports HTTP/2.
// HTTP/2 was introduced in nginx 1.9.5.
func IsHTTP2Supported(version string) bool {
	return compareVersions(version, "1.9.5") >= 0
}

// IsHTTP3Supported returns true if the nginx version supports QUIC/HTTP3.
// HTTP/3 support was introduced in nginx 1.25.0.
func IsHTTP3Supported(version string) bool {
	return compareVersions(version, "1.25.0") >= 0
}

// FindNginxConfigPath checks common locations for the nginx configuration file
// and returns the first one found.
func FindNginxConfigPath() string {
	paths := []string{
		"/etc/nginx/nginx.conf",
		"/usr/local/nginx/conf/nginx.conf",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// FindNginxSitesDir checks common locations for the nginx sites directory
// and returns the first one found.
func FindNginxSitesDir() string {
	dirs := []string{
		"/etc/nginx/sites-enabled/",
		"/etc/nginx/conf.d/",
	}
	for _, d := range dirs {
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			return d
		}
	}
	return ""
}

// GenerateCompatibleNginx generates an nginx config compatible with the detected
// nginx version. For nginx >= 1.25.0, it uses the modern `http2 on;` directive.
// For older versions that support HTTP/2 (>= 1.9.5), it uses the deprecated
// `listen ... http2` directive. For versions that don't support HTTP/2, no HTTP/2
// directives are included.
func GenerateCompatibleNginx(params ProxyParams, info NginxInfo) string {
	var b strings.Builder

	b.WriteString("# KorisPanel Nginx Configuration\n")
	b.WriteString(fmt.Sprintf("# Generated for: %s\n", params.Domain))
	if info.Installed && info.Version != "" {
		b.WriteString(fmt.Sprintf("# Detected nginx version: %s\n", info.Version))
	}
	b.WriteString("\n")

	http2Supported := IsHTTP2Supported(info.Version)
	// nginx 1.25.1+ deprecated the listen directive form of http2 in favor of `http2 on;`
	useModernHTTP2 := compareVersions(info.Version, "1.25.0") >= 0

	if params.SSLEnabled {
		// HTTP → HTTPS redirect block
		b.WriteString("server {\n")
		b.WriteString("    listen 80;\n")
		b.WriteString(fmt.Sprintf("    server_name %s;\n", params.Domain))
		b.WriteString("    return 301 https://$server_name$request_uri;\n")
		b.WriteString("}\n\n")

		// HTTPS server block
		b.WriteString("server {\n")
		if http2Supported && !useModernHTTP2 {
			// Legacy directive: listen 443 ssl http2;
			b.WriteString("    listen 443 ssl http2;\n")
		} else {
			b.WriteString("    listen 443 ssl;\n")
		}
		b.WriteString(fmt.Sprintf("    server_name %s;\n\n", params.Domain))

		if useModernHTTP2 {
			// Modern directive: http2 on;
			b.WriteString("    http2 on;\n\n")
		}

		b.WriteString(fmt.Sprintf("    ssl_certificate %s;\n", params.SSLCertPath))
		b.WriteString(fmt.Sprintf("    ssl_certificate_key %s;\n\n", params.SSLKeyPath))
		b.WriteString("    # SSL hardening\n")
		b.WriteString("    ssl_protocols TLSv1.2 TLSv1.3;\n")
		b.WriteString("    ssl_ciphers HIGH:!aNULL:!MD5;\n")
		b.WriteString("    ssl_prefer_server_ciphers on;\n\n")
	} else {
		b.WriteString("server {\n")
		b.WriteString("    listen 80;\n")
		b.WriteString(fmt.Sprintf("    server_name %s;\n\n", params.Domain))
	}

	// Health check passthrough
	b.WriteString("    # Health check endpoint\n")
	b.WriteString("    location /api/health {\n")
	b.WriteString(fmt.Sprintf("        proxy_pass http://%s;\n", params.BackendAddr))
	b.WriteString("        proxy_set_header Host $host;\n")
	b.WriteString("    }\n\n")

	// WebSocket location for /api/ws/
	b.WriteString("    # WebSocket support\n")
	b.WriteString("    location /api/ws/ {\n")
	b.WriteString(fmt.Sprintf("        proxy_pass http://%s;\n", params.BackendAddr))
	b.WriteString("        proxy_http_version 1.1;\n")
	b.WriteString("        proxy_set_header Upgrade $http_upgrade;\n")
	b.WriteString("        proxy_set_header Connection \"upgrade\";\n")
	b.WriteString("        proxy_set_header Host $host;\n")
	b.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
	b.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
	b.WriteString("        proxy_set_header X-Forwarded-Proto $scheme;\n")
	b.WriteString("        proxy_read_timeout 86400s;\n")
	b.WriteString("    }\n\n")

	// Default proxy pass
	b.WriteString("    # Reverse proxy to panel backend\n")
	b.WriteString("    location / {\n")
	b.WriteString(fmt.Sprintf("        proxy_pass http://%s;\n", params.BackendAddr))
	b.WriteString("        proxy_set_header Host $host;\n")
	b.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
	b.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
	b.WriteString("        proxy_set_header X-Forwarded-Proto $scheme;\n")
	b.WriteString("    }\n")
	b.WriteString("}\n")

	return b.String()
}

// compareVersions compares two semver-style version strings (e.g. "1.24.0").
// Returns: -1 if a < b, 0 if a == b, 1 if a > b.
// Returns -1 if either version string is empty or malformed.
func compareVersions(a, b string) int {
	if a == "" || b == "" {
		return -1
	}

	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	// Pad to 3 parts
	for len(aParts) < 3 {
		aParts = append(aParts, "0")
	}
	for len(bParts) < 3 {
		bParts = append(bParts, "0")
	}

	for i := 0; i < 3; i++ {
		aNum, errA := strconv.Atoi(aParts[i])
		bNum, errB := strconv.Atoi(bParts[i])
		if errA != nil || errB != nil {
			return -1
		}
		if aNum < bNum {
			return -1
		}
		if aNum > bNum {
			return 1
		}
	}
	return 0
}
