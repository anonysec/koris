package main

import (
	"strings"
	"testing"
)

func TestIsValidWireGuardKey(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		valid bool
	}{
		{"valid key", "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY=", true},
		{"empty", "", false},
		{"too short", "YWJj", false},
		{"too long", "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY=aa", false},
		{"invalid base64", "$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$", false},
		{"43 chars valid base64", "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NQ==", false},
		{"newline in key", "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXox\njM0NTY=", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isValidWireGuardKey(tc.key)
			if got != tc.valid {
				t.Errorf("isValidWireGuardKey(%q) = %v, want %v", tc.key, got, tc.valid)
			}
		})
	}
}

func TestIsValidAllowedIPs(t *testing.T) {
	tests := []struct {
		name  string
		ips   string
		valid bool
	}{
		{"single IPv4 CIDR", "10.0.0.1/32", true},
		{"single IPv4 subnet", "10.0.0.0/24", true},
		{"multiple CIDRs", "10.0.0.1/32, 192.168.1.0/24", true},
		{"IPv6 CIDR", "fd00::/128", true},
		{"mixed IPv4 and IPv6", "10.0.0.1/32, fd00::1/128", true},
		{"empty string", "", false},
		{"newline injection", "10.0.0.1/32\nPublicKey = evil", false},
		{"carriage return injection", "10.0.0.1/32\rPublicKey = evil", false},
		{"not a CIDR", "10.0.0.1", false},
		{"garbage", "not-an-ip/32", false},
		{"empty segment", "10.0.0.1/32,", false},
		{"just comma", ",", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isValidAllowedIPs(tc.ips)
			if got != tc.valid {
				t.Errorf("isValidAllowedIPs(%q) = %v, want %v", tc.ips, got, tc.valid)
			}
		})
	}
}

func TestRemovePeerFromConfig(t *testing.T) {
	// Test that trailing empty lines are trimmed after removal
	config := `[Interface]
PrivateKey = testkey
Address = 10.0.0.1/24

[Peer]
PublicKey = peer1key
AllowedIPs = 10.0.0.2/32

[Peer]
PublicKey = peer2key
AllowedIPs = 10.0.0.3/32
`
	result := removePeerFromConfig(config, "peer1key")

	// Should not have excessive trailing newlines
	if strings.HasSuffix(result, "\n\n") {
		t.Error("result should not end with multiple newlines")
	}
	// Should still end with a single newline
	if !strings.HasSuffix(result, "\n") {
		t.Error("result should end with a single newline")
	}
	// Should still contain peer2
	if !strings.Contains(result, "peer2key") {
		t.Error("result should still contain peer2key")
	}
	// Should not contain peer1
	if strings.Contains(result, "peer1key") {
		t.Error("result should not contain peer1key")
	}
}

func TestRemovePeerFromConfig_LastPeer(t *testing.T) {
	config := `[Interface]
PrivateKey = testkey
Address = 10.0.0.1/24

[Peer]
PublicKey = peer1key
AllowedIPs = 10.0.0.2/32
`
	result := removePeerFromConfig(config, "peer1key")

	// Should not have excessive trailing newlines
	if strings.HasSuffix(result, "\n\n") {
		t.Errorf("result should not end with multiple newlines, got: %q", result)
	}
	// Should still end with a single newline
	if !strings.HasSuffix(result, "\n") {
		t.Error("result should end with a single newline")
	}
	// Should still contain Interface section
	if !strings.Contains(result, "[Interface]") {
		t.Error("result should still contain [Interface]")
	}
}
