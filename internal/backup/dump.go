package backup

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
)

// streamPgDump executes pg_dump and returns a reader for the dump output.
// It streams stdout directly without buffering the entire dump in memory.
// On error (non-zero exit), returns an error with stderr content.
//
// Plain SQL format is used so the dump is stored as dump.sql in the archive
// and can be restored by piping it into psql.
func streamPgDump(ctx context.Context, cfg Config) (io.ReadCloser, func() error, error) {
	port := cfg.DBPort
	if port == 0 {
		port = 5432
	}
	args := []string{
		"-h", cfg.DBHost,
		"-p", strconv.Itoa(port),
		"-U", cfg.DBUser,
		"--no-owner",
		"--no-privileges",
		"--clean",
		"--if-exists",
		cfg.DBName,
	}

	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	cmd.Env = append(cmd.Environ(), "PGPASSWORD="+cfg.DBPass)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("pg_dump stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("pg_dump start: %w", err)
	}

	// wait function to be called after reading is done
	wait := func() error {
		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("pg_dump failed: %w, stderr: %s", err, stderr.String())
		}
		return nil
	}

	return stdout, wait, nil
}
