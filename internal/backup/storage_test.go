package backup

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		wantName string
	}{
		{
			name:     "known timestamp",
			input:    time.Date(2024, 1, 15, 2, 0, 0, 0, time.UTC),
			wantName: "backup-2024-01-15-020000.tar.gz",
		},
		{
			name:     "end of year",
			input:    time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			wantName: "backup-2024-12-31-235959.tar.gz",
		},
		{
			name:     "epoch",
			input:    time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			wantName: "backup-1970-01-01-000000.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateFilename(tt.input)
			if got != tt.wantName {
				t.Errorf("generateFilename() = %q, want %q", got, tt.wantName)
			}
			if !strings.HasSuffix(got, ".tar.gz") {
				t.Errorf("filename %q does not end with .tar.gz", got)
			}
		})
	}
}

func TestGenerateFilename_TarGzSuffix(t *testing.T) {
	timestamps := []time.Time{
		time.Date(2020, 6, 15, 12, 30, 45, 0, time.UTC),
		time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2099, 11, 28, 8, 15, 22, 0, time.UTC),
	}

	for _, ts := range timestamps {
		got := generateFilename(ts)
		if !strings.HasSuffix(got, ".tar.gz") {
			t.Errorf("generateFilename(%v) = %q, missing .tar.gz suffix", ts, got)
		}
	}
}
