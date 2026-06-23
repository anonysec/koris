package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"KorisPanel/node/internal/logger"
)

// Known core engines that we report status for in the metrics push.
var knownCores = []string{"xray", "sing-box"}

// CoreMetrics holds per-core status information reported in metrics push.
type CoreMetrics struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "running", "stopped", "failed", "not_installed"
}

// coreServiceTemplate generates a systemd service unit file for a VPN core engine.
func coreServiceTemplate(coreName string) string {
	// Use title-cased core name for description
	displayName := coreName
	switch coreName {
	case "xray":
		displayName = "Xray"
	case "sing-box":
		displayName = "Sing-Box"
	}

	return fmt.Sprintf(`[Unit]
Description=%s - VPN Core Engine
After=network.target

[Service]
ExecStart=/usr/local/bin/%s run -confdir /etc/%s/conf.d
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`, displayName, coreName, coreName)
}

// coreInstallPath returns the binary install path for a core.
func coreInstallPath(coreName string) string {
	return fmt.Sprintf("/usr/local/bin/%s", coreName)
}

// coreServiceFilePath returns the systemd service file path for a core.
func coreServiceFilePath(coreName string) string {
	return fmt.Sprintf("/etc/systemd/system/%s.service", coreName)
}

// downloadAndVerify downloads a binary from the given URL and verifies its SHA-256 checksum.
// Returns the binary data on success, or an error if download or checksum verification fails.
func downloadAndVerify(downloadURL, checksumSHA256 string, log *logger.Logger) ([]byte, error) {
	log.Info("downloading core binary", map[string]any{
		"url": downloadURL,
	})

	resp, err := http.Get(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("download binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read binary body: %w", err)
	}

	// Verify SHA-256 checksum
	h := sha256.Sum256(data)
	actualHash := hex.EncodeToString(h[:])
	if !strings.EqualFold(actualHash, strings.TrimSpace(checksumSHA256)) {
		return nil, fmt.Errorf("checksum mismatch: expected %s, got %s", checksumSHA256, actualHash)
	}

	return data, nil
}

// handleCoreInstall handles the "core_install" task action:
// downloads the core binary, verifies its checksum, installs it, creates a systemd service,
// and starts the core.
func handleCoreInstall(payload map[string]any, log *logger.Logger) (string, map[string]any, string) {
	coreName := payloadStr(payload, "core_name")
	version := payloadStr(payload, "version")
	downloadURL := payloadStr(payload, "download_url")
	checksumSHA256 := payloadStr(payload, "checksum_sha256")

	if coreName == "" {
		return "failed", map[string]any{}, "core_name is required"
	}
	if downloadURL == "" {
		return "failed", map[string]any{}, "download_url is required"
	}
	if checksumSHA256 == "" {
		return "failed", map[string]any{}, "checksum_sha256 is required"
	}

	// Download and verify checksum
	data, err := downloadAndVerify(downloadURL, checksumSHA256, log)
	if err != nil {
		return "failed", map[string]any{}, err.Error()
	}

	// Write binary to install path
	installPath := coreInstallPath(coreName)
	if err := os.MkdirAll("/usr/local/bin", 0755); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("create bin dir: %s", err.Error())
	}
	if err := os.WriteFile(installPath, data, 0755); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("write binary: %s", err.Error())
	}

	// Create config directory for the core
	confDir := fmt.Sprintf("/etc/%s/conf.d", coreName)
	if err := os.MkdirAll(confDir, 0755); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("create config dir: %s", err.Error())
	}

	// Create systemd service file
	servicePath := coreServiceFilePath(coreName)
	serviceContent := coreServiceTemplate(coreName)
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("write service file: %s", err.Error())
	}

	// Reload systemd, enable and start the service
	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("daemon-reload: %s", err.Error())
	}

	serviceName := coreName + ".service"
	if out, err := exec.Command("systemctl", "enable", serviceName).CombinedOutput(); err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("enable service: %s", err.Error())
	}

	if out, err := exec.Command("systemctl", "start", serviceName).CombinedOutput(); err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("start service: %s", err.Error())
	}

	log.Info("core installed", map[string]any{
		"core_name":    coreName,
		"version":      version,
		"install_path": installPath,
		"service":      serviceName,
	})

	return "succeeded", map[string]any{
		"core_name":    coreName,
		"version":      version,
		"install_path": installPath,
		"service":      serviceName,
		"status":       "running",
	}, ""
}

