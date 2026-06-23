package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"KorisPanel/node/internal/logger"
)

const (
	obfs4proxyBinaryPath = "/usr/local/bin/obfs4proxy"
	obfs4proxyServiceDir = "/etc/systemd/system"
)

// obfs4ServiceName returns the systemd unit name for obfs4proxy on the given port.
func obfs4ServiceName(port int) string {
	return fmt.Sprintf("obfs4proxy-%d.service", port)
}

// obfs4ServicePath returns the full path to the systemd unit file.
func obfs4ServicePath(port int) string {
	return filepath.Join(obfs4proxyServiceDir, obfs4ServiceName(port))
}

// obfs4ServiceUnit generates the systemd unit file content for obfs4proxy.
func obfs4ServiceUnit(port int) string {
	return fmt.Sprintf(`[Unit]
Description=obfs4proxy obfuscation layer (port %d)
After=network.target

[Service]
Type=simple
ExecStart=%s -enableLogging -logLevel INFO
Environment=TOR_PT_MANAGED_TRANSPORT_VER=1
Environment=TOR_PT_STATE_LOCATION=/var/lib/obfs4proxy-%d
Environment=TOR_PT_SERVER_TRANSPORTS=obfs4
Environment=TOR_PT_SERVER_BINDADDR=obfs4-0.0.0.0:%d
Environment=TOR_PT_ORPORT=127.0.0.1:9001
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`, port, obfs4proxyBinaryPath, port, port)
}

// quicTunnelServiceName returns the systemd unit name for QUIC tunneling.
func quicTunnelServiceName(port int) string {
	return fmt.Sprintf("quic-tunnel-%d.service", port)
}

// quicTunnelServicePath returns the full path to the QUIC tunnel systemd unit file.
func quicTunnelServicePath(port int) string {
	return filepath.Join(obfs4proxyServiceDir, quicTunnelServiceName(port))
}

// ensureObfs4Binary downloads and installs the obfs4proxy binary if not present.
func ensureObfs4Binary(log *logger.Logger) error {
	if _, err := os.Stat(obfs4proxyBinaryPath); err == nil {
		log.Debug("obfs4proxy binary already present", map[string]any{"path": obfs4proxyBinaryPath})
		return nil
	}

	log.Info("installing obfs4proxy via apt", map[string]any{})

	// Try apt-get install first (available in Debian/Ubuntu repos)
	cmd := exec.Command("apt-get", "install", "-y", "obfs4proxy")
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback: download binary directly
		downloadURL := "https://github.com/AnonySec/obfs4proxy-linux/releases/latest/download/obfs4proxy-linux-amd64"
		if runtime.GOARCH == "arm64" {
			downloadURL = "https://github.com/AnonySec/obfs4proxy-linux/releases/latest/download/obfs4proxy-linux-arm64"
		}
		log.Info("apt install failed, downloading obfs4proxy binary", map[string]any{
			"url":    downloadURL,
			"output": strings.TrimSpace(string(out)),
		})

		dlCmd := exec.Command("curl", "-fsSL", "-o", obfs4proxyBinaryPath, downloadURL)
		dlOut, dlErr := dlCmd.CombinedOutput()
		if dlErr != nil {
			return fmt.Errorf("download obfs4proxy: %s (%s)", dlErr.Error(), strings.TrimSpace(string(dlOut)))
		}

		if err := os.Chmod(obfs4proxyBinaryPath, 0755); err != nil {
			return fmt.Errorf("chmod obfs4proxy: %s", err.Error())
		}
	}

	log.Info("obfs4proxy installed", map[string]any{"path": obfs4proxyBinaryPath})
	return nil
}

// executeAntiDPIDeploy handles the "anti_dpi_deploy" task:
// installs obfs4proxy or sets up QUIC tunneling based on the method configured.
func executeAntiDPIDeploy(payload map[string]any, log *logger.Logger) (string, map[string]any, string) {
	method := payloadStr(payload, "method")
	port := int(payloadInt(payload, "port"))

	if port <= 0 || port > 65535 {
		return "failed", map[string]any{}, "invalid port"
	}

	switch method {
	case "obfs4":
		return deployObfs4(port, log)
	case "quic":
		return deployQUICTunnel(port, payload, log)
	case "ws_tunnel":
		return deployWSTunnel(port, payload, log)
	case "none":
		return "succeeded", map[string]any{"message": "no obfuscation method configured"}, ""
	default:
		return "failed", map[string]any{}, fmt.Sprintf("unsupported anti-dpi method: %s", method)
	}
}

// deployObfs4 installs obfs4proxy and creates a systemd service.
func deployObfs4(port int, log *logger.Logger) (string, map[string]any, string) {
	if err := ensureObfs4Binary(log); err != nil {
		return "failed", map[string]any{}, err.Error()
	}

	// Create state directory
	stateDir := fmt.Sprintf("/var/lib/obfs4proxy-%d", port)
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("create state dir: %s", err.Error())
	}

	// Write systemd unit file
	unitContent := obfs4ServiceUnit(port)
	unitPath := obfs4ServicePath(port)
	if err := os.WriteFile(unitPath, []byte(unitContent), 0644); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("write service unit: %s", err.Error())
	}

	// Reload systemd daemon
	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("daemon-reload: %s", err.Error())
	}

	// Enable and start the service
	svcName := obfs4ServiceName(port)
	if out, err := exec.Command("systemctl", "enable", svcName).CombinedOutput(); err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("enable service: %s", err.Error())
	}

	if out, err := exec.Command("systemctl", "start", svcName).CombinedOutput(); err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("start service: %s", err.Error())
	}

	log.Info("anti-dpi obfs4 deployed", map[string]any{"port": port, "service": svcName})
	return "succeeded", map[string]any{"method": "obfs4", "port": port, "service": svcName, "status": "running"}, ""
}

