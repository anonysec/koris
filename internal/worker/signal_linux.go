//go:build linux

package worker

import (
	"os"
	"syscall"
)

// signalTerminate sends SIGTERM to the process for graceful shutdown on Linux.
func signalTerminate(p *os.Process) error {
	return p.Signal(syscall.SIGTERM)
}
