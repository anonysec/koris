package api

import (
	"KorisPanel/panel/internal/auth"
	"KorisPanel/panel/internal/wireguard"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func (s *Server) portalProfileDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	username, ok := s.currentCustomer(r)
	if !ok {
		writeJSONCode(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "unauthorized"})
		return
	}
	path := r.URL.Path
	switch {
	case strings.HasSuffix(path, "/openvpn-passwordless.ovpn"):
		nodeID, _ := strconv.ParseInt(r.URL.Query().Get("node_id"), 10, 64)
		profile := s.openVPNProfilePasswordless(username, r, nodeID)
		_, _, _, nodeName := s.openVPNEndpointNode(r, nodeID)
		nodeBase := safeFilename(nodeName)
		if nodeBase == "" {
			nodeBase = "vpn"
		}
		filename := safeFilename(username) + "-" + nodeBase + ".ovpn"
		w.Header().Set("Content-Type", "application/x-openvpn-profile; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename*=UTF-8''`+url.PathEscape(filename))
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(profile))
	case strings.HasSuffix(path, "/openvpn-tcp.ovpn"):
		nodeID, _ := strconv.ParseInt(r.URL.Query().Get("node_id"), 10, 64)
		profile := s.openVPNProfileTCP(username, r, nodeID)
		_, _, _, nodeName := s.openVPNEndpointNode(r, nodeID)
		nodeBase := safeFilename(nodeName)
		if nodeBase == "" {
			nodeBase = "vpn"
		}
		filename := nodeBase + "-TCP.ovpn"
		w.Header().Set("Content-Type", "application/x-openvpn-profile; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename*=UTF-8''`+url.PathEscape(filename))
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(profile))
	case strings.HasSuffix(path, "/openvpn.ovpn"):
		nodeID, _ := strconv.ParseInt(r.URL.Query().Get("node_id"), 10, 64)
		passwordless := r.URL.Query().Get("passwordless") == "true"
		var profile string
		if passwordless && s.canUsePasswordless(username) {
			profile = s.openVPNProfilePasswordless(username, r, nodeID)
		} else {
			profile = s.openVPNProfile(username, r, nodeID)
		}
		_, _, _, nodeName := s.openVPNEndpointNode(r, nodeID)
		nodeBase := safeFilename(nodeName)
		if nodeBase == "" {
			nodeBase = "vpn"
		}
		// Passwordless configs are per-user; standard OpenVPN is generic (node name only)
		var filename string
		if passwordless {
			filename = safeFilename(username) + "-" + nodeBase + ".ovpn"
		} else {
			filename = nodeBase + ".ovpn"
		}
		w.Header().Set("Content-Type", "application/x-openvpn-profile; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename*=UTF-8''`+url.PathEscape(filename))
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(profile))
	case strings.HasSuffix(path, "/l2tp.mobileconfig"):
		nodeID, _ := strconv.ParseInt(r.URL.Query().Get("node_id"), 10, 64)
		profile := s.l2tpMobileConfig(username, r, nodeID)
		_, _, _, nodeName := s.openVPNEndpointNode(r, nodeID)
		nodeBase := safeFilename(nodeName)
		if nodeBase == "" {
			nodeBase = "vpn"
		}
		// mobileconfig embeds username — always per-user
		filename := safeFilename(username) + "-" + nodeBase + ".mobileconfig"
		w.Header().Set("Content-Type", "application/x-apple-aspen-config; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename*=UTF-8''`+url.PathEscape(filename))
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(profile))
	case strings.HasSuffix(path, "/ikev2.mobileconfig"):
		nodeID, _ := strconv.ParseInt(r.URL.Query().Get("node_id"), 10, 64)
		profile := s.ikev2MobileConfig(username, r, nodeID)
		_, _, _, nodeName := s.openVPNEndpointNode(r, nodeID)
		nodeBase := safeFilename(nodeName)
		if nodeBase == "" {
			nodeBase = "vpn"
		}
		// mobileconfig embeds username — always per-user
		filename := safeFilename(username) + "-" + nodeBase + "-ikev2.mobileconfig"
		w.Header().Set("Content-Type", "application/x-apple-aspen-config; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename*=UTF-8''`+url.PathEscape(filename))
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(profile))
	case strings.HasSuffix(path, "/wireguard.conf"):
		nodeID, _ := strconv.ParseInt(r.URL.Query().Get("node_id"), 10, 64)
		s.portalWireguardConfByNode(w, r, username, nodeID)
	default:
		writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found"})
	}
}

