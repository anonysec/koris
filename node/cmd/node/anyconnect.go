package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"KorisPanel/node/internal/logger"
)

const (
	ocservConfPath      = "/etc/ocserv/ocserv.conf"
	ocservCertPath      = "/etc/ocserv/server-cert.pem"
	ocservKeyPath       = "/etc/ocserv/server-key.pem"
	ocservServiceName   = "ocserv"
	anyconnectStateDir  = "/etc/panel-node"
	anyconnectStateFile = "/etc/panel-node/anyconnect.state"
)

// AnyConnectState tracks the desired state of ocserv for auto-restart.
type AnyConnectState struct {
	Enabled bool `json:"enabled"`
	Port    int  `json:"port"`
}

// AnyConnectSession represents an active AnyConnect session from occtl output.
type AnyConnectSession struct {
	Username string `json:"username"`
	RxBytes  int64  `json:"rx_bytes"`
	TxBytes  int64  `json:"tx_bytes"`
}

// AnyConnectMetrics holds the metrics for AnyConnect reported in the push payload.
type AnyConnectMetrics struct {
	Status   string              `json:"status"` // "up", "down", "crashed", "not_configured"
	Port     int                 `json:"port"`
	Sessions int                 `json:"sessions"`
	Users    []AnyConnectSession `json:"users,omitempty"`
}

// loadAnyConnectState reads the local state file for auto-restart tracking.
func loadAnyConnectState() (*AnyConnectState, error) {
	data, err := os.ReadFile(anyconnectStateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &AnyConnectState{}, nil
		}
		return nil, err
	}
	var state AnyConnectState
	if err := json.Unmarshal(data, &state); err != nil {
		return &AnyConnectState{}, nil
	}
	return &state, nil
}

// saveAnyConnectState persists the desired state for auto-restart tracking.
func saveAnyConnectState(state *AnyConnectState) error {
	if err := os.MkdirAll(anyconnectStateDir, 0755); err != nil {
		return err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(anyconnectStateFile, data, 0600)
}

// removeAnyConnectState removes the state file when AnyConnect is disabled.
func removeAnyConnectState() {
	os.Remove(anyconnectStateFile)
}

// generateOcservConfig produces a basic ocserv.conf with the specified port
// and FreeRADIUS authentication.
func generateOcservConfig(port int) string {
	var b strings.Builder
	b.WriteString("# ocserv configuration - managed by KorisPanel\n")
	b.WriteString(fmt.Sprintf("tcp-port = %d\n", port))
	b.WriteString(fmt.Sprintf("udp-port = %d\n", port))
	b.WriteString("\n")
	b.WriteString("# Authentication via RADIUS (FreeRADIUS)\n")
	b.WriteString("auth = \"radius[config=/etc/radcli/radiusclient.conf,groupconfig=true]\"\n")
	b.WriteString("acct = \"radius[config=/etc/radcli/radiusclient.conf]\"\n")
	b.WriteString("\n")
	b.WriteString("# TLS certificate\n")
	b.WriteString(fmt.Sprintf("server-cert = %s\n", ocservCertPath))
	b.WriteString(fmt.Sprintf("server-key = %s\n", ocservKeyPath))
	b.WriteString("\n")
	b.WriteString("# Network settings\n")
	b.WriteString("ipv4-network = 192.168.99.0\n")
	b.WriteString("ipv4-netmask = 255.255.255.0\n")
	b.WriteString("dns = 8.8.8.8\n")
	b.WriteString("dns = 8.8.4.4\n")
	b.WriteString("\n")
	b.WriteString("# Connection limits\n")
	b.WriteString("max-clients = 128\n")
	b.WriteString("max-same-clients = 2\n")
	b.WriteString("\n")
	b.WriteString("# Device and routing\n")
	b.WriteString("device = vpns\n")
	b.WriteString("default-domain = vpn.local\n")
	b.WriteString("try-mtu-discovery = true\n")
	b.WriteString("compression = true\n")
	b.WriteString("\n")
	b.WriteString("# Keepalive and timeouts\n")
	b.WriteString("keepalive = 32400\n")
	b.WriteString("dpd = 90\n")
	b.WriteString("mobile-dpd = 1800\n")
	b.WriteString("idle-timeout = 1200\n")
	b.WriteString("mobile-idle-timeout = 2400\n")
	return b.String()
}

// handleAnyConnectEnable handles the "anyconnect_enable" task action.
// It installs ocserv if not present, writes the config, enables and starts the service.
func handleAnyConnectEnable(payload map[string]any, log *logger.Logger) (string, map[string]any, string) {
	port := int(payloadInt(payload, "port"))
	if port <= 0 || port > 65535 {
		port = 443
	}

	// Check if ocserv is already installed
	if _, err := exec.LookPath("ocserv"); err != nil {
		// Install ocserv via apt-get
		log.Info("[anyconnect] installing ocserv", nil)
		cmd := exec.Command("apt-get", "install", "-y", "ocserv")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("install ocserv: %s", err.Error())
		}
		log.Info("[anyconnect] ocserv installed", nil)
	}

	// Ensure config directory exists
	if err := os.MkdirAll("/etc/ocserv", 0755); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("create config dir: %s", err.Error())
	}

	// Write ocserv configuration
	confContent := generateOcservConfig(port)
	if err := os.WriteFile(ocservConfPath, []byte(confContent), 0644); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("write ocserv.conf: %s", err.Error())
	}

	// Enable and start ocserv via systemd
	if out, err := exec.Command("systemctl", "enable", ocservServiceName).CombinedOutput(); err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("enable ocserv: %s", err.Error())
	}

	if out, err := exec.Command("systemctl", "start", ocservServiceName).CombinedOutput(); err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("start ocserv: %s", err.Error())
	}

	// Save state for auto-restart tracking
	state := &AnyConnectState{
		Enabled: true,
		Port:    port,
	}
	if err := saveAnyConnectState(state); err != nil {
		log.Warn("[anyconnect] failed to save state", map[string]any{"error": err.Error()})
	}

	log.Info("[anyconnect] enabled", map[string]any{"port": port})
	return "succeeded", map[string]any{
		"port":    port,
		"service": ocservServiceName,
		"status":  "running",
	}, ""
}

