package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"KorisPanel/node/internal/logger"
)

const (
	mtprotoStateFile = "/etc/panel-node/mtproto.state"
	mtprotoStateDir  = "/etc/panel-node"
)

// MTProtoState tracks the desired state of the mtg proxy for auto-restart.
type MTProtoState struct {
	Enabled bool   `json:"enabled"`
	Port    int    `json:"port"`
	Secret  string `json:"secret"`
}

// loadMTProtoState reads the local state file for auto-restart tracking.
func loadMTProtoState() (*MTProtoState, error) {
	data, err := os.ReadFile(mtprotoStateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &MTProtoState{}, nil
		}
		return nil, err
	}
	var state MTProtoState
	if err := json.Unmarshal(data, &state); err != nil {
		return &MTProtoState{}, nil
	}
	return &state, nil
}

// saveMTProtoState persists the desired state for auto-restart tracking.
func saveMTProtoState(state *MTProtoState) error {
	if err := os.MkdirAll(mtprotoStateDir, 0755); err != nil {
		return err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(mtprotoStateFile, data, 0600)
}

// removeMTProtoState removes the state file when the proxy is disabled.
func removeMTProtoState() {
	os.Remove(mtprotoStateFile)
}

// handleMTProtoEnable handles the "mtproto_enable" task action.
// It installs mtg if not present, writes the systemd service file,
// enables and starts the mtg service.
func handleMTProtoEnable(payload map[string]any, log *logger.Logger) (string, map[string]any, string) {
	port := int(payloadInt(payload, "port"))
	secret := payloadStr(payload, "secret")

	if port <= 0 || port > 65535 {
		return "failed", map[string]any{}, "invalid port"
	}
	if secret == "" {
		return "failed", map[string]any{}, "secret is required"
	}

	// Ensure mtg binary is installed
	if err := ensureMtgBinary(log); err != nil {
		return "failed", map[string]any{}, err.Error()
	}

	// Write systemd unit file
	unitContent := mtgServiceUnit(port, secret)
	unitPath := mtgServicePath(port)
	if err := os.WriteFile(unitPath, []byte(unitContent), 0644); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("write service unit: %s", err.Error())
	}

	// Reload systemd daemon
	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("daemon-reload: %s", err.Error())
	}

	// Enable the service
	svcName := mtgServiceName(port)
	if out, err := exec.Command("systemctl", "enable", svcName).CombinedOutput(); err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("enable service: %s", err.Error())
	}

	// Start the service
	if out, err := exec.Command("systemctl", "start", svcName).CombinedOutput(); err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("start service: %s", err.Error())
	}

	// Save state for auto-restart tracking
	state := &MTProtoState{
		Enabled: true,
		Port:    port,
		Secret:  secret,
	}
	if err := saveMTProtoState(state); err != nil {
		log.Warn("failed to save mtproto state", map[string]any{"error": err.Error()})
	}

	log.Info("[mtproto] enabled", map[string]any{"port": port})
	return "succeeded", map[string]any{"port": port, "service": svcName, "status": "running"}, ""
}

// handleMTProtoDisable handles the "mtproto_disable" task action.
// It stops and disables the mtg service, optionally removes the service file.
func handleMTProtoDisable(payload map[string]any, log *logger.Logger) (string, map[string]any, string) {
	// Load state to determine port
	state, _ := loadMTProtoState()
	port := int(payloadInt(payload, "port"))
	if port <= 0 && state != nil && state.Port > 0 {
		port = state.Port
	}
	if port <= 0 {
		// Try to find any running mtg service
		matches, _ := filepath.Glob(filepath.Join(mtgServiceDir, "mtg-*.service"))
		if len(matches) > 0 {
			// Extract port from first match filename: mtg-PORT.service
			base := filepath.Base(matches[0])
			base = strings.TrimPrefix(base, "mtg-")
			base = strings.TrimSuffix(base, ".service")
			if p, err := strconv.Atoi(base); err == nil {
				port = p
			}
		}
		if port <= 0 {
			return "failed", map[string]any{}, "unable to determine mtproto port"
		}
	}

	svcName := mtgServiceName(port)

	// Stop the service (ignore error if already stopped)
	exec.Command("systemctl", "stop", svcName).CombinedOutput()

	// Disable the service (ignore error if not enabled)
	exec.Command("systemctl", "disable", svcName).CombinedOutput()

	// Remove the unit file
	unitPath := mtgServicePath(port)
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		log.Warn("failed to remove unit file", map[string]any{"path": unitPath, "error": err.Error()})
	}

	// Reload systemd daemon
	exec.Command("systemctl", "daemon-reload").CombinedOutput()

	// Remove state file
	removeMTProtoState()

	// Check if any other mtg services exist; if not, remove the binary
	binaryRemoved := false
	remaining, _ := filepath.Glob(filepath.Join(mtgServiceDir, "mtg-*.service"))
	if len(remaining) == 0 {
		if err := os.Remove(mtgBinaryPath); err == nil {
			binaryRemoved = true
			log.Info("[mtproto] removed binary (no remaining proxies)", map[string]any{"path": mtgBinaryPath})
		}
	}

	log.Info("[mtproto] disabled", map[string]any{"port": port})
	return "succeeded", map[string]any{"port": port, "service": svcName, "disabled": true, "binary_removed": binaryRemoved}, ""
}

