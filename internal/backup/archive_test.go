package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriteArchive_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test-backup.tar.gz")

	// Prepare test data
	dumpContent := "CREATE TABLE users (id INT); INSERT INTO users VALUES (1), (2);"
	dumpReader := strings.NewReader(dumpContent)

	nodeConfigs := []NodeConfigs{
		{
			NodeName: "node-1",
			Files: map[string][]byte{
				"wg0.conf":         []byte("[Interface]\nPrivateKey=abc\n"),
				"ipsec/ipsec.conf": []byte("config setup\n"),
			},
		},
		{
			NodeName: "node-2",
			Files: map[string][]byte{
				"wg0.conf": []byte("[Interface]\nPrivateKey=xyz\n"),
			},
		},
	}

	manifest := GenerateManifest(
		time.Date(2024, 1, 15, 2, 0, 0, 0, time.UTC),
		"2.1.0",
		"radius_next",
		[]string{"node-1", "node-2"},
		[]SkippedNode{{Name: "node-3", Reason: "timeout"}},
		nil,
		0, 0,
	)

	// Write archive
	if err := WriteArchive(outputPath, dumpReader, nodeConfigs, manifest); err != nil {
		t.Fatalf("WriteArchive failed: %v", err)
	}

	// Verify the file exists
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("archive file not created: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("archive file is empty")
	}

	// Read back the tar.gz and verify contents
	entries := readTarGz(t, outputPath)

	// Assert dump.sql exists with correct content
	if content, ok := entries["dump.sql"]; !ok {
		t.Error("dump.sql not found in archive")
	} else if string(content) != dumpContent {
		t.Errorf("dump.sql content mismatch: got %q, want %q", string(content), dumpContent)
	}

	// Assert manifest.json exists and is valid JSON
	if content, ok := entries["manifest.json"]; !ok {
		t.Error("manifest.json not found in archive")
	} else {
		var m Manifest
		if err := json.Unmarshal(content, &m); err != nil {
			t.Errorf("manifest.json is not valid JSON: %v", err)
		}
		if m.Version != "1.0" {
			t.Errorf("manifest version: got %q, want %q", m.Version, "1.0")
		}
		if m.Database != "radius_next" {
			t.Errorf("manifest database: got %q, want %q", m.Database, "radius_next")
		}
		if m.PanelVersion != "2.1.0" {
			t.Errorf("manifest panel_version: got %q, want %q", m.PanelVersion, "2.1.0")
		}
		if len(m.NodesIncluded) != 2 {
			t.Errorf("manifest nodes_included count: got %d, want 2", len(m.NodesIncluded))
		}
		if len(m.NodesSkipped) != 1 {
			t.Errorf("manifest nodes_skipped count: got %d, want 1", len(m.NodesSkipped))
		}
	}

	// Assert node config files exist with correct content
	assertEntry(t, entries, "configs/node-1/wg0.conf", "[Interface]\nPrivateKey=abc\n")
	assertEntry(t, entries, "configs/node-1/ipsec/ipsec.conf", "config setup\n")
	assertEntry(t, entries, "configs/node-2/wg0.conf", "[Interface]\nPrivateKey=xyz\n")
}

func TestWriteArchive_EmptyDump(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "empty-dump.tar.gz")

	dumpReader := strings.NewReader("")
	manifest := GenerateManifest(time.Now(), "1.0.0", "testdb", nil, nil, nil, 0, 0)

	if err := WriteArchive(outputPath, dumpReader, nil, manifest); err != nil {
		t.Fatalf("WriteArchive with empty dump failed: %v", err)
	}

	entries := readTarGz(t, outputPath)

	if _, ok := entries["dump.sql"]; !ok {
		t.Error("dump.sql not found in archive even with empty content")
	}
	if _, ok := entries["manifest.json"]; !ok {
		t.Error("manifest.json not found in archive")
	}
}

func TestChecksumRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "checksum-test.tar.gz")

	// Create a real archive
	dumpReader := strings.NewReader("SELECT 1;")
	manifest := GenerateManifest(time.Now(), "1.0.0", "testdb", []string{"node-1"}, nil, nil, 0, 0)
	nodeConfigs := []NodeConfigs{
		{NodeName: "node-1", Files: map[string][]byte{"test.conf": []byte("data")}},
	}

	if err := WriteArchive(outputPath, dumpReader, nodeConfigs, manifest); err != nil {
		t.Fatalf("WriteArchive failed: %v", err)
	}

	// Compute checksum
	checksum, err := ComputeChecksum(outputPath)
	if err != nil {
		t.Fatalf("ComputeChecksum failed: %v", err)
	}

	// Verify checksum format: 64-char lowercase hex
	if len(checksum) != 64 {
		t.Errorf("checksum length: got %d, want 64", len(checksum))
	}

	// Verify round-trip: VerifyChecksum should return true for computed hash
	valid, err := VerifyChecksum(outputPath, checksum)
	if err != nil {
		t.Fatalf("VerifyChecksum failed: %v", err)
	}
	if !valid {
		t.Error("VerifyChecksum returned false for correctly computed checksum")
	}

	// Verify with wrong checksum should return false
	wrongChecksum := strings.Repeat("a", 64)
	if wrongChecksum == checksum {
		wrongChecksum = strings.Repeat("b", 64)
	}
	valid, err = VerifyChecksum(outputPath, wrongChecksum)
	if err != nil {
		t.Fatalf("VerifyChecksum with wrong hash failed: %v", err)
	}
	if valid {
		t.Error("VerifyChecksum returned true for incorrect checksum")
	}
}

// readTarGz opens a .tar.gz file and returns a map of entry paths to their content.
func readTarGz(t *testing.T, path string) map[string][]byte {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open archive: %v", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	entries := make(map[string][]byte)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar read error: %v", err)
		}
		if hdr.Typeflag == tar.TypeDir {
			continue
		}
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, tr); err != nil {
			t.Fatalf("failed to read tar entry %s: %v", hdr.Name, err)
		}
		entries[hdr.Name] = buf.Bytes()
	}

	return entries
}

// assertEntry checks that the given path exists in the entries map with expected content.
func assertEntry(t *testing.T, entries map[string][]byte, path, expected string) {
	t.Helper()
	content, ok := entries[path]
	if !ok {
		t.Errorf("entry %q not found in archive", path)
		return
	}
	if string(content) != expected {
		t.Errorf("entry %q content mismatch:\ngot:  %q\nwant: %q", path, string(content), expected)
	}
}