// handleAnyConnectDisable handles the "anyconnect_disable" task action.
// It stops and disables the ocserv service.
func handleAnyConnectDisable(payload map[string]any, log *logger.Logger) (string, map[string]any, string) {
	// Stop the service (ignore error if already stopped)
	exec.Command("systemctl", "stop", ocservServiceName).CombinedOutput()

	// Disable the service (ignore error if not enabled)
	exec.Command("systemctl", "disable", ocservServiceName).CombinedOutput()

	// Remove state file
	removeAnyConnectState()

	log.Info("[anyconnect] disabled", nil)
	return "succeeded", map[string]any{
		"service":  ocservServiceName,
		"disabled": true,
	}, ""
}

// handleAnyConnectCertUpdate handles the "anyconnect_cert_update" task action.
// It writes the TLS certificate and key files, then reloads ocserv.
func handleAnyConnectCertUpdate(payload map[string]any, log *logger.Logger) (string, map[string]any, string) {
	certPEM := payloadStr(payload, "cert_pem")
	keyPEM := payloadStr(payload, "key_pem")

	if certPEM == "" {
		return "failed", map[string]any{}, "cert_pem is required"
	}
	if keyPEM == "" {
		return "failed", map[string]any{}, "key_pem is required"
	}

	// Ensure config directory exists
	if err := os.MkdirAll("/etc/ocserv", 0755); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("create config dir: %s", err.Error())
	}

	// Write certificate file
	if err := os.WriteFile(ocservCertPath, []byte(certPEM), 0644); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("write cert: %s", err.Error())
	}

	// Write key file with restrictive permissions
	if err := os.WriteFile(ocservKeyPath, []byte(keyPEM), 0600); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("write key: %s", err.Error())
	}

	// Reload ocserv to pick up new certificate
	if out, err := exec.Command("systemctl", "reload", ocservServiceName).CombinedOutput(); err != nil {
		// If reload fails, try restart
		if out2, err2 := exec.Command("systemctl", "restart", ocservServiceName).CombinedOutput(); err2 != nil {
			return "failed", map[string]any{
				"reload_output":  string(out),
				"restart_output": string(out2),
			}, fmt.Sprintf("reload/restart ocserv: %s / %s", err.Error(), err2.Error())
		}
	}

	log.Info("[anyconnect] certificate updated", map[string]any{
		"cert_path": ocservCertPath,
		"key_path":  ocservKeyPath,
	})
	return "succeeded", map[string]any{
		"cert_path": ocservCertPath,
		"key_path":  ocservKeyPath,
		"reloaded":  true,
	}, ""
}

