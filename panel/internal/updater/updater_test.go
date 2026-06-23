package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		latest   string
		expected bool
	}{
		{"newer patch", "v0.92.0", "v0.92.1", true},
		{"newer minor", "v0.92.1", "v0.93.0", true},
		{"newer major", "v0.92.1", "v1.0.0", true},
		{"same version", "v0.92.1", "v0.92.1", false},
		{"older version", "v0.93.0", "v0.92.1", false},
		{"without v prefix", "0.92.0", "0.92.1", true},
		{"mixed prefix", "v0.92.0", "0.92.1", true},
		{"invalid current", "invalid", "v0.92.1", false},
		{"invalid latest", "v0.92.1", "invalid", false},
		{"empty strings", "", "", false},
		{"two part version", "v0.92", "v0.93.0", false},
		{"large numbers", "v0.99.99", "v0.100.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareVersions(tt.current, tt.latest)
			if result != tt.expected {
				t.Errorf("CompareVersions(%q, %q) = %v, want %v",
					tt.current, tt.latest, result, tt.expected)
			}
		})
	}
}

func TestVerifyChecksum(t *testing.T) {
	data := []byte("hello world")
	h := sha256.Sum256(data)
	validChecksum := hex.EncodeToString(h[:])

	tests := []struct {
		name     string
		data     []byte
		checksum string
		expected bool
	}{
		{"valid checksum", data, validChecksum, true},
		{"uppercase checksum", data, "B94D27B9934D3E08A52E52D7DA7DABFAC484EFE37A5380EE9088F7ACE2EFCDE9", true},
		{"invalid checksum", data, "0000000000000000000000000000000000000000000000000000000000000000", false},
		{"empty checksum", data, "", false},
		{"checksum with whitespace", data, "  " + validChecksum + "  ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := verifyChecksum(tt.data, tt.checksum)
			if result != tt.expected {
				t.Errorf("verifyChecksum() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCheck(t *testing.T) {
	// Set up a test server that returns release info
	release := releaseResponse{
		Version:     "v0.93.0",
		DownloadURL: "https://example.com/panel-v0.93.0",
		Checksum:    "abc123",
		Changelog:   "Bug fixes and improvements",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()

	u := New("v0.92.0", server.URL, "/usr/local/bin/koris")

	info, err := u.Check()
	if err != nil {
		t.Fatalf("Check() error: %v", err)
	}

	if !info.Available {
		t.Error("Check() should report update available")
	}
	if info.CurrentVersion != "v0.92.0" {
		t.Errorf("CurrentVersion = %q, want %q", info.CurrentVersion, "v0.92.0")
	}
	if info.LatestVersion != "v0.93.0" {
		t.Errorf("LatestVersion = %q, want %q", info.LatestVersion, "v0.93.0")
	}
	if info.DownloadURL != "https://example.com/panel-v0.93.0" {
		t.Errorf("DownloadURL = %q, want %q", info.DownloadURL, "https://example.com/panel-v0.93.0")
	}
	if info.Changelog != "Bug fixes and improvements" {
		t.Errorf("Changelog = %q, want %q", info.Changelog, "Bug fixes and improvements")
	}
}

func TestCheckNoUpdate(t *testing.T) {
	release := releaseResponse{
		Version:     "v0.92.0",
		DownloadURL: "https://example.com/panel-v0.92.0",
		Checksum:    "abc123",
		Changelog:   "",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()

	u := New("v0.92.0", server.URL, "/usr/local/bin/koris")

	info, err := u.Check()
	if err != nil {
		t.Fatalf("Check() error: %v", err)
	}

	if info.Available {
		t.Error("Check() should not report update available for same version")
	}
}

func TestCheckServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	u := New("v0.92.0", server.URL, "/usr/local/bin/koris")

	_, err := u.Check()
	if err == nil {
		t.Error("Check() should return error on server failure")
	}
}

func TestApplyVerifiesChecksum(t *testing.T) {
	binaryData := []byte("#!/bin/sh\necho panel v0.93.0")
	wrongChecksum := "0000000000000000000000000000000000000000000000000000000000000000"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(binaryData)
	}))
	defer server.Close()

	// Create a temp binary to act as the current binary
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "panel")
	os.WriteFile(binaryPath, []byte("old binary"), 0755)

	u := New("v0.92.0", "", binaryPath)

	info := &UpdateInfo{
		Available:   true,
		DownloadURL: server.URL + "/download",
		Checksum:    wrongChecksum,
	}

	err := u.Apply(info, nil)
	if err == nil {
		t.Error("Apply() should fail on checksum mismatch")
	}
}

func TestApplySuccess(t *testing.T) {
	binaryData := []byte("#!/bin/sh\necho panel v0.93.0")
	h := sha256.Sum256(binaryData)
	checksum := hex.EncodeToString(h[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(binaryData)
	}))
	defer server.Close()

	// Create a temp binary to act as the current binary
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "panel")
	os.WriteFile(binaryPath, []byte("old binary"), 0755)

	u := New("v0.92.0", "", binaryPath)
	// Override signalRestart to not actually restart anything in tests
	// We test up to the point before restart by checking binary was replaced

	info := &UpdateInfo{
		Available:      true,
		LatestVersion:  "v0.93.0",
		CurrentVersion: "v0.92.0",
		DownloadURL:    server.URL + "/download",
		Checksum:       checksum,
	}

	// Track progress calls
	var stages []string
	progress := func(stage string, pct float64) {
		if pct == 0.0 {
			stages = append(stages, stage)
		}
	}

	// Apply will fail at the restart step (no systemctl in test), but binary should be replaced
	_ = u.Apply(info, progress)

	// Verify progress stages were called
	expectedStages := []string{"downloading", "verifying", "backing_up", "replacing", "restarting"}
	for i, expected := range expectedStages {
		if i >= len(stages) {
			break
		}
		if stages[i] != expected {
			t.Errorf("stage[%d] = %q, want %q", i, stages[i], expected)
		}
	}

	// Verify backup was created
	if _, err := os.Stat(binaryPath + ".bak"); os.IsNotExist(err) {
		t.Error("backup file should exist after Apply")
	}
}

