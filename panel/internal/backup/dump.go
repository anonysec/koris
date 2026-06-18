package backup

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
)

// streamMySQLDump executes mysqldump and returns a reader for the dump output.
// It streams stdout directly without buffering the entire dump in memory.
// On error (non-zero exit), returns an error with stderr content.
func streamMySQLDump(ctx context.Context, cfg Config) (io.ReadCloser, func() error, error) {
	args := []string{
		"--single-transaction",
		"--routines",
		"--triggers",
		"-h", cfg.DBHost,
		"-u", cfg.DBUser,
		cfg.DBName,
	}

	cmd := exec.CommandContext(ctx, "mysqldump", args...)
	cmd.Env = append(cmd.Environ(), "MYSQL_PWD="+cfg.DBPass)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("mysqldump stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("mysqldump start: %w", err)
	}

	// wait function to be called after reading is done
	wait := func() error {
		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("mysqldump failed: %w, stderr: %s", err, stderr.String())
		}
		return nil
	}

	return stdout, wait, nil
}
