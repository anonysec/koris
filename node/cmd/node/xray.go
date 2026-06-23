package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"KorisPanel/node/internal/logger"
)

const (
	xrayBinaryPath     = "/usr/local/bin/xray"
	xrayConfigDir      = "/usr/local/etc/xray"
	xrayConfigPath     = "/usr/local/etc/xray/config.json"
	xrayServicePath    = "/etc/systemd/system/xray.service"
	xrayServiceName    = "xray.service"
	xrayDefaultVersion = "latest"
	xrayDownloadBase   = "https://github.com/XTLS/Xray-core/releases"
)

// xrayServiceUnit generates the systemd unit file content for the Xray service.
func xrayServiceUnit() string {
	return `[Unit]
Description=Xray Service - managed by KorisPanel
Documentation=https://xtls.github.io
After=network.target nss-lookup.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/xray run -config /usr/local/etc/xray/config.json
Restart=on-failure
RestartSec=5
LimitNOFILE=65535
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_ADMIN CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
`
}

// xrayDownloadURL constructs the download URL for a given Xray version.
func xrayDownloadURL(version string) string {
	arch := "64"
	if runtime.GOARCH == "arm64" {
		arch = "arm64-v8a"
	}
	if version == "" || version == xrayDefaultVersion {
		return fmt.Sprintf("%s/latest/download/Xray-linux-%s.zip", xrayDownloadBase, arch)
	}
	return fmt.Sprintf("%s/download/v%s/Xray-linux-%s.zip", xrayDownloadBase, version, arch)
}

// executeXrayDeploy handles the "xray_deploy" task:
// downloads and installs the Xray binary, writes the config JSON,
// creates the systemd service unit, enables and starts the service.
func executeXrayDeploy(payload map[string]any, log *logger.Logger) (string, map[string]any, string) {
	configJSON := payloadStr(payload, "config_json")
	version := payloadStr(payload, "version")
	downloadURL := payloadStr(payload, "download_url")

	if configJSON == "" {
		return "failed", map[string]any{}, "config_json is required"
	}

	// Validate that config_json is valid JSON
	if !json.Valid([]byte(configJSON)) {
		return "failed", map[string]any{}, "config_json is not valid JSON"
	}

	// Determine download URL
	if downloadURL == "" {
		downloadURL = xrayDownloadURL(version)
	}

	// Install xray binary
	if err := installXrayBinary(downloadURL, log); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("install xray: %s", err.Error())
	}

	// Ensure config directory exists
	if err := os.MkdirAll(xrayConfigDir, 0755); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("create config dir: %s", err.Error())
	}

	// Write xray config JSON
	if err := os.WriteFile(xrayConfigPath, []byte(configJSON), 0644); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("write config: %s", err.Error())
	}

	// Write systemd service unit
	if err := os.WriteFile(xrayServicePath, []byte(xrayServiceUnit()), 0644); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("write service unit: %s", err.Error())
	}

	// Reload systemd daemon
	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("daemon-reload: %s", err.Error())
	}

	// Enable the service
	if out, err := exec.Command("systemctl", "enable", xrayServiceName).CombinedOutput(); err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("enable service: %s", err.Error())
	}

	// Start (or restart if already running) the service
	if out, err := exec.Command("systemctl", "restart", xrayServiceName).CombinedOutput(); err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("start service: %s", err.Error())
	}

	// Get installed version
	installedVersion := getXrayVersion()

	log.Info("xray deployed", map[string]any{
		"version":     installedVersion,
		"config_path": xrayConfigPath,
		"service":     xrayServiceName,
	})
	return "succeeded", map[string]any{
		"version":      installedVersion,
		"config_path":  xrayConfigPath,
		"service":      xrayServiceName,
		"service_unit": xrayServicePath,
		"status":       "running",
	}, ""
}

