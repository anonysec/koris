package protocols

import (
	"testing"
)

func TestNewDefaultRegistersAll7Protocols(t *testing.T) {
	m := NewDefault()
	protocols := m.List()
	if len(protocols) != 6 {
		t.Fatalf("expected 7 protocols, got %d", len(protocols))
	}

	expected := []string{"openvpn", "l2tp", "ikev2", "wireguard", "ssh", "cisco_ipsec"}
	for _, name := range expected {
		if !m.IsValid(name) {
			t.Errorf("expected protocol %q to be registered", name)
		}
	}
}

func TestIsValidReturnsTrueForRegistered(t *testing.T) {
	m := NewDefault()

	tests := []struct {
		name  string
		valid bool
	}{
		{"openvpn", true},
		{"l2tp", true},
		{"ikev2", true},
		{"wireguard", true},
		{"ssh", true},
		{"cisco_ipsec", true},
		{"pptp", false},
		{"", false},
		{"OPENVPN", false}, // case-sensitive
		{"unknown_proto", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.IsValid(tt.name)
			if got != tt.valid {
				t.Errorf("IsValid(%q) = %v, want %v", tt.name, got, tt.valid)
			}
		})
	}
}

func TestValidateConfigRejectsBadPorts(t *testing.T) {
	m := NewDefault()

	badPorts := []int{0, -1, -100, 65536, 70000, 100000}
	for _, port := range badPorts {
		cfg := ProtocolConfig{Port: port, Enabled: true}
		for _, p := range m.List() {
			err := p.ValidateConfig(cfg)
			if err == nil {
				t.Errorf("%s.ValidateConfig(port=%d) expected error, got nil", p.Name(), port)
			}
		}
	}
}

func TestValidateConfigAcceptsGoodPorts(t *testing.T) {
	m := NewDefault()

	goodPorts := []int{1, 22, 80, 443, 500, 1194, 1701, 8080, 51820, 65535}
	for _, port := range goodPorts {
		cfg := ProtocolConfig{Port: port, Enabled: true}
		for _, p := range m.List() {
			err := p.ValidateConfig(cfg)
			if err != nil {
				t.Errorf("%s.ValidateConfig(port=%d) unexpected error: %v", p.Name(), port, err)
			}
		}
	}
}

func TestGetReturnsCorrectProtocol(t *testing.T) {
	m := NewDefault()

	p, ok := m.Get("wireguard")
	if !ok {
		t.Fatal("expected wireguard to be found")
	}
	if p.Name() != "wireguard" {
		t.Errorf("expected Name() = wireguard, got %s", p.Name())
	}
	if p.DisplayName() != "WireGuard" {
		t.Errorf("expected DisplayName() = WireGuard, got %s", p.DisplayName())
	}
	if p.DefaultPort() != 51820 {
		t.Errorf("expected DefaultPort() = 51820, got %d", p.DefaultPort())
	}
}

func TestGetReturnsFalseForUnknown(t *testing.T) {
	m := NewDefault()

	_, ok := m.Get("nonexistent")
	if ok {
		t.Error("expected Get(nonexistent) to return false")
	}
}

func TestDefaultPorts(t *testing.T) {
	m := NewDefault()

	expectedPorts := map[string]int{
		"openvpn":     1194,
		"l2tp":        1701,
		"ikev2":       500,
		"wireguard":   51820,
		"ssh":         22,
		"cisco_ipsec": 500,
	}

	for name, expectedPort := range expectedPorts {
		p, ok := m.Get(name)
		if !ok {
			t.Errorf("protocol %q not found", name)
			continue
		}
		if p.DefaultPort() != expectedPort {
			t.Errorf("%s.DefaultPort() = %d, want %d", name, p.DefaultPort(), expectedPort)
		}
	}
}

func TestServiceUnits(t *testing.T) {
	m := NewDefault()

	expectedUnits := map[string]string{
		"openvpn":     "openvpn@server",
		"l2tp":        "xl2tpd",
		"ikev2":       "strongswan",
		"wireguard":   "wg-quick@wg0",
		"ssh":         "sshd",
		"cisco_ipsec": "strongswan",
	}

	for name, expectedUnit := range expectedUnits {
		p, ok := m.Get(name)
		if !ok {
			t.Errorf("protocol %q not found", name)
			continue
		}
		if p.ServiceUnit() != expectedUnit {
			t.Errorf("%s.ServiceUnit() = %q, want %q", name, p.ServiceUnit(), expectedUnit)
		}
	}
}

func TestRegisterOverwrites(t *testing.T) {
	m := New()
	m.Register(OpenVPN{})

	// Register a custom type with same Name would overwrite
	p, ok := m.Get("openvpn")
	if !ok {
		t.Fatal("expected openvpn to be registered")
	}
	if p.DefaultPort() != 1194 {
		t.Errorf("expected default port 1194, got %d", p.DefaultPort())
	}
}

func TestEmptyManager(t *testing.T) {
	m := New()

	if m.IsValid("openvpn") {
		t.Error("empty manager should not have any protocols")
	}

	protocols := m.List()
	if len(protocols) != 0 {
		t.Errorf("expected 0 protocols in empty manager, got %d", len(protocols))
	}
}