func (s *Server) openVPNEndpoint(r *http.Request) (host string, port int, proto string, nodeName string) {
	port = 1194
	proto = "udp"
	_ = s.DB.QueryRow(`SELECT openvpn_port,openvpn_protocol FROM vpn_core_settings WHERE id=1`).Scan(&port, &proto)
	var address string
	_ = s.DB.QueryRow(`SELECT name, address FROM knode_connections WHERE enabled=TRUE ORDER BY CASE status WHEN 'online' THEN 0 WHEN 'stale' THEN 1 ELSE 2 END, id ASC LIMIT 1`).Scan(&nodeName, &address)
	host = strings.TrimSpace(address)
	if host == "" {
		host = r.Host
		if strings.Contains(host, ":") {
			host = strings.Split(host, ":")[0]
		}
	}
	if proto == "" {
		proto = "udp"
	}
	if port <= 0 {
		port = 1194
	}
	return host, port, proto, nodeName
}

func (s *Server) openVPNProfile(username string, r *http.Request, nodeID int64) string {
	return s.openVPNProfileWithAuth(username, r, nodeID, true)
}

func (s *Server) openVPNProfilePasswordless(username string, r *http.Request, nodeID int64) string {
	return s.openVPNProfileWithAuth(username, r, nodeID, false)
}

// openVPNProfileTCP generates a TCP-based OpenVPN config on port 443.
// Uses the user's preferred node as primary, with backup nodes as fallback.
func (s *Server) openVPNProfileTCP(username string, r *http.Request, nodeID int64) string {
	host, _, _, nodeName := s.openVPNEndpointNode(r, nodeID)
	if nodeName == "" {
		nodeName = host
	}
	caBlock := inlineOpenVPNBlockFromContent("ca", s.openVPNCACert(nodeID))
	tlsCryptBlock := inlineOpenVPNBlockFromContent("tls-crypt", s.openVPNTLSCryptKey(nodeID))

	// Get TCP port from node config or default to 8443
	tcpPort := 8443
	if nodeID > 0 {
		_ = s.DB.QueryRow(`SELECT port FROM node_vpn_configs WHERE node_id=$1 AND protocol='openvpn-tcp' AND enabled=TRUE LIMIT 1`, nodeID).Scan(&tcpPort)
	}

	// Build remote lines for TCP using domain bindings if available.
	domainEndpoints := s.protocolDomainEndpoints(nodeID, "openvpn-tcp")

	var remoteLines string
	if len(domainEndpoints) > 0 {
		// Use domain bindings in failover position order
		for i, ep := range domainEndpoints {
			if i == 0 {
				remoteLines = fmt.Sprintf("remote %s %d tcp", ep.DomainName, tcpPort)
			} else {
				remoteLines += fmt.Sprintf("\nremote %s %d tcp", ep.DomainName, tcpPort)
			}
		}
	} else {
		// Fallback: no domain bindings — use node endpoint
		remoteLines = fmt.Sprintf("remote %s %d tcp", host, tcpPort)

		// Get user's preferred node — put it first if different from default
		var preferredNodeID int64
		_ = s.DB.QueryRow(`SELECT COALESCE(preferred_node_id, 0) FROM customers WHERE username=$1 AND deleted_at IS NULL`, username).Scan(&preferredNodeID)
		if preferredNodeID > 0 && preferredNodeID != nodeID {
			var prefIP string
			if s.DB.QueryRow(`SELECT address FROM knode_connections WHERE id=$1 AND enabled=TRUE`, preferredNodeID).Scan(&prefIP) == nil {
				prefHost := strings.TrimSpace(prefIP)
				if prefHost != "" && prefHost != host {
					// Preferred node goes first
					remoteLines = fmt.Sprintf("remote %s %d tcp\nremote %s %d tcp", prefHost, tcpPort, host, tcpPort)
				}
			}
		}
	}

	// Add other active nodes as backup
	var preferredForBackup int64
	if len(domainEndpoints) == 0 {
		_ = s.DB.QueryRow(`SELECT COALESCE(preferred_node_id, 0) FROM customers WHERE username=$1 AND deleted_at IS NULL`, username).Scan(&preferredForBackup)
	}
	rows, err := s.DB.Query(`
		SELECT n.address
		FROM knode_connections n
		JOIN node_vpn_configs c ON c.node_id = n.id AND c.protocol = 'openvpn' AND c.enabled = TRUE
		WHERE n.enabled = TRUE AND n.id <> $1 AND n.id <> $2
		ORDER BY n.id`, nodeID, preferredForBackup)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ip string
			if rows.Scan(&ip) == nil {
				backupHost := strings.TrimSpace(ip)
				if backupHost != "" && backupHost != host {
					remoteLines += fmt.Sprintf("\nremote %s %d tcp", backupHost, tcpPort)
				}
			}
		}
	}

	return fmt.Sprintf(`# KorisPanel OpenVPN TCP Profile
# User: %s
# Node: %s
# Generated: %s
# TCP mode — supports node selection via portal
client
dev tun
%s
resolv-retry infinite
nobind
persist-key
persist-tun
remote-cert-tls server
setenv CLIENT_CERT 0
auth-user-pass
auth-nocache
auth SHA256
data-ciphers AES-256-GCM:AES-128-GCM:CHACHA20-POLY1305
data-ciphers-fallback AES-256-GCM
verb 3
pull
%s%s`, username, nodeName, time.Now().UTC().Format(time.RFC3339), remoteLines, caBlock, tlsCryptBlock)
}

