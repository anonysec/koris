package main

import (
	"bytes"
	"strings"
	"testing"

	"KorisPanel/node/internal/logger"
)

// testLogger creates a logger instance for testing that discards output.
func testLogger() *logger.Logger {
	return logger.NewWithWriter(logger.LevelDebug, &bytes.Buffer{})
}

func TestCoreServiceTemplate(t *testing.T) {
	tests := []struct {
		name        string
		coreName    string
		wantExec    string
		wantConfDir string
		wantDesc    string
	}{
		{
			name:        "xray core",
			coreName:    "xray",
			wantExec:    "ExecStart=/usr/local/bin/xray run -confdir /etc/xray/conf.d",
			wantConfDir: "/etc/xray/conf.d",
			wantDesc:    "Xray - VPN Core Engine",
		},
		{
			name:        "sing-box core",
			coreName:    "sing-box",
			wantExec:    "ExecStart=/usr/local/bin/sing-box run -confdir /etc/sing-box/conf.d",
			wantConfDir: "/etc/sing-box/conf.d",
			wantDesc:    "Sing-Box - VPN Core Engine",
		},
		{
			name:        "custom core",
			coreName:    "hysteria",
			wantExec:    "ExecStart=/usr/local/bin/hysteria run -confdir /etc/hysteria/conf.d",
			wantConfDir: "/etc/hysteria/conf.d",
			wantDesc:    "hysteria - VPN Core Engine",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := coreServiceTemplate(tt.coreName)

			if !strings.Contains(result, tt.wantExec) {
				t.Errorf("coreServiceTemplate(%q) missing ExecStart line: want %q in output", tt.coreName, tt.wantExec)
			}
			if !strings.Contains(result, tt.wantDesc) {
				t.Errorf("coreServiceTemplate(%q) missing description: want %q in output", tt.coreName, tt.wantDesc)
			}
			if !strings.Contains(result, "Restart=on-failure") {
				t.Errorf("coreServiceTemplate(%q) missing Restart=on-failure", tt.coreName)
			}
			if !strings.Contains(result, "RestartSec=5") {
				t.Errorf("coreServiceTemplate(%q) missing RestartSec=5", tt.coreName)
			}
			if !strings.Contains(result, "[Unit]") || !strings.Contains(result, "[Service]") || !strings.Contains(result, "[Install]") {
				t.Errorf("coreServiceTemplate(%q) missing required systemd sections", tt.coreName)
			}
			if !strings.Contains(result, "WantedBy=multi-user.target") {
				t.Errorf("coreServiceTemplate(%q) missing WantedBy=multi-user.target", tt.coreName)
			}
			if !strings.Contains(result, "After=network.target") {
				t.Errorf("coreServiceTemplate(%q) missing After=network.target", tt.coreName)
			}
		})
	}
}

func TestCoreInstallPath(t *testing.T) {
	tests := []struct {
		coreName string
		want     string
	}{
		{"xray", "/usr/local/bin/xray"},
		{"sing-box", "/usr/local/bin/sing-box"},
		{"hysteria", "/usr/local/bin/hysteria"},
	}

	for _, tt := range tests {
		t.Run(tt.coreName, func(t *testing.T) {
			got := coreInstallPath(tt.coreName)
			if got != tt.want {
				t.Errorf("coreInstallPath(%q) = %q, want %q", tt.coreName, got, tt.want)
			}
		})
	}
}

func TestCoreServiceFilePath(t *testing.T) {
	tests := []struct {
		coreName string
		want     string
	}{
		{"xray", "/etc/systemd/system/xray.service"},
		{"sing-box", "/etc/systemd/system/sing-box.service"},
	}

	for _, tt := range tests {
		t.Run(tt.coreName, func(t *testing.T) {
			got := coreServiceFilePath(tt.coreName)
			if got != tt.want {
				t.Errorf("coreServiceFilePath(%q) = %q, want %q", tt.coreName, got, tt.want)
			}
		})
	}
}

func TestHandleCoreInstall_MissingFields(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]any
		wantErr string
	}{
		{
			name:    "missing core_name",
			payload: map[string]any{"download_url": "http://x", "checksum_sha256": "abc"},
			wantErr: "core_name is required",
		},
		{
			name:    "missing download_url",
			payload: map[string]any{"core_name": "xray", "checksum_sha256": "abc"},
			wantErr: "download_url is required",
		},
		{
			name:    "missing checksum",
			payload: map[string]any{"core_name": "xray", "download_url": "http://x"},
			wantErr: "checksum_sha256 is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, _, errText := handleCoreInstall(tt.payload, testLogger())
			if status != "failed" {
				t.Errorf("handleCoreInstall() status = %q, want %q", status, "failed")
			}
			if errText != tt.wantErr {
				t.Errorf("handleCoreInstall() errText = %q, want %q", errText, tt.wantErr)
			}
		})
	}
}

func TestHandleCoreUpdate_MissingFields(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]any
		wantErr string
	}{
		{
			name:    "missing core_name",
			payload: map[string]any{"download_url": "http://x", "checksum_sha256": "abc"},
			wantErr: "core_name is required",
		},
		{
			name:    "missing download_url",
			payload: map[string]any{"core_name": "xray", "checksum_sha256": "abc"},
			wantErr: "download_url is required",
		},
		{
			name:    "missing checksum",
			payload: map[string]any{"core_name": "xray", "download_url": "http://x"},
			wantErr: "checksum_sha256 is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, _, errText := handleCoreUpdate(tt.payload, testLogger())
			if status != "failed" {
				t.Errorf("handleCoreUpdate() status = %q, want %q", status, "failed")
			}
			if errText != tt.wantErr {
				t.Errorf("handleCoreUpdate() errText = %q, want %q", errText, tt.wantErr)
			}
		})
	}
}

func TestHandleCoreRemove_MissingCoreName(t *testing.T) {
	payload := map[string]any{}
	status, _, errText := handleCoreRemove(payload, testLogger())
	if status != "failed" {
		t.Errorf("handleCoreRemove() status = %q, want %q", status, "failed")
	}
	if errText != "core_name is required" {
		t.Errorf("handleCoreRemove() errText = %q, want %q", errText, "core_name is required")
	}
}

func TestKnownCores(t *testing.T) {
	// Verify the known cores list contains expected entries
	expected := map[string]bool{"xray": true, "sing-box": true}
	for _, core := range knownCores {
		if !expected[core] {
			t.Errorf("unexpected core in knownCores: %q", core)
		}
	}
	if len(knownCores) != len(expected) {
		t.Errorf("knownCores has %d entries, expected %d", len(knownCores), len(expected))
	}
}
