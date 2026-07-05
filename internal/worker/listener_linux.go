//go:build linux

package worker

import (
	"context"
	"net"
	"syscall"
)

// SO_REUSEPORT Linux socket option constant.
const soReusePort = 0xF // syscall.SO_REUSEPORT equivalent (15)

// ListenReusePort creates a TCP listener with SO_REUSEPORT enabled,
// allowing multiple processes to bind to the same address for
// kernel-level load balancing.
func ListenReusePort(network, addr string) (net.Listener, error) {
	cfg := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var opErr error
			err := c.Control(func(fd uintptr) {
				opErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, soReusePort, 1)
			})
			if err != nil {
				return err
			}
			return opErr
		},
	}
	return cfg.Listen(context.Background(), network, addr)
}