// canUsePasswordless checks if a customer is allowed to generate passwordless configs.
// Requires: global setting enabled AND customer's plan allows passwordless.
func (s *Server) canUsePasswordless(username string) bool {
	// Check global setting
	var enabled string
	_ = s.DB.QueryRow(`SELECT setting_value FROM panel_settings WHERE setting_key='passwordless_configs_enabled'`).Scan(&enabled)
	if enabled != "true" {
		return false
	}
	// Check per-plan setting
	var allowPasswordless int
	err := s.DB.QueryRow(`SELECT COALESCE(p.allow_passwordless, 0) FROM customers c JOIN plans p ON p.id = c.plan_id WHERE c.username = $1 AND c.deleted_at IS NULL LIMIT 1`, username).Scan(&allowPasswordless)
	if err != nil {
		return false
	}
	return allowPasswordless == 1
}

func (s *Server) openVPNProfileWithAuth(username string, r *http.Request, nodeID int64, withAuth bool) string {
	host, port, proto, nodeName := s.openVPNEndpointNode(r, nodeID)
	if nodeName == "" {
		nodeName = host
	}

	// Resolve actual nodeID if 0 was passed (picks the first available node)
	resolvedNodeID := nodeID
	if resolvedNodeID <= 0 {
		_ = s.DB.QueryRow(`SELECT id FROM knode_connections WHERE enabled=TRUE ORDER BY CASE status WHEN 'online' THEN 0 WHEN 'stale' THEN 1 ELSE 2 END, id ASC LIMIT 1`).Scan(&resolvedNodeID)
	}

	caBlock := inlineOpenVPNBlockFromContent("ca", s.openVPNCACert(resolvedNodeID))
	tlsCryptBlock := inlineOpenVPNBlockFromContent("tls-crypt", s.openVPNTLSCryptKey(resolvedNodeID))

	authLine := "auth-user-pass\n"
	authComment := "# Login with your VPN username/password when OpenVPN asks for credentials."
	if !withAuth {
		// Certificate-based (passwordless) mode: embed client cert+key.
		clientCert, clientKey := s.ensureClientCert(username, resolvedNodeID)
		authLine = ""
		authComment = "# Certificate-based auth — no password needed."
		if clientCert != "" {
			caBlock += inlineOpenVPNBlockFromContent("cert", clientCert)
		}
		if clientKey != "" {
			caBlock += inlineOpenVPNBlockFromContent("key", clientKey)
		}
	}

	// Build remote lines using domain bindings in failover position order.
	// If domain bindings exist for this protocol on this node, use them instead of raw IPs.
	bindingProtocol := "openvpn-udp"
	if proto == "tcp" {
		bindingProtocol = "openvpn-tcp"
	}
	domainEndpoints := s.protocolDomainEndpoints(resolvedNodeID, bindingProtocol)

	var remoteLines string
	if len(domainEndpoints) > 0 {
		// Use domain bindings: primary is first active domain, rest are backups in position order
		for i, ep := range domainEndpoints {
			if i == 0 {
				remoteLines = fmt.Sprintf("remote %s %d %s", ep.DomainName, port, proto)
			} else {
				remoteLines += fmt.Sprintf("\nremote %s %d %s", ep.DomainName, port, proto)
			}
		}
	} else {
		// Fallback: no domain bindings — use node's endpoint (resolved by openVPNEndpointNode)
		remoteLines = fmt.Sprintf("remote %s %d %s", host, port, proto)

		// Legacy fallback: add backup domains from this node's backup_domains field
		if resolvedNodeID > 0 {
			var backupDomains *string
			_ = s.DB.QueryRow(`SELECT backup_domains FROM knode_connections WHERE id=$1`, resolvedNodeID).Scan(&backupDomains)
			if backupDomains != nil && strings.TrimSpace(*backupDomains) != "" {
				for _, d := range strings.Split(*backupDomains, ",") {
					d = strings.TrimSpace(d)
					if d != "" && d != host {
						remoteLines += fmt.Sprintf("\nremote %s %d %s", d, port, proto)
					}
				}
			}
		}
	}

	// Add backup remotes from other nodes (prefer domain over IP)
	rows, err := s.DB.Query(`
		SELECT COALESCE(NULLIF(TRIM(n.domain),''), n.address) AS endpoint
		FROM knode_connections n
		JOIN node_vpn_configs c ON c.node_id = n.id AND c.protocol = 'openvpn' AND c.enabled = TRUE
		WHERE n.enabled = TRUE AND n.id <> $1
		ORDER BY n.id`, resolvedNodeID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var endpoint string
			if rows.Scan(&endpoint) == nil {
				endpoint = strings.TrimSpace(endpoint)
				if endpoint != "" && endpoint != host {
					remoteLines += fmt.Sprintf("\nremote %s %d %s", endpoint, port, proto)
				}
			}
		}
	}

	return fmt.Sprintf(`# KorisPanel generated OpenVPN profile
# User: %s
# Node: %s
# Generated: %s
%s
client
dev tun
%s
resolv-retry infinite
nobind
persist-key
persist-tun
remote-cert-tls server
setenv CLIENT_CERT 0
%sauth-nocache
auth SHA256
data-ciphers AES-256-GCM:AES-128-GCM:CHACHA20-POLY1305
data-ciphers-fallback AES-256-GCM
explicit-exit-notify 1
verb 3
pull
%s%s`, username, nodeName, time.Now().UTC().Format(time.RFC3339), authComment, remoteLines, authLine, caBlock, tlsCryptBlock)
}

