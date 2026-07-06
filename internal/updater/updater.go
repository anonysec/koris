package updater

import (
	"github.com/anonysec/koris/internal/safepath"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// maxDownloadSize is the maximum size of an update binary (256 MB).
const maxDownloadSize = 256 << 20

// validateURL checks that a URL is HTTPS with a public hostname, preventing SSRF.
func validateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("only HTTPS URLs are allowed, got %q", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("empty hostname")
	}
	// Block private/loopback/link-local addresses
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("DNS lookup failed for %q: %w", host, err)
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("URL resolves to blocked address %s (SSRF protection)", ip)
		}
	}
	return nil
}

// UpdateInfo holds information about an available update.
type UpdateInfo struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	DownloadURL    string `json:"download_url"`
	Checksum       string `json:"checksum"` // SHA256 hex-encoded
	Changelog      string `json:"changelog"`
	Available      bool   `json:"available"`
}

// Updater handles checking for, applying, and rolling back panel updates.
type Updater struct {
	currentVersion string
	releaseURL     string // URL to check for releases (JSON endpoint)
	binaryPath     string // path to the current binary
	backupPath     string // path to store backup
	client         *http.Client
}

// releaseResponse is the expected JSON format from the release endpoint.
type releaseResponse struct {
	Version     string `json:"version"`
	DownloadURL string `json:"download_url"`
	Checksum    string `json:"checksum_sha256"`
	Changelog   string `json:"changelog"`
}

