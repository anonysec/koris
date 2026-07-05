package protocols

import "fmt"

// --- Protocol implementations ---

// OpenVPN implements the Protocol interface for OpenVPN.
type OpenVPN struct{}

func (OpenVPN) Name() string        { return "openvpn" }
func (OpenVPN) DisplayName() string { return "OpenVPN" }
func (OpenVPN) DefaultPort() int    { return 1194 }
func (OpenVPN) ServiceUnit() string { return "openvpn@server" }
func (OpenVPN) ValidateConfig(cfg ProtocolConfig) error {
	return validatePort(cfg.Port)
}

// L2TP implements the Protocol interface for L2TP/IPsec.
type L2TP struct{}

func (L2TP) Name() string        { return "l2tp" }
func (L2TP) DisplayName() string { return "L2TP/IPsec" }
func (L2TP) DefaultPort() int    { return 1701 }
func (L2TP) ServiceUnit() string { return "xl2tpd" }
func (L2TP) ValidateConfig(cfg ProtocolConfig) error {
	return validatePort(cfg.Port)
}

// IKEv2 implements the Protocol interface for IKEv2.
type IKEv2 struct{}

func (IKEv2) Name() string        { return "ikev2" }
func (IKEv2) DisplayName() string { return "IKEv2" }
func (IKEv2) DefaultPort() int    { return 500 }
func (IKEv2) ServiceUnit() string { return "strongswan" }
func (IKEv2) ValidateConfig(cfg ProtocolConfig) error {
	return validatePort(cfg.Port)
}

// WireGuard implements the Protocol interface for WireGuard.
type WireGuard struct{}

func (WireGuard) Name() string        { return "wireguard" }
func (WireGuard) DisplayName() string { return "WireGuard" }
func (WireGuard) DefaultPort() int    { return 51820 }
func (WireGuard) ServiceUnit() string { return "wg-quick@wg0" }
func (WireGuard) ValidateConfig(cfg ProtocolConfig) error {
	return validatePort(cfg.Port)
}

// SSH implements the Protocol interface for SSH tunneling.
type SSH struct{}

func (SSH) Name() string        { return "ssh" }
func (SSH) DisplayName() string { return "SSH Tunnel" }
func (SSH) DefaultPort() int    { return 22 }
func (SSH) ServiceUnit() string { return "sshd" }
func (SSH) ValidateConfig(cfg ProtocolConfig) error {
	return validatePort(cfg.Port)
}

// CiscoIPSec implements the Protocol interface for Cisco IPSec.
type CiscoIPSec struct{}

func (CiscoIPSec) Name() string        { return "cisco_ipsec" }
func (CiscoIPSec) DisplayName() string { return "Cisco IPSec" }
func (CiscoIPSec) DefaultPort() int    { return 500 }
func (CiscoIPSec) ServiceUnit() string { return "strongswan" }
func (CiscoIPSec) ValidateConfig(cfg ProtocolConfig) error {
	return validatePort(cfg.Port)
}

// --- Helpers ---

func validatePort(port int) error {
	if port <= 0 || port >= 65536 {
		return fmt.Errorf("invalid port %d: must be between 1 and 65535", port)
	}
	return nil
}

// --- Default registry ---

// NewDefault creates a ProtocolManager with all supported protocols pre-registered.
func NewDefault() *ProtocolManager {
	m := New()
	m.Register(OpenVPN{})
	m.Register(L2TP{})
	m.Register(IKEv2{})
	m.Register(WireGuard{})
	m.Register(SSH{})
	m.Register(CiscoIPSec{})
	return m
}