// handleCoreUpdate handles the "core_update" task action:
// downloads the new binary, verifies checksum, stops the core, replaces the binary,
// and restarts the service.
func handleCoreUpdate(payload map[string]any, log *logger.Logger) (string, map[string]any, string) {
	coreName := payloadStr(payload, "core_name")
	version := payloadStr(payload, "version")
	downloadURL := payloadStr(payload, "download_url")
	checksumSHA256 := payloadStr(payload, "checksum_sha256")

	if coreName == "" {
		return "failed", map[string]any{}, "core_name is required"
	}
	if downloadURL == "" {
		return "failed", map[string]any{}, "download_url is required"
	}
	if checksumSHA256 == "" {
		return "failed", map[string]any{}, "checksum_sha256 is required"
	}

	// Download and verify checksum
	data, err := downloadAndVerify(downloadURL, checksumSHA256, log)
	if err != nil {
		return "failed", map[string]any{}, err.Error()
	}

	serviceName := coreName + ".service"
	installPath := coreInstallPath(coreName)

	// Stop the core service
	if out, err := exec.Command("systemctl", "stop", serviceName).CombinedOutput(); err != nil {
		log.Warn("failed to stop core before update", map[string]any{
			"core_name": coreName,
			"output":    strings.TrimSpace(string(out)),
			"error":     err.Error(),
		})
		// Continue anyway — the binary replacement should still work
	}

	// Replace the binary
	if err := os.WriteFile(installPath, data, 0755); err != nil {
		// Try to restart the old binary
		exec.Command("systemctl", "start", serviceName).CombinedOutput()
		return "failed", map[string]any{}, fmt.Sprintf("write binary: %s", err.Error())
	}

	// Start the core service
	if out, err := exec.Command("systemctl", "start", serviceName).CombinedOutput(); err != nil {
		errMsg := fmt.Sprintf("start service after update: %s (%s)", err.Error(), strings.TrimSpace(string(out)))
		log.Error("core failed to start after update", map[string]any{
			"core_name": coreName,
			"version":   version,
			"output":    strings.TrimSpace(string(out)),
			"error":     err.Error(),
		})
		return "failed", map[string]any{
			"core_name": coreName,
			"version":   version,
			"output":    strings.TrimSpace(string(out)),
		}, errMsg
	}

	log.Info("core updated", map[string]any{
		"core_name":    coreName,
		"version":      version,
		"install_path": installPath,
	})

	return "succeeded", map[string]any{
		"core_name":    coreName,
		"version":      version,
		"install_path": installPath,
		"status":       "running",
	}, ""
}

// handleCoreRemove handles the "core_remove" task action:
// stops and disables the service, removes the binary and service file,
// and runs daemon-reload.
func handleCoreRemove(payload map[string]any, log *logger.Logger) (string, map[string]any, string) {
	coreName := payloadStr(payload, "core_name")
	if coreName == "" {
		return "failed", map[string]any{}, "core_name is required"
	}

	serviceName := coreName + ".service"
	installPath := coreInstallPath(coreName)
	servicePath := coreServiceFilePath(coreName)

	// Stop and disable the service
	exec.Command("systemctl", "stop", serviceName).CombinedOutput()
	exec.Command("systemctl", "disable", serviceName).CombinedOutput()

	// Remove binary
	if err := os.Remove(installPath); err != nil && !os.IsNotExist(err) {
		log.Warn("failed to remove core binary", map[string]any{
			"core_name": coreName,
			"path":      installPath,
			"error":     err.Error(),
		})
	}

	// Remove service file
	if err := os.Remove(servicePath); err != nil && !os.IsNotExist(err) {
		log.Warn("failed to remove core service file", map[string]any{
			"core_name": coreName,
			"path":      servicePath,
			"error":     err.Error(),
		})
	}

	// Reload systemd daemon
	exec.Command("systemctl", "daemon-reload").CombinedOutput()

	log.Info("core removed", map[string]any{
		"core_name": coreName,
	})

	return "succeeded", map[string]any{
		"core_name": coreName,
		"removed":   true,
	}, ""
}

// collectCoreMetrics checks the status of each known core and returns their metrics.
// Used in the metrics push to report core health to the panel.
func collectCoreMetrics() []CoreMetrics {
	var metrics []CoreMetrics

	for _, coreName := range knownCores {
		installPath := coreInstallPath(coreName)

		// Check if the binary is installed
		if _, err := os.Stat(installPath); err != nil {
			// Core is not installed — skip it
			continue
		}

		// Check service status via systemctl is-active
		serviceName := coreName + ".service"
		out, err := exec.Command("systemctl", "is-active", serviceName).CombinedOutput()
		status := "stopped"
		if err == nil {
			switch strings.TrimSpace(string(out)) {
			case "active":
				status = "running"
			case "failed":
				status = "failed"
			default:
				status = "stopped"
			}
		} else {
			// If systemctl returns non-zero, check if it's "failed"
			outStr := strings.TrimSpace(string(out))
			if outStr == "failed" {
				status = "failed"
			}
		}

		metrics = append(metrics, CoreMetrics{
			Name:   coreName,
			Status: status,
		})
	}

	return metrics
}
