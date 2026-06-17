package wireguard

import (
	"strings"
	"testing"
)

func TestGenerateClientConfig(t *testing.T) {
	cfg := ClientConfig{
		PrivateKey:      "cGVlcl9wcml2YXRlX2tleV9iYXNlNjRfZW5jb2RlZA==",
		Address:         "10.0.0.2/32",
		DNS:             "1.1.1.1, 8.8.8.8",
		ServerPublicKey: "c2VydmVyX3B1YmxpY19rZXlfYmFzZTY0X2VuY29kZWQ=",
		PresharedKey:    "cHJlc2hhcmVkX2tleV9iYXNlNjRfZW5jb2RlZF9oZXJl",
		Endpoint:        "vpn.example.com:51820",
	}

	result := GenerateClientConfig(cfg)

	// Verify [Interface] section
	if !strings.Contains(result, "[Interface]") {
		t.Error("missing [Interface] section")
	}
	if !strings.Contains(result, "PrivateKey = "+cfg.PrivateKey) {
		t.Error("missing or incorrect PrivateKey")
	}
	if !strings.Contains(result, "Address = "+cfg.Address) {
		t.Error("missing or incorrect Address")
	}
	if !strings.Contains(result, "DNS = "+cfg.DNS) {
		t.Error("missing or incorrect DNS")
	}

	// Verify [Peer] section
	if !strings.Contains(result, "[Peer]") {
		t.Error("missing [Peer] section")
	}
	if !strings.Contains(result, "PublicKey = "+cfg.ServerPublicKey) {
		t.Error("missing or incorrect server PublicKey")
	}
	if !strings.Contains(result, "PresharedKey = "+cfg.PresharedKey) {
		t.Error("missing or incorrect PresharedKey")
	}
	if !strings.Contains(result, "AllowedIPs = 0.0.0.0/0, ::/0") {
		t.Error("missing or incorrect AllowedIPs")
	}
	if !strings.Contains(result, "Endpoint = "+cfg.Endpoint) {
		t.Error("missing or incorrect Endpoint")
	}
	if !strings.Contains(result, "PersistentKeepalive = 25") {
		t.Error("missing PersistentKeepalive")
	}
}

func TestGenerateClientConfigFormat(t *testing.T) {
	cfg := ClientConfig{
		PrivateKey:      "abc123privatekey==",
		Address:         "10.8.0.5/24",
		DNS:             "9.9.9.9",
		ServerPublicKey: "xyz789serverpubkey==",
		PresharedKey:    "psk000presharedkey==",
		Endpoint:        "192.168.1.1:51820",
	}

	result := GenerateClientConfig(cfg)

	// Verify the [Interface] section comes before [Peer]
	ifaceIdx := strings.Index(result, "[Interface]")
	peerIdx := strings.Index(result, "[Peer]")
	if ifaceIdx < 0 || peerIdx < 0 {
		t.Fatal("missing required sections")
	}
	if ifaceIdx >= peerIdx {
		t.Error("[Interface] should come before [Peer]")
	}

	// Verify there is a blank line separating sections
	between := result[ifaceIdx:peerIdx]
	if !strings.Contains(between, "\n\n") {
		t.Error("expected blank line between [Interface] and [Peer] sections")
	}
}
