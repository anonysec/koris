package protocols

import "encoding/json"

// Protocol defines the interface that every VPN protocol implementation must satisfy.
type Protocol interface {
	// Name returns the protocol identifier (e.g., "openvpn", "wireguard").
	Name() string
	// DisplayName returns a human-readable protocol name (e.g., "OpenVPN").
	DisplayName() string
	// DefaultPort returns the default listening port for this protocol.
	DefaultPort() int
	// ServiceUnit returns the systemd service unit name (e.g., "openvpn@server").
	ServiceUnit() string
	// ValidateConfig checks whether the given config is valid for this protocol.
	ValidateConfig(cfg ProtocolConfig) error
}

// ProtocolConfig holds generic per-node protocol configuration.
type ProtocolConfig struct {
	Port      int             `json:"port"`
	Network   string          `json:"network"` // "tcp", "udp", or "both"
	Enabled   bool            `json:"enabled"`
	ExtraJSON json.RawMessage `json:"extra,omitempty"` // protocol-specific settings
}

// ProtocolManager is a registry of available protocol implementations.
type ProtocolManager struct {
	protocols map[string]Protocol
}

// New creates an empty ProtocolManager ready for protocol registration.
func New() *ProtocolManager {
	return &ProtocolManager{
		protocols: make(map[string]Protocol),
	}
}

// Register adds a protocol implementation to the manager.
// If a protocol with the same name already exists, it is overwritten.
func (m *ProtocolManager) Register(p Protocol) {
	m.protocols[p.Name()] = p
}

// Get returns the protocol with the given name and whether it was found.
func (m *ProtocolManager) Get(name string) (Protocol, bool) {
	p, ok := m.protocols[name]
	return p, ok
}

// List returns all registered protocols in no guaranteed order.
func (m *ProtocolManager) List() []Protocol {
	out := make([]Protocol, 0, len(m.protocols))
	for _, p := range m.protocols {
		out = append(out, p)
	}
	return out
}

// IsValid reports whether a protocol with the given name is registered.
func (m *ProtocolManager) IsValid(name string) bool {
	_, ok := m.protocols[name]
	return ok
}