// handleMTProtoRotateSecret handles the "mtproto_rotate_secret" task action.
// It updates the mtg service with a new secret and restarts the service.
func handleMTProtoRotateSecret(payload map[string]any, log *logger.Logger) (string, map[string]any, string) {
	newSecret := payloadStr(payload, "new_secret")
	if newSecret == "" {
		// Also try "secret" key for compatibility
		newSecret = payloadStr(payload, "secret")
	}
	if newSecret == "" {
		return "failed", map[string]any{}, "new_secret is required"
	}

	// Load current state to get port
	state, _ := loadMTProtoState()
	port := state.Port
	if port <= 0 {
		// Try to find running mtg service
		matches, _ := filepath.Glob(filepath.Join(mtgServiceDir, "mtg-*.service"))
		if len(matches) > 0 {
			base := filepath.Base(matches[0])
			base = strings.TrimPrefix(base, "mtg-")
			base = strings.TrimSuffix(base, ".service")
			if p, err := strconv.Atoi(base); err == nil {
				port = p
			}
		}
		if port <= 0 {
			return "failed", map[string]any{}, "unable to determine mtproto port"
		}
	}

	// Rewrite the service unit file with new secret
	unitContent := mtgServiceUnit(port, newSecret)
	unitPath := mtgServicePath(port)
	if err := os.WriteFile(unitPath, []byte(unitContent), 0644); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("write service unit: %s", err.Error())
	}

	// Reload systemd daemon
	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("daemon-reload: %s", err.Error())
	}

	// Restart the service
	svcName := mtgServiceName(port)
	if out, err := exec.Command("systemctl", "restart", svcName).CombinedOutput(); err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("restart service: %s", err.Error())
	}

	// Update state file
	state.Secret = newSecret
	state.Port = port
	state.Enabled = true
	if err := saveMTProtoState(state); err != nil {
		log.Warn("failed to save mtproto state", map[string]any{"error": err.Error()})
	}

	log.Info("[mtproto] secret rotated", map[string]any{"port": port})
	return "succeeded", map[string]any{"port": port, "service": svcName, "rotated": true}, ""
}

// mtprotoStatus checks the current status of the mtg service.
// Returns "up", "down", or "not_installed".
func mtprotoStatus() string {
	state, err := loadMTProtoState()
	if err != nil || !state.Enabled || state.Port <= 0 {
		return "not_installed"
	}
	svcName := mtgServiceName(state.Port)
	out, err := exec.Command("systemctl", "is-active", svcName).Output()
	status := strings.TrimSpace(string(out))
	if err != nil || status != "active" {
		return "down"
	}
	return "up"
}

// mtprotoConnections attempts to get the connection count from mtg.
// If mtg exposes stats, uses them; otherwise returns 0.
func mtprotoConnections() int {
	// mtg does not expose a stats endpoint by default.
	// Try reading from /proc to count established connections on the proxy port.
	state, err := loadMTProtoState()
	if err != nil || !state.Enabled || state.Port <= 0 {
		return 0
	}

	// Count established TCP connections on the mtproto port
	portStr := strconv.Itoa(state.Port)
	out, err := exec.Command("ss", "-tn", "state", "established", "sport", "=", ":"+portStr).Output()
	if err != nil {
		return 0
	}
	// Count lines minus header
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) <= 1 {
		return 0
	}
	return len(lines) - 1
}

// mtprotoAutoRestart checks if mtg was supposed to be running but is dead,
// and attempts one restart. Returns the status after the check.
// Should be called from the metrics push loop.
func mtprotoAutoRestart(log *logger.Logger) string {
	state, err := loadMTProtoState()
	if err != nil || !state.Enabled || state.Port <= 0 {
		return "not_configured"
	}

	svcName := mtgServiceName(state.Port)
	out, _ := exec.Command("systemctl", "is-active", svcName).Output()
	status := strings.TrimSpace(string(out))

	if status == "active" {
		return "up"
	}

	// Service is supposed to be running but it's not — attempt one restart
	log.Warn("[mtproto] service down, attempting restart", map[string]any{
		"port":    state.Port,
		"service": svcName,
	})

	if restartOut, err := exec.Command("systemctl", "restart", svcName).CombinedOutput(); err != nil {
		log.Error("[mtproto] restart failed", map[string]any{
			"port":   state.Port,
			"error":  err.Error(),
			"output": string(restartOut),
		})
		return "crashed"
	}

	log.Info("[mtproto] restarted successfully", map[string]any{"port": state.Port})
	return "up"
}

// MTProtoMetrics holds the metrics for the mtproto proxy reported in the push payload.
type MTProtoMetrics struct {
	Status      string `json:"status"` // "up", "down", "crashed", "not_configured"
	Port        int    `json:"port"`
	Connections int    `json:"connections"`
}

// collectMTProtoMetrics gathers mtproto status and connection info for the metrics push.
func collectMTProtoMetrics(log *logger.Logger) *MTProtoMetrics {
	state, err := loadMTProtoState()
	if err != nil || !state.Enabled || state.Port <= 0 {
		return nil
	}

	status := mtprotoAutoRestart(log)
	connections := 0
	if status == "up" {
		connections = mtprotoConnections()
	}

	return &MTProtoMetrics{
		Status:      status,
		Port:        state.Port,
		Connections: connections,
	}
}
