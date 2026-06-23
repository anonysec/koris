//go:build !linux

package worker

import "net"

// ListenReusePort on non-Linux platforms falls back to standard net.Listen
// since SO_REUSEPORT is a Linux-specific socket option.
func ListenReusePort(network, addr string) (net.Listener, error) {
	return net.Listen(network, addr)
}
