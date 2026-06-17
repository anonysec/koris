package main

import (
	"net"
	"testing"
)

func TestFirstIPv6ReturnsValidOrEmpty(t *testing.T) {
	result := firstIPv6()
	if result == "" {
		// No global IPv6 address available (common in CI); that is acceptable.
		t.Log("firstIPv6() returned empty string (no global IPv6 available)")
		return
	}
	ip := net.ParseIP(result)
	if ip == nil {
		t.Fatalf("firstIPv6() returned invalid IP: %q", result)
	}
	if ip.To4() != nil {
		t.Fatalf("firstIPv6() returned an IPv4 address: %q", result)
	}
	if ip.IsLinkLocalUnicast() {
		t.Fatalf("firstIPv6() returned a link-local address: %q", result)
	}
	if ip.IsLoopback() {
		t.Fatalf("firstIPv6() returned loopback address: %q", result)
	}
}

func TestFirstIPv6SkipsLinkLocal(t *testing.T) {
	// This test verifies that if firstIPv6 returns a result, it is never link-local.
	result := firstIPv6()
	if result == "" {
		t.Skip("no IPv6 address available on this system")
	}
	ip := net.ParseIP(result)
	if ip == nil {
		t.Fatalf("firstIPv6() returned unparseable address: %q", result)
	}
	if ip.IsLinkLocalUnicast() {
		t.Fatalf("firstIPv6() should skip link-local addresses, got: %q", result)
	}
}