// New creates a new Updater instance.
// If binaryPath is empty, os.Executable() is used to detect it.
func New(version, releaseURL, binaryPath string) *Updater {
	if binaryPath == "" {
		exe, err := os.Executable()
		if err == nil {
			binaryPath = exe
		}
	}

	backupPath := binaryPath + ".bak"

	return &Updater{
		currentVersion: version,
		releaseURL:     releaseURL,
		binaryPath:     binaryPath,
		backupPath:     backupPath,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Check queries the release URL for the latest version and returns UpdateInfo.
func (u *Updater) Check() (*UpdateInfo, error) {
	if err := validateURL(u.releaseURL); err != nil {
		return nil, fmt.Errorf("[updater] release URL validation: %w", err)
	}
	req, err := http.NewRequest(http.MethodGet, u.releaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("[updater] create check request: %w", err)
	}
	req.Header.Set("User-Agent", "github.com/anonysec/koris/"+u.currentVersion)

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("[updater] check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("[updater] release endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var release releaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("[updater] decode release response: %w", err)
	}

	available := CompareVersions(u.currentVersion, release.Version)

	info := &UpdateInfo{
		CurrentVersion: u.currentVersion,
		LatestVersion:  release.Version,
		DownloadURL:    release.DownloadURL,
		Checksum:       release.Checksum,
		Changelog:      release.Changelog,
		Available:      available,
	}

	return info, nil
}

// Apply downloads the update, verifies its checksum, backs up the current binary,
// replaces it, and signals a graceful restart. The progress callback is called at
// each stage with a stage name and completion percentage (0.0 to 1.0).
func (u *Updater) Apply(info *UpdateInfo, progress func(stage string, pct float64)) error {
	if info == nil || !info.Available {
		return fmt.Errorf("[updater] no update available to apply")
	}

	if progress == nil {
		progress = func(string, float64) {}
	}

	// Stage 1: Download
	progress("downloading", 0.0)
	data, err := u.download(info.DownloadURL)
	if err != nil {
		return fmt.Errorf("[updater] download failed: %w", err)
	}
	progress("downloading", 1.0)

	// Stage 2: Verify checksum
	progress("verifying", 0.0)
	if !verifyChecksum(data, info.Checksum) {
		actual := sha256.Sum256(data)
		return fmt.Errorf("[updater] checksum mismatch: expected %s, got %s",
			info.Checksum, hex.EncodeToString(actual[:]))
	}
	progress("verifying", 1.0)

	// Stage 3: Backup current binary
	progress("backing_up", 0.0)
	if err := u.backupBinary(); err != nil {
		return fmt.Errorf("[updater] backup failed: %w", err)
	}
	progress("backing_up", 1.0)

	// Stage 4: Replace binary
	progress("replacing", 0.0)
	if err := u.replaceBinary(data); err != nil {
		// Attempt rollback on replace failure
		_ = u.Rollback()
		return fmt.Errorf("[updater] replace failed: %w", err)
	}
	progress("replacing", 1.0)

	// Stage 5: Signal restart
	progress("restarting", 0.0)
	if err := u.signalRestart(); err != nil {
		log.Printf("[updater] restart signal failed, attempting rollback: %v", err)
		_ = u.Rollback()
		return fmt.Errorf("[updater] restart failed: %w", err)
	}
	progress("restarting", 1.0)

	log.Printf("[updater] update applied: %s -> %s", info.CurrentVersion, info.LatestVersion)
	return nil
}

// Rollback restores the backup binary over the current binary and signals a restart.
func (u *Updater) Rollback() error {
	if _, err := os.Stat(u.backupPath); os.IsNotExist(err) {
		return fmt.Errorf("[updater] no backup found at %s", u.backupPath)
	}

	// Read backup
	data, err := safepath.ReadFile(u.backupPath)
	if err != nil {
		return fmt.Errorf("[updater] read backup: %w", err)
	}

	// Write backup over current binary via temp file
	dir := filepath.Dir(u.binaryPath)
	tmpFile, err := os.CreateTemp(dir, "panel-rollback-*")
	if err != nil {
		return fmt.Errorf("[updater] create rollback temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("[updater] write rollback temp file: %w", err)
	}
	tmpFile.Close()

	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("[updater] chmod rollback file: %w", err)
	}

	if err := os.Rename(tmpPath, u.binaryPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("[updater] rename rollback binary: %w", err)
	}

	log.Printf("[updater] rollback applied from %s", u.backupPath)

	// Signal restart after rollback
	if err := u.signalRestart(); err != nil {
		return fmt.Errorf("[updater] restart after rollback failed: %w", err)
	}

	return nil
}

// download fetches the binary from the given URL and returns its contents.
// The URL is validated against SSRF and the response is capped at maxDownloadSize.
func (u *Updater) download(dlURL string) ([]byte, error) {
	if err := validateURL(dlURL); err != nil {
		return nil, fmt.Errorf("download URL validation: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, dlURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}
	req.Header.Set("User-Agent", "github.com/anonysec/koris/"+u.currentVersion)

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("download returned %d: %s", resp.StatusCode, string(body))
	}

	// Cap download size to prevent memory exhaustion
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxDownloadSize+1))
	if err != nil {
		return nil, fmt.Errorf("read download body: %w", err)
	}
	if int64(len(data)) > maxDownloadSize {
		return nil, fmt.Errorf("download exceeds maximum size of %d bytes", maxDownloadSize)
	}

	return data, nil
}

// backupBinary copies the current binary to the backup path.
func (u *Updater) backupBinary() error {
	src, err := safepath.Open(u.binaryPath)
	if err != nil {
		return fmt.Errorf("open current binary: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(u.backupPath)
	if err != nil {
		return fmt.Errorf("create backup file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy to backup: %w", err)
	}

	// Preserve executable permissions
	info, err := os.Stat(u.binaryPath)
	if err == nil {
		_ = os.Chmod(u.backupPath, info.Mode())
	}

	return nil
}

// replaceBinary writes the new binary data to the binary path via a temp file rename.
func (u *Updater) replaceBinary(data []byte) error {
	dir := filepath.Dir(u.binaryPath)
	tmpFile, err := os.CreateTemp(dir, "panel-update-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp file: %w", err)
	}
	tmpFile.Close()

	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("chmod temp file: %w", err)
	}

	if err := os.Rename(tmpPath, u.binaryPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename new binary: %w", err)
	}

	return nil
}

// signalRestart signals a graceful restart. It first attempts SIGHUP for graceful
// restart, falling back to systemctl restart if available.
//
// In test builds (detected via testing.Testing()) this is a no-op — otherwise
// the SIGHUP fallback would kill the test binary.
func (u *Updater) signalRestart() error {
	if testing.Testing() {
		return nil
	}
	// Try systemctl first (most common deployment)
	if _, err := exec.LookPath("systemctl"); err == nil {
		cmd := exec.Command("systemctl", "restart", "koris-panel")
		if err := cmd.Run(); err != nil {
			log.Printf("[updater] systemctl restart failed: %v, trying SIGHUP", err)
		} else {
			return nil
		}
	}

	// Fallback: send SIGHUP to self for graceful restart
	proc, err := os.FindProcess(syscall.Getpid())
	if err != nil {
		return fmt.Errorf("find self process: %w", err)
	}

	if err := proc.Signal(syscall.SIGHUP); err != nil {
		return fmt.Errorf("send SIGHUP: %w", err)
	}

	return nil
}

// verifyChecksum computes the SHA-256 hash of data and compares it to the
// expected hex-encoded checksum string.
func verifyChecksum(data []byte, expected string) bool {
	if expected == "" {
		return false
	}
	h := sha256.Sum256(data)
	actual := hex.EncodeToString(h[:])
	return strings.EqualFold(actual, strings.TrimSpace(expected))
}

// CompareVersions returns true if the latest version is greater than the current version.
// Supports formats "v1.2.3" and "1.2.3". Comparison is numeric on major.minor.patch.
func CompareVersions(current, latest string) bool {
	curMajor, curMinor, curPatch, okCur := parseVersion(current)
	latMajor, latMinor, latPatch, okLat := parseVersion(latest)

	if !okCur || !okLat {
		return false
	}

	if latMajor != curMajor {
		return latMajor > curMajor
	}
	if latMinor != curMinor {
		return latMinor > curMinor
	}
	return latPatch > curPatch
}

// parseVersion parses a semver string like "v0.93.1" or "0.93.1" into major, minor, patch.
func parseVersion(v string) (major, minor, patch int, ok bool) {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")

	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return 0, 0, 0, false
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, false
	}
	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, false
	}
	patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, false
	}

	return major, minor, patch, true
}
