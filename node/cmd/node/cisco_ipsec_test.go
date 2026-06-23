package main

import (
	"strings"
	"testing"
)

func TestGenerateCiscoConf(t *testing.T) {
	tests := []struct {
		name    string
		leftID  string
		network string
		dns     string
		expect  []string
	}{
		{
			name:    "basic config",
			leftID:  "vpn.example.com",
			network: "10.10.0.0/24",
			dns:     "8.8.8.8, 8.8.4.4",
			expect: []string{
				"conn cisco-ipsec",
				"keyexchange=ikev1",
				"authby=xauthpsk",
				"xauth=server",
				"left=%defaultroute",
				"leftid=vpn.example.com",
				"leftsubnet=0.0.0.0/0",
				"rightsourceip=10.10.0.0/24",
				"rightdns=8.8.8.8, 8.8.4.4",
				"right=%any",
				"rightauth=eap-radius",
				"auto=add",
				"aggressive=yes",
				"fragmentation=yes",
				"dpdaction=clear",
				"dpddelay=30s",
				"dpdtimeout=120s",
			},
		},
		{
			name:    "custom dns and network",
			leftID:  "node1.vpn.io",
			network: "172.16.0.0/16",
			dns:     "1.1.1.1",
			expect: []string{
				"leftid=node1.vpn.io",
				"rightsourceip=172.16.0.0/16",
				"rightdns=1.1.1.1",
				"keyexchange=ikev1",
				"rightauth=eap-radius",
			},
		},
		{
			name:    "ike and esp ciphers present",
			leftID:  "test.local",
			network: "10.0.0.0/24",
			dns:     "8.8.8.8",
			expect: []string{
				"ike=aes256-sha256-modp2048,aes128-sha1-modp2048!",
				"esp=aes256-sha256,aes128-sha1!",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := generateCiscoConf(tc.leftID, tc.network, tc.dns)
			for _, directive := range tc.expect {
				if !strings.Contains(result, directive) {
					t.Errorf("generateCiscoConf() missing directive %q\nGot:\n%s", directive, result)
				}
			}
		})
	}
}

func TestGenerateCiscoConf_Header(t *testing.T) {
	result := generateCiscoConf("vpn.test.com", "10.0.0.0/24", "8.8.8.8")
	if !strings.HasPrefix(result, "# Cisco IPSec") {
		t.Errorf("expected config to start with comment header, got: %q", result[:40])
	}
}

func TestGenerateCiscoSecrets(t *testing.T) {
	tests := []struct {
		name   string
		psk    string
		expect string
	}{
		{
			name:   "simple psk",
			psk:    "mysecretkey",
			expect: `: PSK "mysecretkey"`,
		},
		{
			name:   "psk with special chars",
			psk:    "p@ss!w0rd#123",
			expect: `: PSK "p@ss!w0rd#123"`,
		},
		{
			name:   "long psk",
			psk:    "abcdefghijklmnopqrstuvwxyz0123456789ABCDEF",
			expect: `: PSK "abcdefghijklmnopqrstuvwxyz0123456789ABCDEF"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := generateCiscoSecrets(tc.psk)
			if !strings.Contains(result, tc.expect) {
				t.Errorf("generateCiscoSecrets(%q) missing %q\nGot: %s", tc.psk, tc.expect, result)
			}
		})
	}
}

func TestGenerateCiscoSecrets_Format(t *testing.T) {
	result := generateCiscoSecrets("testkey")

	// Should have the comment header
	if !strings.HasPrefix(result, "# Cisco IPSec PSK") {
		t.Errorf("expected header comment, got: %q", result)
	}

	// Should end with newline
	if !strings.HasSuffix(result, "\n") {
		t.Error("expected trailing newline")
	}

	// Should contain exactly one ': PSK' line
	lines := strings.Split(strings.TrimSpace(result), "\n")
	pskLines := 0
	for _, line := range lines {
		if strings.Contains(line, ": PSK") {
			pskLines++
		}
	}
	if pskLines != 1 {
		t.Errorf("expected exactly 1 PSK line, got %d", pskLines)
	}
}

func TestMtgServiceName(t *testing.T) {
	tests := []struct {
		name   string
		port   int
		expect string
	}{
		{"default port", 443, "mtg-443.service"},
		{"custom port", 8443, "mtg-8443.service"},
		{"low port", 80, "mtg-80.service"},
		{"high port", 65535, "mtg-65535.service"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mtgServiceName(tc.port)
			if got != tc.expect {
				t.Errorf("mtgServiceName(%d) = %q, want %q", tc.port, got, tc.expect)
			}
		})
	}
}
