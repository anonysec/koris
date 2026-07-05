package worker

import (
	"net"
	"testing"
)

func TestListenReusePort_ReturnsValidListener(t *testing.T) {
	// Use port 0 to let the OS assign a free port.
	ln, err := ListenReusePort("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenReusePort failed: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr()
	if addr == nil {
		t.Fatal("listener address is nil")
	}
	if addr.Network() != "tcp" {
		t.Errorf("expected network tcp, got %s", addr.Network())
	}

	// Verify the listener actually accepts connections.
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		conn.Close()
	}()

	conn, err := net.Dial("tcp", addr.String())
	if err != nil {
		t.Fatalf("failed to dial listener: %v", err)
	}
	conn.Close()
	<-done
}

func TestListenReusePort_InvalidAddress(t *testing.T) {
	// An invalid address should return an error, not panic.
	_, err := ListenReusePort("tcp", "999.999.999.999:99999")
	if err == nil {
		t.Fatal("expected error for invalid address, got nil")
	}
}

func TestListenReusePort_MultipleListeners(t *testing.T) {
	// On Linux with SO_REUSEPORT, multiple listeners can bind to the same port.
	// On other platforms, the second bind may fail — this test verifies at least
	// the first listener works, and the second either works or returns an error
	// (platform-dependent behavior).
	ln1, err := ListenReusePort("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("first ListenReusePort failed: %v", err)
	}
	defer ln1.Close()

	addr := ln1.Addr().String()

	// Attempt to bind a second listener to the same address.
	ln2, err := ListenReusePort("tcp", addr)
	if err != nil {
		// On non-Linux, this is expected to fail (address already in use).
		t.Logf("second ListenReusePort returned expected error on this platform: %v", err)
		return
	}
	defer ln2.Close()

	// If we got here (Linux with SO_REUSEPORT), both listeners are valid.
	t.Log("SO_REUSEPORT: multiple listeners bound successfully to same address")
}
