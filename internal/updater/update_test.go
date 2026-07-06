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

func TestVerifyChecksum_PanelUpdater(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
		want     bool
	}{
		{
			name:     "valid checksum",
			data:     []byte("hello world"),
			expected: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
			want:     true,
		},
		{
			name:     "invalid checksum",
			data:     []byte("hello world"),
			expected: "0000000000000000000000000000000000000000000000000000000000000000",
			want:     false,
		},
		{
			name:     "empty data",
			data:     []byte(""),
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			want:     true,
		},
		{
			name:     "empty expected",
			data:     []byte("test"),
			expected: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VerifyChecksum(tt.data, tt.expected)
			if got != tt.want {
				t.Errorf("VerifyChecksum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewPanelUpdater(t *testing.T) {
	notified := false
	notifier := func(msg string) { notified = true; _ = msg }

	u := NewPanelUpdater("http://example.com/release.json", "/usr/local/bin/koris", "/usr/local/bin/koris.bak", notifier)

	if u.ReleaseURL != "http://example.com/release.json" {
		t.Errorf("ReleaseURL = %q, want %q", u.ReleaseURL, "http://example.com/release.json")
	}
	if u.BinaryPath != "/usr/local/bin/koris" {
		t.Errorf("BinaryPath = %q, want %q", u.BinaryPath, "/usr/local/bin/koris")
	}
	if u.RollbackPath != "/usr/local/bin/koris.bak" {
		t.Errorf("RollbackPath = %q, want %q", u.RollbackPath, "/usr/local/bin/koris.bak")
	}
	if u.Notifier == nil {
		t.Error("Notifier should not be nil")
	}

	u.Notifier("test")
	if !notified {
		t.Error("Notifier was not called")
	}
}

func TestCheckLatest(t *testing.T) {
	expected := ReleaseInfo{
		Version:   "0.95.0",
		Changelog: "Bug fixes and improvements",
		URL:       "https://example.com/panel-0.95.0",
		Checksum:  "abc123def456",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	u := NewPanelUpdater(srv.URL, "/tmp/panel", "/tmp/panel.bak", nil)
	info, err := u.CheckLatest()
	if err != nil {
		t.Fatalf("CheckLatest() error: %v", err)
	}

	if info.Version != expected.Version {
		t.Errorf("Version = %q, want %q", info.Version, expected.Version)
	}
	if info.Changelog != expected.Changelog {
		t.Errorf("Changelog = %q, want %q", info.Changelog, expected.Changelog)
	}
	if info.URL != expected.URL {
		t.Errorf("URL = %q, want %q", info.URL, expected.URL)
	}
	if info.Checksum != expected.Checksum {
		t.Errorf("Checksum = %q, want %q", info.Checksum, expected.Checksum)
	}
}

func TestCheckLatest_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	u := NewPanelUpdater(srv.URL, "/tmp/panel", "/tmp/panel.bak", nil)
	_, err := u.CheckLatest()
	if err == nil {
		t.Fatal("CheckLatest() expected error for HTTP 500")
	}
}

func TestApply_Success(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "panel")
	rollbackPath := filepath.Join(tmpDir, "panel.bak")

	// Write a fake current binary
	if err := os.WriteFile(binaryPath, []byte("old-binary"), 0755); err != nil {
		t.Fatal(err)
	}

	// Prepare new binary data and compute checksum
	newBinary := []byte("new-binary-content-v2")
	h := sha256.Sum256(newBinary)
	checksum := hex.EncodeToString(h[:])

	// Serve the new binary
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(newBinary)
	}))
	defer srv.Close()

	u := NewPanelUpdater("", binaryPath, rollbackPath, nil)

	info := &ReleaseInfo{
		Version:  "0.95.0",
		URL:      srv.URL,
		Checksum: checksum,
	}

	if err := u.Apply(info); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	// Verify new binary was written
	got, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(newBinary) {
		t.Errorf("binary content = %q, want %q", got, newBinary)
	}

	// Verify rollback backup was created
	backup, err := os.ReadFile(rollbackPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(backup) != "old-binary" {
		t.Errorf("backup content = %q, want %q", backup, "old-binary")
	}
}

func TestApply_ChecksumMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "panel")
	rollbackPath := filepath.Join(tmpDir, "panel.bak")

	if err := os.WriteFile(binaryPath, []byte("old-binary"), 0755); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("some-binary-data"))
	}))
	defer srv.Close()

	u := NewPanelUpdater("", binaryPath, rollbackPath, nil)

	info := &ReleaseInfo{
		Version:  "0.95.0",
		URL:      srv.URL,
		Checksum: "0000000000000000000000000000000000000000000000000000000000000000",
	}

	err := u.Apply(info)
	if err == nil {
		t.Fatal("Apply() expected checksum mismatch error")
	}
}

func TestRollback(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "panel")
	rollbackPath := filepath.Join(tmpDir, "panel.bak")

	// Write current (broken) binary and backup
	if err := os.WriteFile(binaryPath, []byte("broken-binary"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rollbackPath, []byte("good-binary"), 0755); err != nil {
		t.Fatal(err)
	}

	var notifiedMsg string
	notifier := func(msg string) { notifiedMsg = msg }

	u := NewPanelUpdater("", binaryPath, rollbackPath, notifier)

	if err := u.Rollback("startup failed"); err != nil {
		t.Fatalf("Rollback() error: %v", err)
	}

	// Verify binary was restored
	got, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "good-binary" {
		t.Errorf("binary content = %q, want %q", got, "good-binary")
	}

	// Verify notifier was called
	if notifiedMsg != "startup failed" {
		t.Errorf("notifier message = %q, want %q", notifiedMsg, "startup failed")
	}
}

func TestRollback_NoBackup(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "panel")
	rollbackPath := filepath.Join(tmpDir, "panel.bak")

	// Only create the binary, no backup
	if err := os.WriteFile(binaryPath, []byte("current"), 0755); err != nil {
		t.Fatal(err)
	}

	u := NewPanelUpdater("", binaryPath, rollbackPath, nil)

	err := u.Rollback("test")
	if err == nil {
		t.Fatal("Rollback() expected error when no backup exists")
	}
}
