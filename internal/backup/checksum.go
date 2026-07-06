package backup

import (
	"github.com/anonysec/koris/internal/safepath"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// ComputeChecksum computes and returns the hex-encoded SHA-256 hash of the file at filePath.
// It streams the file through the hash writer to avoid loading the entire file in memory.
func ComputeChecksum(filePath string) (string, error) {
	f, err := safepath.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open file for checksum: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("compute hash: %w", err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// WriteChecksumFile writes the checksum to a companion .sha256 file.
func WriteChecksumFile(archivePath, checksum string) error {
	checksumPath := archivePath + ".sha256"
	content := fmt.Sprintf("%s  %s\n", checksum, archivePath)
	return os.WriteFile(checksumPath, []byte(content), 0640)
}

// VerifyChecksum recomputes the SHA-256 hash of the file and compares with expectedHash.
func VerifyChecksum(filePath, expectedHash string) (bool, error) {
	actual, err := ComputeChecksum(filePath)
	if err != nil {
		return false, err
	}
	return actual == expectedHash, nil
}