// installXrayBinary downloads and installs the Xray binary from the given URL.
// It downloads a zip archive, extracts the xray binary, and places it at xrayBinaryPath.
func installXrayBinary(downloadURL string, log *logger.Logger) error {
	log.Info("downloading xray", map[string]any{"url": downloadURL})

	// Create a temp directory for download and extraction
	tmpDir, err := os.MkdirTemp("", "xray-install-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	zipPath := filepath.Join(tmpDir, "xray.zip")

	// Download using curl
	cmd := exec.Command("curl", "-fsSL", "-o", zipPath, downloadURL)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("download: %s (%s)", err.Error(), strings.TrimSpace(string(out)))
	}

	// Extract using unzip
	cmd = exec.Command("unzip", "-o", "-q", zipPath, "-d", tmpDir)
	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("extract: %s (%s)", err.Error(), strings.TrimSpace(string(out)))
	}

	// Move xray binary to target path
	extractedBinary := filepath.Join(tmpDir, "xray")
	if _, err := os.Stat(extractedBinary); err != nil {
		return fmt.Errorf("xray binary not found in archive")
	}

	// Ensure target directory exists
	if err := os.MkdirAll(filepath.Dir(xrayBinaryPath), 0755); err != nil {
		return fmt.Errorf("create bin dir: %w", err)
	}

	// Copy binary to target location (move across filesystems)
	input, err := os.ReadFile(extractedBinary)
	if err != nil {
		return fmt.Errorf("read extracted binary: %w", err)
	}
	if err := os.WriteFile(xrayBinaryPath, input, 0755); err != nil {
		return fmt.Errorf("write binary: %w", err)
	}

	log.Info("xray binary installed", map[string]any{"path": xrayBinaryPath})
	return nil
}

