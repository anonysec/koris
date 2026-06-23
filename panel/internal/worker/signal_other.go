//go:build !linux

package worker

import "os"

// signalTerminate on non-Linux platforms uses os.Interrupt as the best
// available graceful signal. On Windows this is not fully supported, so
// the shutdown path relies on Kill() after the deadline.
func signalTerminate(p *os.Process) error {
	return p.Signal(os.Interrupt)
}
