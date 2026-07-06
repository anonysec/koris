package updater

import (
	"github.com/anonysec/koris/internal/safepath"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/anonysec/koris/internal/safehttp"
)

// NotifyFunc is a callback used to send notifications (e.g., Telegram) on update events.
type NotifyFunc func(message string)

// ReleaseInfo holds version metadata from the release endpoint.
type ReleaseInfo struct {
	Version   string `json:"version"`
	Changelog string `json:"changelog"`
	URL       string `json:"url"`
	Checksum  string `json:"checksum_sha256"`
}

// PanelUpdater handles panel binary self-update with checksum verification and rollback.
type PanelUpdater struct {
	ReleaseURL   string
	BinaryPath   string
	RollbackPath string
	Notifier     NotifyFunc
	Client       *http.Client // injectable for tests; defaults to safehttp.NewClient
}

// NewPanelUpdater creates a new PanelUpdater instance.
func NewPanelUpdater(releaseURL, binaryPath, rollbackPath string, notifier NotifyFunc) *PanelUpdater {
	return &PanelUpdater{
		ReleaseURL:   releaseURL,
		BinaryPath:   binaryPath,
		RollbackPath: rollbackPath,
		Notifier:     notifier,
	}
}

// CheckLatest queries the release endpoint and returns available version info.
func (u *PanelUpdater) CheckLatest() (*ReleaseInfo, error) {
	client := u.Client
	if client == nil {
		client = safehttp.NewClient(60 * 1e9)
	}
	resp, err := client.Get(u.ReleaseURL)
	if err != nil {
		return nil, fmt.Errorf("[update] check latest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("[update] release endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var info ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("[update] decode release response: %w", err)
	}

	return &info, nil
}

// Apply downloads the new binary, verifies its checksum, backs up the current binary,
// and writes the new binary to BinaryPath.
func (u *PanelUpdater) Apply(info *ReleaseInfo) error {
	// Download binary from info.URL
	client := u.Client
	if client == nil {
		client = safehttp.NewClient(120 * 1e9)
	}
	resp, err := client.Get(info.URL)
	if err != nil {
		return fmt.Errorf("[update] download binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("[update] download returned %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("[update] read download body: %w", err)
	}

	// Verify SHA-256 checksum
	if !VerifyChecksum(data, info.Checksum) {
		h := sha256.Sum256(data)
		actual := hex.EncodeToString(h[:])
		return fmt.Errorf("[update] checksum mismatch: expected %s, got %s", info.Checksum, actual)
	}

	// Backup current binary to RollbackPath
	if err := copyFile(u.BinaryPath, u.RollbackPath); err != nil {
		return fmt.Errorf("[update] backup current binary: %w", err)
	}

	// Write new binary to BinaryPath with 0755 permissions
	if err := os.WriteFile(u.BinaryPath, data, 0755); err != nil {
		return fmt.Errorf("[update] write new binary: %w", err)
	}

	log.Printf("[update] applied version %s successfully", info.Version)
	return nil
}

// Rollback restores the previous binary from RollbackPath and notifies via Notifier.
func (u *PanelUpdater) Rollback(reason string) error {
	if err := copyFile(u.RollbackPath, u.BinaryPath); err != nil {
		return fmt.Errorf("[update] rollback copy: %w", err)
	}

	if u.Notifier != nil {
		u.Notifier(reason)
	}

	log.Printf("[update] rollback: %s", reason)
	return nil
}

// VerifyChecksum computes the SHA-256 hex digest of data and compares it to expected.
func VerifyChecksum(data []byte, expected string) bool {
	h := sha256.Sum256(data)
	actual := hex.EncodeToString(h[:])
	return actual == expected
}

// copyFile copies the contents of src to dst, preserving 0755 permissions.
func copyFile(src, dst string) error {
	in, err := safepath.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return nil
}
