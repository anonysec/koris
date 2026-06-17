package wireguard

import "fmt"

// ClientConfig holds the information needed to generate a WireGuard client configuration file.
type ClientConfig struct {
	PrivateKey      string
	Address         string
	DNS             string
	ServerPublicKey string
	PresharedKey    string
	Endpoint        string
}

// GenerateClientConfig produces a complete WireGuard .conf file string
// for a client peer, given the client's private key, address (allowed_ips),
// DNS servers, the server's public key, preshared key, and server endpoint.
func GenerateClientConfig(cfg ClientConfig) string {
	return fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s
DNS = %s

[Peer]
PublicKey = %s
PresharedKey = %s
AllowedIPs = 0.0.0.0/0, ::/0
Endpoint = %s
PersistentKeepalive = 25
`, cfg.PrivateKey, cfg.Address, cfg.DNS, cfg.ServerPublicKey, cfg.PresharedKey, cfg.Endpoint)
}
