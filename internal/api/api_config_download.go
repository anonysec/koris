package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// handleConfigDownload handles GET /d/{sub_token}/{protocol}
// This is a safe, token-authenticated endpoint for VPN config downloads.
// No session cookie required — the sub_token in the URL authenticates the user.
// The URL is not easily guessable (24-char random token) and doesn't expose API structure.
func (s *Server) handleConfigDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	// Parse path: /d/{token}/{protocol}
	path := strings.TrimPrefix(r.URL.Path, "/d/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	token := parts[0]
	protocol := strings.TrimSuffix(parts[1], "/")

	// Authenticate via sub_token
	var username, status string
	err := s.DB.QueryRow(
		`SELECT username, status FROM customers WHERE sub_token=$1 AND deleted_at IS NULL LIMIT 1`,
		token,
	).Scan(&username, &status)
	if err != nil || status == "disabled" || status == "deleted" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	nodeID, _ := strconv.ParseInt(r.URL.Query().Get("node_id"), 10, 64)

	switch strings.ToLower(protocol) {
	case "openvpn", "openvpn-udp", "openvpn.ovpn":
		profile := s.openVPNProfile(username, r, nodeID)
		_, _, _, nodeName := s.openVPNEndpointNode(r, nodeID)
		filename := safeFilename(nodeName) + ".ovpn"
		if filename == ".ovpn" {
			filename = "vpn.ovpn"
		}
		w.Header().Set("Content-Type", "application/x-openvpn-profile; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename*=UTF-8''%s`, url.PathEscape(filename)))
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = w.Write([]byte(profile))

	case "openvpn-tcp", "openvpn-tcp.ovpn":
		profile := s.openVPNProfileTCP(username, r, nodeID)
		_, _, _, nodeName := s.openVPNEndpointNode(r, nodeID)
		filename := safeFilename(nodeName) + "-TCP.ovpn"
		if filename == "-TCP.ovpn" {
			filename = "vpn-TCP.ovpn"
		}
		w.Header().Set("Content-Type", "application/x-openvpn-profile; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename*=UTF-8''%s`, url.PathEscape(filename)))
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = w.Write([]byte(profile))

	case "l2tp", "l2tp.mobileconfig":
		profile := s.l2tpMobileConfig(username, r, nodeID)
		_, _, _, nodeName := s.openVPNEndpointNode(r, nodeID)
		filename := safeFilename(username) + "-" + safeFilename(nodeName) + ".mobileconfig"
		if filename == "-.mobileconfig" {
			filename = "vpn-l2tp.mobileconfig"
		}
		w.Header().Set("Content-Type", "application/x-apple-aspen-config; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename*=UTF-8''%s`, url.PathEscape(filename)))
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = w.Write([]byte(profile))

	case "ikev2", "ikev2.mobileconfig":
		profile := s.ikev2MobileConfig(username, r, nodeID)
		_, _, _, nodeName := s.openVPNEndpointNode(r, nodeID)
		filename := safeFilename(username) + "-" + safeFilename(nodeName) + "-ikev2.mobileconfig"
		if filename == "--ikev2.mobileconfig" {
			filename = "vpn-ikev2.mobileconfig"
		}
		w.Header().Set("Content-Type", "application/x-apple-aspen-config; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename*=UTF-8''%s`, url.PathEscape(filename)))
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = w.Write([]byte(profile))

	case "wireguard", "wg", "wireguard.conf":
		config := s.wireguardConfig(username, nodeID)
		if config == "" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		filename := safeFilename(username) + "-wg.conf"
		w.Header().Set("Content-Type", "application/x-wireguard-profile; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename*=UTF-8''%s`, url.PathEscape(filename)))
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = w.Write([]byte(config))

	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}
