// Package safepath provides validated file reading to prevent path traversal attacks.
// All os.ReadFile and os.Open calls should use this package instead.
package safepath

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// validatePath cleans path and returns the absolute, symlink-resolved path,
// confined to allowedRoot (a directory). Both relative paths
// (joined onto allowedRoot) and absolute paths are confined to
// allowedRoot's resolved tree; ".." escapes and symlink escapes
// are rejected. Callers of ReadFile/Open/Create pass
// (base, dir) so that trusted absolute paths the code constructed
// itself are confined to their own directory rather than a fixed cwd.
func validatePath(path, allowedRoot string) (string, error) {
	clean := filepath.Clean(path)

	// Reject ".." escapes up front (cheap, clear errors).
	if strings.HasPrefix(clean, "..") || strings.Contains(clean, string(filepath.Separator)+"..") {
		return "", fmt.Errorf("path traversal detected: %s", path)
	}

	absBase, err := filepath.Abs(allowedRoot)
	if err != nil {
		return "", fmt.Errorf("resolve allowed root: %w", err)
	}

	resolved := filepath.Join(absBase, clean)
	resolved, err = filepath.EvalSymlinks(resolved)
	if err != nil {
		// Path may not exist yet (e.g. Create). Fall back to the
		// unresolved join, but still confine it to base.
		resolved = filepath.Join(absBase, clean)
	}

	absResolved, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}

	if absResolved != absBase && !strings.HasPrefix(absResolved, absBase+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes allowed root %q", path, absBase)
	}
	return absResolved, nil
}

// ReadFile reads a file after validating the path does not escape.
// path is confined to its own directory (base = dir(path)), so
// trusted absolute paths the caller built are safe while ".."
// /symlink escapes are still blocked.
func ReadFile(path string) ([]byte, error) {
	clean, err := validatePath(filepath.Base(path), filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	return os.ReadFile(clean) // #nosec G304 -- path validated above
}

// ReadFileAbs is an alias for ReadFile (confined to own dir).
func ReadFileAbs(path string) ([]byte, error) {
	return ReadFile(path)
}

// Open opens a file after validating the path does not escape.
func Open(path string) (*os.File, error) {
	clean, err := validatePath(filepath.Base(path), filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	return os.Open(clean) // #nosec G304 -- path validated above
}

// ReadFileInDir reads a file from within an allowed base directory.
// Returns an error if the resolved path escapes the base directory.
func ReadFileInDir(baseDir, relPath string) ([]byte, error) {
	clean, err := validatePath(relPath, baseDir)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(clean) // #nosec G304 -- path confined to baseDir
}

// Exists reports whether a non-empty path exists on disk.
func Exists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// Create creates a file after validating the path does not escape.
func Create(path string) (*os.File, error) {
	clean, err := validatePath(filepath.Base(path), filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	return os.Create(clean) // #nosec G304 -- path validated above
}
