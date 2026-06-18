package backup

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// generateFilename returns a backup filename based on the given timestamp.
// Format: backup-YYYY-MM-DD-HHmmss.tar.gz
func generateFilename(t time.Time) string {
	return fmt.Sprintf("backup-%s.tar.gz", t.Format("2006-01-02-150405"))
}

// ensureStorageDir creates the storage directory with 0750 permissions if it does not exist.
func ensureStorageDir(path string) error {
	return os.MkdirAll(path, 0750)
}

// listArchiveFiles returns all .tar.gz files in the given directory, sorted by name (ascending).
func listArchiveFiles(dir string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var archives []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".tar.gz") {
			archives = append(archives, e)
		}
	}

	sort.Slice(archives, func(i, j int) bool {
		return archives[i].Name() < archives[j].Name()
	})

	return archives, nil
}