// getXrayVersion runs `xray version` and returns the version string.
func getXrayVersion() string {
	out, err := exec.Command(xrayBinaryPath, "version").CombinedOutput()
	if err != nil {
		return "unknown"
	}
	// First line typically: "Xray 1.8.4 (Xray, Penetrates Everything.) ..."
	lines := strings.SplitN(string(out), "\n", 2)
	if len(lines) > 0 {
		parts := strings.Fields(lines[0])
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	return strings.TrimSpace(string(out))
}

// executeXraySyncUsers handles the "xray_sync_users" task:
// updates the inbound users list in the Xray config for the specified protocols,
// then restarts the xray service to apply changes.
func executeXraySyncUsers(payload map[string]any, log *logger.Logger) (string, map[string]any, string) {
	// Parse users from payload
	usersRaw, ok := payload["users"]
	if !ok {
		return "failed", map[string]any{}, "users field is required"
	}
	usersSlice, ok := usersRaw.([]any)
	if !ok {
		return "failed", map[string]any{}, "users must be an array"
	}

	// Parse protocols from payload
	protocolsRaw, ok := payload["protocols"]
	if !ok {
		return "failed", map[string]any{}, "protocols field is required"
	}
	protocolsSlice, ok := protocolsRaw.([]any)
	if !ok || len(protocolsSlice) == 0 {
		return "failed", map[string]any{}, "protocols must be a non-empty array"
	}

	protocols := make(map[string]bool)
	for _, p := range protocolsSlice {
		protocols[strings.ToLower(fmt.Sprint(p))] = true
	}

	// Parse user objects
	type xrayUser struct {
		UUID  string
		Email string
		Level int
	}
	var users []xrayUser
	for _, u := range usersSlice {
		uMap, ok := u.(map[string]any)
		if !ok {
			continue
		}
		uuid := fmt.Sprint(uMap["uuid"])
		email := fmt.Sprint(uMap["email"])
		level := 0
		if lv, exists := uMap["level"]; exists {
			switch v := lv.(type) {
			case float64:
				level = int(v)
			case int:
				level = v
			}
		}
		if uuid == "" || uuid == "<nil>" {
			continue
		}
		if email == "<nil>" {
			email = ""
		}
		users = append(users, xrayUser{UUID: uuid, Email: email, Level: level})
	}

	if len(users) == 0 {
		return "failed", map[string]any{}, "no valid users provided"
	}

	// Read existing Xray config
	configData, err := os.ReadFile(xrayConfigPath)
	if err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("read xray config: %s", err.Error())
	}

	var config map[string]any
	if err := json.Unmarshal(configData, &config); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("parse xray config: %s", err.Error())
	}

	// Get inbounds array
	inboundsRaw, ok := config["inbounds"]
	if !ok {
		return "failed", map[string]any{}, "xray config has no inbounds"
	}
	inbounds, ok := inboundsRaw.([]any)
	if !ok {
		return "failed", map[string]any{}, "inbounds is not an array"
	}

	// Track which protocols were updated
	updatedProtocols := []string{}

	for i, inboundRaw := range inbounds {
		inbound, ok := inboundRaw.(map[string]any)
		if !ok {
			continue
		}

		protocol := strings.ToLower(fmt.Sprint(inbound["protocol"]))
		if !protocols[protocol] {
			continue
		}

		// Get or create settings object
		settings, ok := inbound["settings"].(map[string]any)
		if !ok {
			settings = map[string]any{}
		}

		// Build clients array based on protocol type
		var clients []map[string]any

		switch protocol {
		case "vless":
			for _, u := range users {
				clients = append(clients, map[string]any{
					"id":    u.UUID,
					"email": u.Email,
					"level": u.Level,
					"flow":  "",
				})
			}
			settings["clients"] = clients

		case "vmess":
			for _, u := range users {
				clients = append(clients, map[string]any{
					"id":      u.UUID,
					"email":   u.Email,
					"level":   u.Level,
					"alterId": 0,
				})
			}
			settings["clients"] = clients

		case "trojan":
			for _, u := range users {
				clients = append(clients, map[string]any{
					"password": u.UUID,
					"email":    u.Email,
					"level":    u.Level,
				})
			}
			settings["clients"] = clients

		case "shadowsocks":
			// Shadowsocks multi-user format: each user is in clients array with password
			for _, u := range users {
				clients = append(clients, map[string]any{
					"password": u.UUID,
					"email":    u.Email,
					"level":    u.Level,
				})
			}
			settings["clients"] = clients

		default:
			continue
		}

		inbound["settings"] = settings
		inbounds[i] = inbound
		updatedProtocols = append(updatedProtocols, protocol)
	}

	if len(updatedProtocols) == 0 {
		return "failed", map[string]any{}, "no matching inbounds found for specified protocols"
	}

	config["inbounds"] = inbounds

	// Marshal updated config with indentation for readability
	updatedData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("marshal config: %s", err.Error())
	}

	// Write back the updated config
	if err := os.WriteFile(xrayConfigPath, updatedData, 0644); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("write xray config: %s", err.Error())
	}

	// Restart xray service to apply changes
	if out, err := exec.Command("systemctl", "restart", xrayServiceName).CombinedOutput(); err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("restart xray: %s", err.Error())
	}

	log.Info("xray users synced", map[string]any{
		"users_count":       len(users),
		"protocols_updated": updatedProtocols,
	})

	return "succeeded", map[string]any{
		"users_synced":      len(users),
		"protocols_updated": updatedProtocols,
	}, ""
}

const (
	xrayStatsServer = "127.0.0.1:10085"

	// xrayConfDir is the directory for per-inbound config fragments (xray-core).
	xrayConfDir = "/etc/xray/conf.d"

	// singBoxConfDir is the directory for per-inbound config fragments (sing-box).
	singBoxConfDir = "/etc/sing-box/conf.d"
)

// confDirForCore returns the appropriate config fragment directory based on core_name.
func confDirForCore(coreName string) string {
	if coreName == "sing-box" {
		return singBoxConfDir
	}
	return xrayConfDir
}

// reloadXrayCore sends a SIGHUP to xray to reload configuration fragments,
// or falls back to systemctl reload if pkill -HUP is unavailable.
func reloadXrayCore(coreName string, log *logger.Logger) error {
	var serviceName string
	var processName string

	if coreName == "sing-box" {
		serviceName = "sing-box"
		processName = "sing-box"
	} else {
		serviceName = "xray"
		processName = "xray"
	}

	// Try SIGHUP via pkill first (graceful reload without full restart)
	cmd := exec.Command("pkill", "-HUP", processName)
	if out, err := cmd.CombinedOutput(); err != nil {
		// Fallback to systemctl reload
		log.Debug("pkill -HUP failed, trying systemctl reload", map[string]any{
			"process": processName,
			"output":  strings.TrimSpace(string(out)),
			"error":   err.Error(),
		})
		reloadCmd := exec.Command("systemctl", "reload", serviceName)
		if out2, err2 := reloadCmd.CombinedOutput(); err2 != nil {
			// Final fallback: systemctl restart
			restartCmd := exec.Command("systemctl", "restart", serviceName)
			if out3, err3 := restartCmd.CombinedOutput(); err3 != nil {
				return fmt.Errorf("reload %s: pkill failed (%s), systemctl reload failed (%s), restart failed (%s)",
					coreName, strings.TrimSpace(string(out)),
					strings.TrimSpace(string(out2)),
					strings.TrimSpace(string(out3)))
			}
		}
	}
	return nil
}

