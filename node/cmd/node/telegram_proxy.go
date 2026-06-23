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
	mtgBinaryPath  = "/usr/local/bin/mtg"
	mtgDownloadURL = "https://github.com/9seconds/mtg/releases/latest/download/mtg-linux-amd64"
	mtgServiceDir  = "/etc/systemd/system"
)

// mtgServiceName returns the systemd unit name for an mtg proxy on the given port.
func mtgServiceName(port int) string {
	return fmt.Sprintf("mtg-%d.service", port)
}

// mtgServicePath returns the full path to the systemd unit file.
func mtgServicePath(port int) string {
	return filepath.Join(mtgServiceDir, mtgServiceName(port))
}

// mtgServiceUnit generates the systemd unit file content for an mtg proxy.
func mtgServiceUnit(port int, secret string) string {
	return fmt.Sprintf(`[Unit]
Description=MTProto Proxy (port %d)
After=network.target

[Service]
Type=simple
ExecStart=%s run %s --bind 0.0.0.0:%d
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
`, port, mtgBinaryPath, secret, port)
}

// ensureMtgBinary downloads and installs the mtg binary if it doesn't exist.
func ensureMtgBinary(log *logger.Logger) error {
	if _, err := os.Stat(mtgBinaryPath); err == nil {
		log.Debug("mtg binary already present", map[string]any{"path": mtgBinaryPath})
		return nil
	}

	log.Info("downloading mtg binary", map[string]any{"url": mtgDownloadURL})

	// Determine correct download URL based on architecture
	downloadURL := mtgDownloadURL
	if runtime.GOARCH == "arm64" {
		downloadURL = "https://github.com/9seconds/mtg/releases/latest/download/mtg-linux-arm64"
	}

	// Download using curl
	cmd := exec.Command("curl", "-fsSL", "-o", mtgBinaryPath, downloadURL)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("download mtg: %s (%s)", err.Error(), strings.TrimSpace(string(out)))
	}

	// Make executable
	if err := os.Chmod(mtgBinaryPath, 0755); err != nil {
		return fmt.Errorf("chmod mtg: %s", err.Error())
	}

	log.Info("mtg binary installed", map[string]any{"path": mtgBinaryPath})
	return nil
}

// executeTelegramProxyDeploy handles the "telegram_proxy_deploy" task:
// installs mtg binary, creates systemd unit, starts and enables the service.
func executeTelegramProxyDeploy(payload map[string]any, log *logger.Logger) (string, map[string]any, string) {
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

	log.Info("telegram proxy deployed", map[string]any{"port": port, "service": svcName})
	return "succeeded", map[string]any{"port": port, "service": svcName, "status": "running"}, ""
}

// executeTelegramProxyStart handles the "telegram_proxy_start" task.
func executeTelegramProxyStart(payload map[string]any, log *logger.Logger) (string, map[string]any, string) {
	port := int(payloadInt(payload, "port"))
	if port <= 0 || port > 65535 {
		return "failed", map[string]any{}, "invalid port"
	}

	svcName := mtgServiceName(port)
	out, err := exec.Command("systemctl", "start", svcName).CombinedOutput()
	if err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("start %s: %s", svcName, err.Error())
	}

	log.Info("telegram proxy started", map[string]any{"port": port, "service": svcName})
	return "succeeded", map[string]any{"port": port, "service": svcName, "status": "running"}, ""
}

// executeTelegramProxyStop handles the "telegram_proxy_stop" task.
func executeTelegramProxyStop(payload map[string]any, log *logger.Logger) (string, map[string]any, string) {
	port := int(payloadInt(payload, "port"))
	if port <= 0 || port > 65535 {
		return "failed", map[string]any{}, "invalid port"
	}

	svcName := mtgServiceName(port)
	out, err := exec.Command("systemctl", "stop", svcName).CombinedOutput()
	if err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("stop %s: %s", svcName, err.Error())
	}

	log.Info("telegram proxy stopped", map[string]any{"port": port, "service": svcName})
	return "succeeded", map[string]any{"port": port, "service": svcName, "status": "stopped"}, ""
}

// executeTelegramProxyRemove handles the "telegram_proxy_remove" task:
// stops and disables the service, removes the unit file, optionally removes the mtg binary.
func executeTelegramProxyRemove(payload map[string]any, log *logger.Logger) (string, map[string]any, string) {
	port := int(payloadInt(payload, "port"))
	if port <= 0 || port > 65535 {
		return "failed", map[string]any{}, "invalid port"
	}

	svcName := mtgServiceName(port)

	// Stop the service (ignore error if already stopped)
	exec.Command("systemctl", "stop", svcName).CombinedOutput()

	// Disable the service (ignore error if not enabled)
	exec.Command("systemctl", "disable", svcName).CombinedOutput()

	// Remove the unit file
	unitPath := mtgServicePath(port)
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return "failed", map[string]any{}, fmt.Sprintf("remove unit file: %s", err.Error())
	}

	// Reload systemd daemon
	exec.Command("systemctl", "daemon-reload").CombinedOutput()

	// Check if any other mtg services exist; if not, remove the binary
	binaryRemoved := false
	matches, _ := filepath.Glob(filepath.Join(mtgServiceDir, "mtg-*.service"))
	if len(matches) == 0 {
		if err := os.Remove(mtgBinaryPath); err == nil {
			binaryRemoved = true
			log.Info("removed mtg binary (no remaining proxies)", map[string]any{"path": mtgBinaryPath})
		}
	}

	log.Info("telegram proxy removed", map[string]any{"port": port, "service": svcName, "binary_removed": binaryRemoved})
	return "succeeded", map[string]any{"port": port, "service": svcName, "removed": true, "binary_removed": binaryRemoved}, ""
}

// payloadStr extracts a string value from the payload map.
func payloadStr(payload map[string]any, key string) string {
	v, ok := payload[key]
	if !ok {
		return ""
	}
	s := fmt.Sprint(v)
	if s == "<nil>" {
		return ""
	}
	return s
}

// payloadInt extracts an integer value from the payload map.
func payloadInt(payload map[string]any, key string) int64 {
	v, ok := payload[key]
	if !ok {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int64(val)
	case int64:
		return val
	case int:
		return int64(val)
	default:
		return 0
	}
}