func getenvFirst(envName string, paths ...string) string {
	if v := strings.TrimSpace(os.Getenv(envName)); v != "" {
		return v
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func inlineOpenVPNBlock(name, filePath string) string {
	if filePath == "" {
		return ""
	}
	b, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	content := strings.TrimSpace(string(b))
	if content == "" {
		return ""
	}
	return fmt.Sprintf("\n<%s>\n%s\n</%s>\n", name, content, name)
}

// inlineOpenVPNBlockFromContent wraps raw PEM content in OpenVPN inline block tags.
func inlineOpenVPNBlockFromContent(name, content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	return fmt.Sprintf("\n<%s>\n%s\n</%s>\n", name, content, name)
}

// openVPNCACert returns the CA certificate PEM content for the given node's OpenVPN core.
// It checks vpn_certificates table first (DB-backed, pushed from knode), then falls back
// to the filesystem paths for backward compatibility with co-located panel+OpenVPN setups.
func (s *Server) openVPNCACert(nodeID int64) string {
	// Priority 1: DB-backed cert from vpn_certificates (works with remote knode)
	var content string
	if nodeID > 0 {
		_ = s.DB.QueryRow(
			`SELECT content FROM vpn_certificates WHERE node_id=$1 AND type='ca' AND status='active' ORDER BY is_default DESC, id DESC LIMIT 1`,
			nodeID,
		).Scan(&content)
	}
	if strings.TrimSpace(content) == "" {
		// Try default (node_id=0 or is_default=true regardless of node)
		_ = s.DB.QueryRow(
			`SELECT content FROM vpn_certificates WHERE type='ca' AND status='active' ORDER BY is_default DESC, id DESC LIMIT 1`,
		).Scan(&content)
	}
	if strings.TrimSpace(content) != "" {
		return strings.TrimSpace(content)
	}

	// Priority 2: Filesystem fallback (legacy: panel and OpenVPN on same host)
	path := getenvFirst("PANEL_OPENVPN_CA_FILE", "/etc/openvpn/server/ca.crt", "/etc/openvpn/easy-rsa/pki/ca.crt")
	if path != "" {
		if b, err := os.ReadFile(path); err == nil {
			return strings.TrimSpace(string(b))
		}
	}
	return ""
}

// openVPNTLSCryptKey returns the tls-crypt key content for the given node's OpenVPN core.
// Same priority logic as openVPNCACert: DB first, filesystem fallback.
func (s *Server) openVPNTLSCryptKey(nodeID int64) string {
	var content string
	if nodeID > 0 {
		_ = s.DB.QueryRow(
			`SELECT content FROM vpn_certificates WHERE node_id=$1 AND type='tls-crypt' AND status='active' ORDER BY is_default DESC, id DESC LIMIT 1`,
			nodeID,
		).Scan(&content)
	}
	if strings.TrimSpace(content) == "" {
		_ = s.DB.QueryRow(
			`SELECT content FROM vpn_certificates WHERE type='tls-crypt' AND status='active' ORDER BY is_default DESC, id DESC LIMIT 1`,
		).Scan(&content)
	}
	if strings.TrimSpace(content) != "" {
		return strings.TrimSpace(content)
	}

	// Filesystem fallback
	path := getenvFirst("PANEL_OPENVPN_TLS_CRYPT_FILE", "/etc/openvpn/server/tc.key", "/etc/openvpn/server/tls-crypt.key", "/etc/openvpn/server/ta.key")
	if path != "" {
		if b, err := os.ReadFile(path); err == nil {
			return strings.TrimSpace(string(b))
		}
	}
	return ""
}

func safeFilename(s string) string {
	return strings.NewReplacer("/", "_", "\\", "_", " ", "_", "\x00", "_").Replace(s)
}

func (s *Server) l2tpMobileConfig(username string, r *http.Request, nodeID int64) string {
	host, _, _, _ := s.openVPNEndpointNode(r, nodeID)

	// Override with L2TP-specific domain binding if available
	if nodeID > 0 {
		if primary := s.protocolPrimaryDomain(nodeID, "l2tp"); primary != "" {
			host = primary
		}
	}

	if host == "" {
		host = r.Host
	}
	var psk string
	_ = s.DB.QueryRow(`SELECT COALESCE(ipsec_psk,'') FROM vpn_core_settings WHERE id=1`).Scan(&psk)
	psk = strings.TrimSpace(psk)
	uuidPayload := strings.ToLower(auth.RandomToken(8) + "-" + auth.RandomToken(4) + "-" + auth.RandomToken(4) + "-" + auth.RandomToken(4) + "-" + auth.RandomToken(12))
	uuidProfile := strings.ToLower(auth.RandomToken(8) + "-" + auth.RandomToken(4) + "-" + auth.RandomToken(4) + "-" + auth.RandomToken(4) + "-" + auth.RandomToken(12))
	pskData := base64.StdEncoding.EncodeToString([]byte(psk))
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>PayloadDescription</key>
			<string>Configures L2TP VPN</string>
			<key>PayloadDisplayName</key>
			<string>Koris L2TP</string>
			<key>PayloadIdentifier</key>
			<string>koris.vpn.l2tp.%s</string>
			<key>PayloadType</key>
			<string>com.apple.vpn.managed</string>
			<key>PayloadUUID</key>
			<string>%s</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
			<key>UserDefinedName</key>
			<string>Koris L2TP</string>
			<key>VPNType</key>
			<string>L2TP</string>
			<key>IPv4</key>
			<dict>
				<key>OverridePrimary</key>
				<integer>1</integer>
			</dict>
			<key>PPP</key>
			<dict>
				<key>AuthName</key>
				<string>%s</string>
				<key>CommRemoteAddress</key>
				<string>%s</string>
				<key>OnDemandEnabled</key>
				<integer>0</integer>
			</dict>
			<key>IPSec</key>
			<dict>
				<key>AuthenticationMethod</key>
				<string>SharedSecret</string>
				<key>SharedSecret</key>
				<data>%s</data>
			</dict>
		</dict>
	</array>
	<key>PayloadDisplayName</key>
	<string>Koris L2TP</string>
	<key>PayloadIdentifier</key>
	<string>koris.vpn.l2tp.profile.%s</string>
	<key>PayloadRemovalDisallowed</key>
	<false/>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>%s</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>`, username, uuidPayload, username, host, pskData, username, uuidProfile)
}

// portalWireguardConfByNode generates a WireGuard config for the customer's peer on the given node.
// If nodeID is 0, it uses any available peer for the customer.
func (s *Server) portalWireguardConfByNode(w http.ResponseWriter, r *http.Request, username string, nodeID int64) {
	// Get customer ID
	var customerID int64
	err := s.DB.QueryRow(`SELECT id FROM customers WHERE username=$1 AND deleted_at IS NULL LIMIT 1`, username).Scan(&customerID)
	if err != nil {
		writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found"})
		return
	}

	// Find the customer's peer for this node
	var peer WgPeer
	var query string
	var args []any
	if nodeID > 0 {
		query = `SELECT id, node_id, public_key, COALESCE(preshared_key,''),
		         COALESCE(private_key_encrypted,''), allowed_ips, COALESCE(endpoint,''), status
		         FROM wg_peers WHERE customer_id=$1 AND node_id=$2 AND status='active' LIMIT 1`
		args = []any{customerID, nodeID}
	} else {
		query = `SELECT id, node_id, public_key, COALESCE(preshared_key,''),
		         COALESCE(private_key_encrypted,''), allowed_ips, COALESCE(endpoint,''), status
		         FROM wg_peers WHERE customer_id=$1 AND status='active' LIMIT 1`
		args = []any{customerID}
	}
	err = s.DB.QueryRow(query, args...).Scan(
		&peer.ID, &peer.NodeID, &peer.PublicKey,
		&peer.PresharedKey, &peer.PrivateKeyEncrypted, &peer.AllowedIPs,
		&peer.Endpoint, &peer.Status)
	if err != nil {
		writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "no_wireguard_peer"})
		return
	}

	if peer.PrivateKeyEncrypted == "" {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "private_key_not_available"})
		return
	}

	// Get server config for this peer's node
	var extraJSON []byte
	err = s.DB.QueryRow(`
		SELECT COALESCE(extra_json,'{}')
		FROM node_vpn_configs WHERE node_id=$1 AND protocol='wireguard'`, peer.NodeID).Scan(&extraJSON)
	if err != nil {
		writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "wireguard_config_not_found"})
		return
	}

	var serverPublicKey, dns1, dns2, serverEndpoint string
	var gamingOptimize bool
	var extra map[string]any
	if err := json.Unmarshal(extraJSON, &extra); err == nil {
		if v, ok := extra["server_public_key"].(string); ok {
			serverPublicKey = v
		}
		if v, ok := extra["dns_1"].(string); ok {
			dns1 = v
		}
		if v, ok := extra["dns_2"].(string); ok {
			dns2 = v
		}
		if v, ok := extra["gaming_optimize"].(bool); ok {
			gamingOptimize = v
		}
	}

	// Get node endpoint
	var nodeIP, nodeDomain string
	var wgPort int
	_ = s.DB.QueryRow(`SELECT COALESCE(address,''), COALESCE(domain,'') FROM knode_connections WHERE id=$1`, peer.NodeID).Scan(&nodeIP, &nodeDomain)
	_ = s.DB.QueryRow(`SELECT port FROM node_vpn_configs WHERE node_id=$1 AND protocol='wireguard'`, peer.NodeID).Scan(&wgPort)

	if nodeDomain != "" {
		serverEndpoint = fmt.Sprintf("%s:%d", nodeDomain, wgPort)
	} else if nodeIP != "" {
		serverEndpoint = fmt.Sprintf("%s:%d", nodeIP, wgPort)
	}

	// Build DNS string
	dns := dns1
	if dns2 != "" {
		dns = dns1 + ", " + dns2
	}
	if dns == "" {
		dns = "1.1.1.1, 8.8.8.8"
	}

	// Generate config
	conf := wireguard.GenerateClientConfig(wireguard.ClientConfig{
		PrivateKey:      peer.PrivateKeyEncrypted,
		Address:         peer.AllowedIPs,
		DNS:             dns,
		ServerPublicKey: serverPublicKey,
		PresharedKey:    peer.PresharedKey,
		Endpoint:        serverEndpoint,
		GamingOptimize:  gamingOptimize,
	})

	// Serve as downloadable .conf file
	var nodeName string
	_ = s.DB.QueryRow(`SELECT COALESCE(name,'') FROM knode_connections WHERE id=$1`, peer.NodeID).Scan(&nodeName)
	if nodeName == "" {
		nodeName = fmt.Sprintf("node%d", peer.NodeID)
	}
	filename := fmt.Sprintf("KorisVPN-%s.conf", safeFilename(nodeName))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename*=UTF-8''`+url.PathEscape(filename))
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(conf))
}

