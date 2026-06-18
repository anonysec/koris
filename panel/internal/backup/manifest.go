package backup

import (
	"time"
)

// Manifest represents the metadata stored in manifest.json within a backup archive.
type Manifest struct {
	Version           string            `json:"version"`
	Timestamp         string            `json:"timestamp"`
	PanelVersion      string            `json:"panel_version"`
	Database          string            `json:"database"`
	NodesIncluded     []string          `json:"nodes_included"`
	NodesSkipped      []SkippedNode     `json:"nodes_skipped"`
	Files             map[string]FileInfo `json:"files"`
	ChecksumAlgorithm string            `json:"checksum_algorithm"`
	Checksum          string            `json:"checksum,omitempty"`
}

// SkippedNode records a node that was skipped during backup with the reason.
type SkippedNode struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// FileInfo stores metadata about a file in the archive.
type FileInfo struct {
	Size       int64 `json:"size,omitempty"`
	FilesCount int   `json:"files_count,omitempty"`
}

// GenerateManifest creates a Manifest with the given parameters.
func GenerateManifest(timestamp time.Time, panelVersion, dbName string, nodesIncluded []string, nodesSkipped []SkippedNode, files map[string]FileInfo) Manifest {
	return Manifest{
		Version:           "1.0",
		Timestamp:         timestamp.UTC().Format(time.RFC3339),
		PanelVersion:      panelVersion,
		Database:          dbName,
		NodesIncluded:     nodesIncluded,
		NodesSkipped:      nodesSkipped,
		Files:             files,
		ChecksumAlgorithm: "sha256",
	}
}
