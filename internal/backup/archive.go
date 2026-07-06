package backup

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/anonysec/koris/internal/safepath"
)

// NodeConfigs holds collected config data for a single node.
type NodeConfigs struct {
	NodeName string
	Files    map[string][]byte // relative path -> content
}

// WriteArchive creates a .tar.gz backup archive at outputPath containing:
// - dump.sql (streamed from dumpReader)
// - configs/{node_name}/{path} for each node
// - manifest.json
func WriteArchive(outputPath string, dumpReader io.Reader, nodeConfigs []NodeConfigs, manifest Manifest) error {
	f, err := safepath.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create archive file: %w", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Write dump.sql from streaming reader.
	// Since we don't know the size upfront, write to a temp file first.
	tmpDump, err := os.CreateTemp("", "koris-dump-*.sql")
	if err != nil {
		return fmt.Errorf("create temp dump: %w", err)
	}
	defer os.Remove(tmpDump.Name())
	defer tmpDump.Close()

	dumpSize, err := io.Copy(tmpDump, dumpReader)
	if err != nil {
		return fmt.Errorf("copy dump to temp: %w", err)
	}
	if _, err := tmpDump.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek temp dump: %w", err)
	}

	if err := tw.WriteHeader(&tar.Header{
		Name:    "dump.sql",
		Size:    dumpSize,
		Mode:    0640,
		ModTime: time.Now(),
	}); err != nil {
		return fmt.Errorf("write dump header: %w", err)
	}
	if _, err := io.Copy(tw, tmpDump); err != nil {
		return fmt.Errorf("write dump content: %w", err)
	}

	// Write node config files
	for _, nc := range nodeConfigs {
		for path, content := range nc.Files {
			fullPath := fmt.Sprintf("configs/%s/%s", nc.NodeName, path)
			if err := tw.WriteHeader(&tar.Header{
				Name:    fullPath,
				Size:    int64(len(content)),
				Mode:    0640,
				ModTime: time.Now(),
			}); err != nil {
				return fmt.Errorf("write config header %s: %w", fullPath, err)
			}
			if _, err := tw.Write(content); err != nil {
				return fmt.Errorf("write config content %s: %w", fullPath, err)
			}
		}
	}

	// Write manifest.json
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := tw.WriteHeader(&tar.Header{
		Name:    "manifest.json",
		Size:    int64(len(manifestData)),
		Mode:    0640,
		ModTime: time.Now(),
	}); err != nil {
		return fmt.Errorf("write manifest header: %w", err)
	}
	if _, err := tw.Write(manifestData); err != nil {
		return fmt.Errorf("write manifest content: %w", err)
	}

	return nil
}