func TestRollbackNoBackup(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "panel")
	os.WriteFile(binaryPath, []byte("current binary"), 0755)

	u := New("v0.93.0", "", binaryPath)

	err := u.Rollback()
	if err == nil {
		t.Error("Rollback() should fail when no backup exists")
	}
}

func TestRollbackSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "panel")
	backupPath := binaryPath + ".bak"

	// Create "current" binary (bad version) and "backup" (good version)
	os.WriteFile(binaryPath, []byte("bad binary v0.93.0"), 0755)
	os.WriteFile(backupPath, []byte("good binary v0.92.0"), 0755)

	u := New("v0.93.0", "", binaryPath)

	// Rollback will fail at restart (no systemctl), but binary should be replaced
	_ = u.Rollback()

	// Verify the binary was restored from backup
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("read binary after rollback: %v", err)
	}
	if string(data) != "good binary v0.92.0" {
		t.Errorf("binary content = %q, want %q", string(data), "good binary v0.92.0")
	}
}

func TestNewAutoDetectsBinaryPath(t *testing.T) {
	u := New("v0.92.0", "https://example.com/releases", "")

	if u.binaryPath == "" {
		t.Error("New() should auto-detect binary path when empty")
	}
	if u.backupPath == "" {
		t.Error("New() should set backup path")
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input        string
		major, minor int
		patch        int
		ok           bool
	}{
		{"v0.92.1", 0, 92, 1, true},
		{"0.92.1", 0, 92, 1, true},
		{"v1.0.0", 1, 0, 0, true},
		{"v0.100.0", 0, 100, 0, true},
		{"invalid", 0, 0, 0, false},
		{"v0.92", 0, 0, 0, false},
		{"", 0, 0, 0, false},
		{"v0.92.abc", 0, 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			major, minor, patch, ok := parseVersion(tt.input)
			if ok != tt.ok {
				t.Fatalf("parseVersion(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			}
			if ok {
				if major != tt.major || minor != tt.minor || patch != tt.patch {
					t.Errorf("parseVersion(%q) = (%d, %d, %d), want (%d, %d, %d)",
						tt.input, major, minor, patch, tt.major, tt.minor, tt.patch)
				}
			}
		})
	}
}