// handleXrayAdd handles the "xray_add" task action:
// writes a config fragment for the given UUID and reloads the core.
func handleXrayAdd(payload map[string]any, log *logger.Logger) (string, map[string]any, string) {
	uuid := payloadStr(payload, "uuid")
	if uuid == "" {
		return "failed", map[string]any{}, "uuid is required"
	}

	coreName := payloadStr(payload, "core_name")
	if coreName == "" {
		coreName = "xray-core"
	}

	// Extract config_fragment — can be a JSON string or an object
	configFragment, err := extractConfigFragment(payload)
	if err != nil {
		return "failed", map[string]any{}, err.Error()
	}

	confDir := confDirForCore(coreName)

	// Ensure config directory exists
	if err := os.MkdirAll(confDir, 0755); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("create conf dir: %s", err.Error())
	}

	// Write config fragment to {uuid}.json
	confPath := filepath.Join(confDir, uuid+".json")
	if err := os.WriteFile(confPath, configFragment, 0644); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("write config fragment: %s", err.Error())
	}

	// Reload core to pick up new config
	if err := reloadXrayCore(coreName, log); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("reload core: %s", err.Error())
	}

	log.Info("xray inbound added", map[string]any{
		"uuid":      uuid,
		"core_name": coreName,
		"conf_path": confPath,
	})
	return "succeeded", map[string]any{
		"uuid":      uuid,
		"conf_path": confPath,
		"core_name": coreName,
	}, ""
}

// handleXrayRemove handles the "xray_remove" task action:
// removes the config fragment for the given UUID and reloads the core.
func handleXrayRemove(payload map[string]any, log *logger.Logger) (string, map[string]any, string) {
	uuid := payloadStr(payload, "uuid")
	if uuid == "" {
		return "failed", map[string]any{}, "uuid is required"
	}

	coreName := payloadStr(payload, "core_name")
	if coreName == "" {
		coreName = "xray-core"
	}

	confDir := confDirForCore(coreName)
	confPath := filepath.Join(confDir, uuid+".json")

	// Remove config file (ignore if not found)
	if err := os.Remove(confPath); err != nil && !os.IsNotExist(err) {
		return "failed", map[string]any{}, fmt.Sprintf("remove config fragment: %s", err.Error())
	}

	// Reload core
	if err := reloadXrayCore(coreName, log); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("reload core: %s", err.Error())
	}

	log.Info("xray inbound removed", map[string]any{
		"uuid":      uuid,
		"core_name": coreName,
		"conf_path": confPath,
	})
	return "succeeded", map[string]any{
		"uuid":    uuid,
		"removed": true,
	}, ""
}

// handleXrayUpdate handles the "xray_update" task action:
// overwrites the config fragment for the given UUID and reloads the core.
func handleXrayUpdate(payload map[string]any, log *logger.Logger) (string, map[string]any, string) {
	uuid := payloadStr(payload, "uuid")
	if uuid == "" {
		return "failed", map[string]any{}, "uuid is required"
	}

	coreName := payloadStr(payload, "core_name")
	if coreName == "" {
		coreName = "xray-core"
	}

	// Extract config_fragment
	configFragment, err := extractConfigFragment(payload)
	if err != nil {
		return "failed", map[string]any{}, err.Error()
	}

	confDir := confDirForCore(coreName)

	// Ensure config directory exists
	if err := os.MkdirAll(confDir, 0755); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("create conf dir: %s", err.Error())
	}

	// Overwrite config fragment
	confPath := filepath.Join(confDir, uuid+".json")
	if err := os.WriteFile(confPath, configFragment, 0644); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("write config fragment: %s", err.Error())
	}

	// Reload core
	if err := reloadXrayCore(coreName, log); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("reload core: %s", err.Error())
	}

	log.Info("xray inbound updated", map[string]any{
		"uuid":      uuid,
		"core_name": coreName,
		"conf_path": confPath,
	})
	return "succeeded", map[string]any{
		"uuid":      uuid,
		"conf_path": confPath,
		"core_name": coreName,
	}, ""
}