// deployQUICTunnel sets up QUIC-based traffic obfuscation.
func deployQUICTunnel(port int, payload map[string]any, log *logger.Logger) (string, map[string]any, string) {
	// QUIC tunneling wraps VPN traffic in QUIC packets to disguise as normal web traffic.
	// This creates a configuration for the QUIC tunnel service.
	svcName := quicTunnelServiceName(port)
	unitPath := quicTunnelServicePath(port)

	bridgeAddress := payloadStr(payload, "bridge_address")

	unitContent := fmt.Sprintf(`[Unit]
Description=QUIC tunnel obfuscation (port %d)
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/quic-tunnel -listen 0.0.0.0:%d -target %s -mode server
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`, port, port, bridgeAddress)

	if err := os.WriteFile(unitPath, []byte(unitContent), 0644); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("write quic service unit: %s", err.Error())
	}

	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("daemon-reload: %s", err.Error())
	}

	if out, err := exec.Command("systemctl", "enable", svcName).CombinedOutput(); err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("enable service: %s", err.Error())
	}

	if out, err := exec.Command("systemctl", "start", svcName).CombinedOutput(); err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("start service: %s", err.Error())
	}

	log.Info("anti-dpi QUIC tunnel deployed", map[string]any{"port": port, "service": svcName})
	return "succeeded", map[string]any{"method": "quic", "port": port, "service": svcName, "status": "running"}, ""
}

// deployWSTunnel sets up WebSocket-based traffic obfuscation.
func deployWSTunnel(port int, payload map[string]any, log *logger.Logger) (string, map[string]any, string) {
	svcName := fmt.Sprintf("ws-tunnel-%d.service", port)
	unitPath := filepath.Join(obfs4proxyServiceDir, svcName)

	bridgeAddress := payloadStr(payload, "bridge_address")

	unitContent := fmt.Sprintf(`[Unit]
Description=WebSocket tunnel obfuscation (port %d)
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/wstunnel -listen 0.0.0.0:%d -target %s
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`, port, port, bridgeAddress)

	if err := os.WriteFile(unitPath, []byte(unitContent), 0644); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("write ws-tunnel service unit: %s", err.Error())
	}

	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("daemon-reload: %s", err.Error())
	}

	if out, err := exec.Command("systemctl", "enable", svcName).CombinedOutput(); err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("enable service: %s", err.Error())
	}

	if out, err := exec.Command("systemctl", "start", svcName).CombinedOutput(); err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("start service: %s", err.Error())
	}

	log.Info("anti-dpi WS tunnel deployed", map[string]any{"port": port, "service": svcName})
	return "succeeded", map[string]any{"method": "ws_tunnel", "port": port, "service": svcName, "status": "running"}, ""
}

// executeAntiDPIRemove handles the "anti_dpi_remove" task:
// stops and removes the obfuscation service for the given method.
func executeAntiDPIRemove(payload map[string]any, log *logger.Logger) (string, map[string]any, string) {
	method := payloadStr(payload, "method")
	port := int(payloadInt(payload, "port"))

	if port <= 0 || port > 65535 {
		// If no port provided, try to clean up all known anti-dpi services
		log.Warn("anti_dpi_remove: invalid or missing port, skipping", map[string]any{"method": method})
		return "succeeded", map[string]any{"message": "no port specified, nothing to remove"}, ""
	}

	var svcName string
	var unitPath string
	var binaryPath string

	switch method {
	case "obfs4":
		svcName = obfs4ServiceName(port)
		unitPath = obfs4ServicePath(port)
		binaryPath = obfs4proxyBinaryPath
	case "quic":
		svcName = quicTunnelServiceName(port)
		unitPath = quicTunnelServicePath(port)
	case "ws_tunnel":
		svcName = fmt.Sprintf("ws-tunnel-%d.service", port)
		unitPath = filepath.Join(obfs4proxyServiceDir, svcName)
	default:
		return "succeeded", map[string]any{"message": "unknown method, nothing to remove"}, ""
	}

	// Stop the service (ignore error if already stopped)
	exec.Command("systemctl", "stop", svcName).CombinedOutput()

	// Disable the service
	exec.Command("systemctl", "disable", svcName).CombinedOutput()

	// Remove the unit file
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return "failed", map[string]any{}, fmt.Sprintf("remove unit file: %s", err.Error())
	}

	// Reload systemd daemon
	exec.Command("systemctl", "daemon-reload").CombinedOutput()

	// For obfs4: remove state directory and binary if no other instances
	if method == "obfs4" {
		stateDir := fmt.Sprintf("/var/lib/obfs4proxy-%d", port)
		os.RemoveAll(stateDir)

		// Check if any other obfs4proxy services exist
		matches, _ := filepath.Glob(filepath.Join(obfs4proxyServiceDir, "obfs4proxy-*.service"))
		if len(matches) == 0 && binaryPath != "" {
			if err := os.Remove(binaryPath); err == nil {
				log.Info("removed obfs4proxy binary (no remaining instances)", map[string]any{"path": binaryPath})
			}
		}
	}

	log.Info("anti-dpi removed", map[string]any{"method": method, "port": port, "service": svcName})
	return "succeeded", map[string]any{"method": method, "port": port, "service": svcName, "removed": true}, ""
}
