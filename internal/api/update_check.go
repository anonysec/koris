//go:build !lite

package api

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// updateCheck handles GET /api/settings/update-check
func (s *Server) updateCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	currentVersion := os.Getenv("KORIS_VERSION")
	if currentVersion == "" {
		currentVersion = "dev"
	}

	writeJSON(w, map[string]any{
		"ok":               true,
		"current_version":  currentVersion,
		"go_version":       runtime.Version(),
		"os":               runtime.GOOS,
		"arch":             runtime.GOARCH,
		"uptime":           getUptime(),
		"latest_version":   currentVersion, // In production, fetch from release API
		"update_available": false,
	})
}

func getUptime() string {
	out, err := exec.Command("cat", "/proc/uptime").Output()
	if err != nil {
		return "unknown"
	}
	parts := strings.Fields(string(out))
	if len(parts) < 1 {
		return "unknown"
	}
	var seconds float64
	if _, err := fmt.Sscanf(parts[0], "%f", &seconds); err != nil {
		return "unknown"
	}
	d := time.Duration(seconds) * time.Second
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, mins)
}
