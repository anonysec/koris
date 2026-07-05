package backup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeChecksum_KnownFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.bin")
	// Use exact bytes to avoid platform line ending differences
	content := []byte{0x68, 0x65, 0x6c, 0x6c, 0x6f} // "hello"
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	hash, err := ComputeChecksum(path)
	if err != nil {
		t.Fatalf("ComputeChecksum error: %v", err)
	}

	// SHA-256 of exact bytes "hello"
	want := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if hash != want {
		t.Errorf("hash = %q, want %q", hash, want)
	}

	// Verify format: 64 lowercase hex chars
	if len(hash) != 64 {
		t.Errorf("hash length = %d, want 64", len(hash))
	}
	for _, c := range hash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("hash contains non-lowercase-hex char: %c", c)
			break
		}
	}
}

func TestVerifyChecksum(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.bin")
	content := []byte{0x68, 0x65, 0x6c, 0x6c, 0x6f} // "hello"
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	correctHash := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"

	tests := []struct {
		name     string
		expected string
		want     bool
	}{
		{
			name:     "matching hash returns true",
			expected: correctHash,
			want:     true,
		},
		{
			name:     "mismatched hash returns false",
			expected: "0000000000000000000000000000000000000000000000000000000000000000",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := VerifyChecksum(path, tt.expected)
			if err != nil {
				t.Fatalf("VerifyChecksum error: %v", err)
			}
			if ok != tt.want {
				t.Errorf("VerifyChecksum() = %v, want %v", ok, tt.want)
			}
		})
	}
}

func TestComputeChecksum_NonExistentFile(t *testing.T) {
	_, err := ComputeChecksum("/nonexistent/path/file.txt")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}