// ensureClientCert checks if a client cert exists for the user on the node.
// If not, it calls the knode gRPC to generate one and stores it in the DB.
// Returns (certPEM, keyPEM). Empty strings if generation fails.
func (s *Server) ensureClientCert(username string, nodeID int64) (string, string) {
	// 1. Check DB for existing cert
	var clientCert, clientKey string
	_ = s.DB.QueryRow(
		`SELECT content FROM vpn_certificates WHERE node_id=$1 AND type='client-cert' AND status='active' AND username=$2 LIMIT 1`,
		nodeID, username,
	).Scan(&clientCert)
	_ = s.DB.QueryRow(
		`SELECT content FROM vpn_certificates WHERE node_id=$1 AND type='client-key' AND status='active' AND username=$2 LIMIT 1`,
		nodeID, username,
	).Scan(&clientKey)

	if clientCert != "" && clientKey != "" {
		return clientCert, clientKey
	}

	// 2. Check if user has a cert from another node (same CA = interchangeable)
	_ = s.DB.QueryRow(
		`SELECT content FROM vpn_certificates WHERE type='client-cert' AND status='active' AND username=$1 LIMIT 1`,
		username,
	).Scan(&clientCert)
	_ = s.DB.QueryRow(
		`SELECT content FROM vpn_certificates WHERE type='client-key' AND status='active' AND username=$1 LIMIT 1`,
		username,
	).Scan(&clientKey)

	if clientCert != "" && clientKey != "" {
		// Cache for this node too
		_, _ = s.DB.Exec(
			`INSERT INTO vpn_certificates (node_id, type, status, username, content, name, is_default, created_at)
			 VALUES ($1, 'client-cert', 'active', $2, $3, $4, FALSE, NOW()) ON CONFLICT DO NOTHING`,
			nodeID, username, clientCert, username+"-cert",
		)
		_, _ = s.DB.Exec(
			`INSERT INTO vpn_certificates (node_id, type, status, username, content, name, is_default, created_at)
			 VALUES ($1, 'client-key', 'active', $2, $3, $4, FALSE, NOW()) ON CONFLICT DO NOTHING`,
			nodeID, username, clientKey, username+"-key",
		)
		return clientCert, clientKey
	}

	// 3. No cert exists anywhere — generate via knode gRPC
	if s.ClientCertSvc == nil {
		return "", ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := s.ClientCertSvc.GenerateClientCert(ctx, nodeID, username)
	if err != nil {
		// Log but don't fail — user just won't get cert in profile
		return "", ""
	}

	// 3. Store cert and key in DB
	_, _ = s.DB.Exec(
		`INSERT INTO vpn_certificates (node_id, type, status, username, content, name, is_default, created_at)
		 VALUES ($1, 'client-cert', 'active', $2, $3, $4, FALSE, NOW())`,
		nodeID, username, result.CertPEM, username+"-cert",
	)
	_, _ = s.DB.Exec(
		`INSERT INTO vpn_certificates (node_id, type, status, username, content, name, is_default, created_at)
		 VALUES ($1, 'client-key', 'active', $2, $3, $4, FALSE, NOW())`,
		nodeID, username, result.KeyPEM, username+"-key",
	)

	// 4. Also update the CA cert in DB if it changed (knode is source of truth)
	if result.CAPEM != "" {
		var existingCA string
		_ = s.DB.QueryRow(`SELECT content FROM vpn_certificates WHERE node_id=$1 AND type='ca' AND status='active' LIMIT 1`, nodeID).Scan(&existingCA)
		if strings.TrimSpace(existingCA) != strings.TrimSpace(result.CAPEM) {
			// Update existing or insert new
			if existingCA != "" {
				_, _ = s.DB.Exec(`UPDATE vpn_certificates SET content=$1 WHERE node_id=$2 AND type='ca' AND status='active'`, result.CAPEM, nodeID)
			} else {
				_, _ = s.DB.Exec(
					`INSERT INTO vpn_certificates (node_id, type, status, content, name, is_default, created_at)
					 VALUES ($1, 'ca', 'active', $2, 'ca-cert', TRUE, NOW())`,
					nodeID, result.CAPEM,
				)
			}
		}
	}

	return result.CertPEM, result.KeyPEM
}