// collectAnyConnectSessions runs `occtl show users` to get active AnyConnect sessions.
// It parses the output to count sessions and extract per-user traffic if available.
func collectAnyConnectSessions() []AnyConnectSession {
	out, err := exec.Command("occtl", "show", "users").CombinedOutput()
	if err != nil {
		return nil
	}

	output := strings.TrimSpace(string(out))
	if output == "" {
		return nil
	}

	// Parse occtl output. Format is typically a table:
	// id   user   vhost   ip   ... rx   tx   ...
	// Lines after the header contain user data.
	lines := strings.Split(output, "\n")
	if len(lines) <= 1 {
		return nil
	}

	var sessions []AnyConnectSession
	// Find column positions from the header line
	header := lines[0]
	userCol := strings.Index(header, "user")
	rxCol := strings.Index(header, "RX")
	txCol := strings.Index(header, "TX")

	// If we can't find standard columns, try simple whitespace parsing
	if userCol < 0 {
		// Fallback: parse each line as whitespace-separated fields
		for _, line := range lines[1:] {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			// Assume: id username [other fields]
			session := AnyConnectSession{
				Username: fields[1],
			}
			// Try to find numeric fields for rx/tx at end of line
			for i := len(fields) - 1; i >= 2 && i > len(fields)-3; i-- {
				if val, err := strconv.ParseInt(fields[i], 10, 64); err == nil {
					if session.TxBytes == 0 {
						session.TxBytes = val
					} else {
						session.RxBytes = val
					}
				}
			}
			sessions = append(sessions, session)
		}
		return sessions
	}

	// Parse with known column positions
	for _, line := range lines[1:] {
		if len(line) <= userCol {
			continue
		}
		// Extract username
		var username string
		if rxCol > userCol {
			endIdx := rxCol
			if endIdx > len(line) {
				endIdx = len(line)
			}
			username = strings.TrimSpace(line[userCol:endIdx])
		} else {
			fields := strings.Fields(line[userCol:])
			if len(fields) > 0 {
				username = fields[0]
			}
		}
		if username == "" {
			continue
		}

		session := AnyConnectSession{Username: username}

		// Extract RX bytes if column exists
		if rxCol >= 0 && rxCol < len(line) {
			rxFields := strings.Fields(line[rxCol:])
			if len(rxFields) > 0 {
				if val, err := strconv.ParseInt(rxFields[0], 10, 64); err == nil {
					session.RxBytes = val
				}
			}
		}

		// Extract TX bytes if column exists
		if txCol >= 0 && txCol < len(line) {
			txFields := strings.Fields(line[txCol:])
			if len(txFields) > 0 {
				if val, err := strconv.ParseInt(txFields[0], 10, 64); err == nil {
					session.TxBytes = val
				}
			}
		}

		sessions = append(sessions, session)
	}

	return sessions
}

// anyconnectAutoRestart checks if ocserv was supposed to be running but is dead,
// and attempts one restart. Returns the status after the check.
func anyconnectAutoRestart(log *logger.Logger) string {
	state, err := loadAnyConnectState()
	if err != nil || !state.Enabled || state.Port <= 0 {
		return "not_configured"
	}

	out, _ := exec.Command("systemctl", "is-active", ocservServiceName).Output()
	status := strings.TrimSpace(string(out))

	if status == "active" {
		return "up"
	}

	// Service is supposed to be running but it's not — attempt one restart
	log.Warn("[anyconnect] service down, attempting restart", map[string]any{
		"port":    state.Port,
		"service": ocservServiceName,
	})

	if restartOut, err := exec.Command("systemctl", "restart", ocservServiceName).CombinedOutput(); err != nil {
		log.Error("[anyconnect] restart failed", map[string]any{
			"port":   state.Port,
			"error":  err.Error(),
			"output": string(restartOut),
		})
		return "crashed"
	}

	log.Info("[anyconnect] restarted successfully", map[string]any{"port": state.Port})
	return "up"
}

// collectAnyConnectMetrics gathers AnyConnect status and session info for the metrics push.
func collectAnyConnectMetrics(log *logger.Logger) *AnyConnectMetrics {
	state, err := loadAnyConnectState()
	if err != nil || !state.Enabled || state.Port <= 0 {
		return nil
	}

	status := anyconnectAutoRestart(log)
	var sessions []AnyConnectSession
	if status == "up" {
		sessions = collectAnyConnectSessions()
	}

	return &AnyConnectMetrics{
		Status:   status,
		Port:     state.Port,
		Sessions: len(sessions),
		Users:    sessions,
	}
}
