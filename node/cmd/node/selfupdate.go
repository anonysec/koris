package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// updateAgentPayload is the expected JSON payload for the update_agent task.
type updateAgentPayload struct {
	Version  string `json:"version"`
	URL      string `json:"url"`
	Checksum string `json:"checksum"`
}

// handleUpdateAgent downloads a new agent binary, verifies its SHA-256 checksum,
// atomically replaces the current binary, and triggers a systemctl restart.
func handleUpdateAgent(payload json.RawMessage) error {
	var p updateAgentPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	if p.Version == "" {
		return fmt.Errorf("missing version in payload")
	}
	if p.URL == "" {
		return fmt.Errorf("missing url in payload")
	}
	if p.Checksum == "" {
		return fmt.Errorf("missing checksum in payload")
	}

	// 1. Download the binary from the URL
	resp, err := http.Get(p.URL)
	if err != nil {
		return fmt.Errorf("download binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	// 2. Read the full response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read binary body: %w", err)
	}

	// 3. Compute SHA-256 hash and verify against the checksum
	h := sha256.Sum256(data)
	actualHash := hex.EncodeToString(h[:])
	if !strings.EqualFold(actualHash, strings.TrimSpace(p.Checksum)) {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", p.Checksum, actualHash)
	}

	// 4. Determine current binary path
	binaryPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}
	binaryPath, err = filepath.EvalSymlinks(binaryPath)
	if err != nil {
		return fmt.Errorf("resolve executable symlinks: %w", err)
	}

	// 5. Write the new binary to a temp file in the same directory
	dir := filepath.Dir(binaryPath)
	tmpPath := filepath.Join(dir, filepath.Base(binaryPath)+".new")

	if err := os.WriteFile(tmpPath, data, 0755); err != nil {
		return fmt.Errorf("write temp binary: %w", err)
	}

	// 6. Set permissions to 0755 (already set via WriteFile, but explicit for clarity)
	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("chmod temp binary: %w", err)
	}

	// 7. Atomic replacement via rename
	if err := os.Rename(tmpPath, binaryPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename binary: %w", err)
	}

	// 8. Log the update
	log.Printf("[update] agent updated to version %s, triggering restart", p.Version)

	// 9. Trigger restart via systemctl
	cmd := exec.Command("systemctl", "restart", "koris-node")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("systemctl restart koris-node: %w", err)
	}

	return nil
}