// extractConfigFragment parses the config_fragment from the payload.
// It accepts either a JSON string or an object/map.
func extractConfigFragment(payload map[string]any) ([]byte, error) {
	raw, ok := payload["config_fragment"]
	if !ok {
		return nil, fmt.Errorf("config_fragment is required")
	}

	switch v := raw.(type) {
	case string:
		// Validate that the string is valid JSON
		if !json.Valid([]byte(v)) {
			return nil, fmt.Errorf("config_fragment is not valid JSON")
		}
		return []byte(v), nil
	case map[string]any:
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal config_fragment: %s", err.Error())
		}
		return data, nil
	default:
		// Try to marshal whatever it is
		data, err := json.MarshalIndent(raw, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("config_fragment must be a JSON string or object")
		}
		return data, nil
	}
}

// xrayStatsQueryResponse represents the JSON output from `xray api statsquery`.
type xrayStatsQueryResponse struct {
	Stat []xrayStatEntry `json:"stat"`
}

// xrayStatEntry represents a single stat entry from Xray's stats API.
type xrayStatEntry struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// isXrayRunning checks if the xray binary exists and the xray service is active.
func isXrayRunning() bool {
	// Check if xray binary exists
	if _, err := os.Stat(xrayBinaryPath); err != nil {
		return false
	}
	// Check if xray service is active
	out, err := exec.Command("systemctl", "is-active", xrayServiceName).CombinedOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "active"
}

// collectXrayStats queries Xray's stats API and returns per-user traffic statistics.
// It runs `xray api statsquery --server=127.0.0.1:10085` and parses the JSON output
// to aggregate uplink/downlink bytes per user email.
func collectXrayStats() []XrayUserStat {
	cmd := exec.Command(xrayBinaryPath, "api", "statsquery", "--server="+xrayStatsServer)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}

	output := strings.TrimSpace(string(out))
	if output == "" || output == "{}" {
		return nil
	}

	var resp xrayStatsQueryResponse
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		return nil
	}

	if len(resp.Stat) == 0 {
		return nil
	}

	// Aggregate uplink/downlink per user email
	// Stat name format: "user>>>email@example.com>>>traffic>>>uplink"
	type trafficAcc struct {
		Uplink   int64
		Downlink int64
	}
	users := make(map[string]*trafficAcc)

	for _, entry := range resp.Stat {
		parts := strings.Split(entry.Name, ">>>")
		if len(parts) != 4 {
			continue
		}
		if parts[0] != "user" || parts[2] != "traffic" {
			continue
		}

		email := parts[1]
		direction := parts[3]
		value, err := strconv.ParseInt(entry.Value, 10, 64)
		if err != nil {
			continue
		}

		acc, ok := users[email]
		if !ok {
			acc = &trafficAcc{}
			users[email] = acc
		}

		switch direction {
		case "uplink":
			acc.Uplink += value
		case "downlink":
			acc.Downlink += value
		}
	}

	// Convert map to slice
	result := make([]XrayUserStat, 0, len(users))
	for email, acc := range users {
		if acc.Uplink == 0 && acc.Downlink == 0 {
			continue
		}
		result = append(result, XrayUserStat{
			Email:    email,
			Uplink:   acc.Uplink,
			Downlink: acc.Downlink,
		})
	}

	return result
}

// resetXrayStats resets the Xray stats counters after collection so that
// subsequent collections return delta values rather than cumulative totals.
func resetXrayStats() {
	exec.Command(xrayBinaryPath, "api", "stats", "--server="+xrayStatsServer, "-reset").Run()
}
