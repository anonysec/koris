package api

import (
	"github.com/anonysec/koris/internal/safeexec"
	"net/http"
	"strings"
)

func (s *Server) diagnostics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	isActive := func(service string) bool {
		cmd := safeexec.MustCommand("systemctl", "is-active", service)
		out, err := cmd.Output()
		if err != nil {
			return false
		}
		return strings.TrimSpace(string(out)) == "active"
	}

	runCmd := func(name string, args ...string) string {
		cmd := safeexec.MustCommand(name, args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	}

	var checks []map[string]any

	checks = append(checks, map[string]any{
		"name":   "PostgreSQL service",
		"ok":     isActive("postgresql"),
		"detail": "systemctl is-active postgresql",
	})
	checks = append(checks, map[string]any{
		"name":   "Auth service",
		"ok":     isActive("freeradius"),
		"detail": "systemctl is-active freeradius",
	})
	checks = append(checks, map[string]any{
		"name":   "Panel service",
		"ok":     isActive("panel"),
		"detail": "systemctl is-active koris",
	})
	checks = append(checks, map[string]any{
		"name":   "OpenVPN service",
		"ok":     isActive("openvpn-server@server") || isActive("openvpn"),
		"detail": "systemctl is-active openvpn-server@server",
	})
	checks = append(checks, map[string]any{
		"name":   "Node agent",
		"ok":     isActive("knode"),
		"detail": "systemctl is-active knode",
	})
	checks = append(checks, map[string]any{
		"name":   "L2TP service",
		"ok":     isActive("xl2tpd"),
		"detail": "systemctl is-active xl2tpd",
	})
	checks = append(checks, map[string]any{
		"name":   "IKEv2 service",
		"ok":     isActive("strongswan") || isActive("strongswan-starter") || isActive("swanctl"),
		"detail": "strongswan service check",
	})

	disk := runCmd("df", "-h", "/")
	if disk == "" {
		disk = "N/A"
	}

	mem := runCmd("free", "-h")
	if mem == "" {
		mem = "N/A"
	}

	ports := runCmd("ss", "-ltnp")

	writeJSON(w, map[string]any{
		"ok":     true,
		"disk":   disk,
		"mem":    mem,
		"checks": checks,
		"ports":  ports,
	})
}
