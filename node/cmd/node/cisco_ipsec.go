package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"KorisPanel/node/internal/logger"
)

const (
	ciscoConfPath       = "/etc/ipsec.d/cisco.conf"
	ciscoSecretsPath    = "/etc/ipsec.d/cisco.secrets"
	ipsecBinaryPath     = "/usr/sbin/ipsec"
	eapRadiusConfPath   = "/etc/strongswan.d/charon/eap-radius.conf"
	freeradiusClientDir = "/etc/freeradius/3.0/clients.conf.d"
)

// executeCiscoIPSecDeploy handles the "cisco_ipsec_deploy" task:
// ensures strongSwan is installed, generates IKEv1+XAUTH config files,
// configures the EAP-RADIUS plugin for RADIUS-based authentication and accounting,
// and restarts the ipsec service.
func executeCiscoIPSecDeploy(payload map[string]any, log *logger.Logger) (string, map[string]any, string) {
	psk := payloadStr(payload, "psk")
	network := payloadStr(payload, "network")
	leftID := payloadStr(payload, "left_id")
	dns := payloadStr(payload, "dns")
	radiusServer := payloadStr(payload, "radius_server")
	radiusSecret := payloadStr(payload, "radius_secret")

	if psk == "" {
		return "failed", map[string]any{}, "psk is required"
	}
	if network == "" {
		return "failed", map[string]any{}, "network is required"
	}
	if leftID == "" {
		return "failed", map[string]any{}, "left_id is required"
	}
	if dns == "" {
		dns = "8.8.8.8, 8.8.4.4"
	}
	if radiusServer == "" {
		radiusServer = "127.0.0.1"
	}
	if radiusSecret == "" {
		radiusSecret = "testing123"
	}

	// Ensure strongSwan is installed
	if _, err := os.Stat(ipsecBinaryPath); err != nil {
		return "failed", map[string]any{}, "strongSwan not installed: /usr/sbin/ipsec not found"
	}

	// Ensure /etc/ipsec.d/ directory exists
	if err := os.MkdirAll("/etc/ipsec.d", 0755); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("create ipsec.d dir: %s", err.Error())
	}

	// Generate cisco.conf with IKEv1+XAUTH connection using RADIUS
	confContent := generateCiscoConf(leftID, network, dns)
	if err := os.WriteFile(ciscoConfPath, []byte(confContent), 0644); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("write cisco.conf: %s", err.Error())
	}

	// Generate cisco.secrets with PSK
	secretsContent := generateCiscoSecrets(psk)
	if err := os.WriteFile(ciscoSecretsPath, []byte(secretsContent), 0600); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("write cisco.secrets: %s", err.Error())
	}

	// Generate EAP-RADIUS plugin config for strongSwan
	if err := writeEAPRadiusConf(radiusServer, radiusSecret); err != nil {
		return "failed", map[string]any{}, fmt.Sprintf("write eap-radius.conf: %s", err.Error())
	}

	// Configure FreeRADIUS to accept this node as a NAS client
	if err := configureFreeRADIUSClient(radiusServer, radiusSecret, leftID, log); err != nil {
		// Non-fatal: FreeRADIUS may be on a different host
		log.Warn("freeradius client config skipped", map[string]any{"error": err.Error()})
	}

	// Restart ipsec service
	out, err := exec.Command("ipsec", "restart").CombinedOutput()
	if err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("ipsec restart: %s", err.Error())
	}

	log.Info("cisco ipsec deployed", map[string]any{
		"left_id":       leftID,
		"network":       network,
		"radius_server": radiusServer,
	})
	return "succeeded", map[string]any{
		"left_id":           leftID,
		"network":           network,
		"conf":              ciscoConfPath,
		"secrets":           ciscoSecretsPath,
		"eap_radius_conf":   eapRadiusConfPath,
		"radius_server":     radiusServer,
		"radius_accounting": true,
	}, ""
}

// executeCiscoIPSecRemove handles the "cisco_ipsec_remove" task:
// removes cisco config files, EAP-RADIUS config, and restarts the ipsec service.
func executeCiscoIPSecRemove(payload map[string]any, log *logger.Logger) (string, map[string]any, string) {
	// Remove cisco.conf
	if err := os.Remove(ciscoConfPath); err != nil && !os.IsNotExist(err) {
		return "failed", map[string]any{}, fmt.Sprintf("remove cisco.conf: %s", err.Error())
	}

	// Remove cisco.secrets
	if err := os.Remove(ciscoSecretsPath); err != nil && !os.IsNotExist(err) {
		return "failed", map[string]any{}, fmt.Sprintf("remove cisco.secrets: %s", err.Error())
	}

	// Remove EAP-RADIUS config (best-effort)
	_ = os.Remove(eapRadiusConfPath)

	// Remove FreeRADIUS client config (best-effort)
	_ = os.Remove(filepath.Join(freeradiusClientDir, "cisco-ipsec-nas.conf"))

	// Restart ipsec service
	out, err := exec.Command("ipsec", "restart").CombinedOutput()
	if err != nil {
		return "failed", map[string]any{"output": string(out)}, fmt.Sprintf("ipsec restart: %s", err.Error())
	}

	log.Info("cisco ipsec removed", nil)
	return "succeeded", map[string]any{"removed": true}, ""
}

// generateCiscoConf builds the strongSwan connection config for IKEv1+XAUTH (Cisco IPSec)
// with RADIUS-based XAUTH authentication and accounting.
func generateCiscoConf(leftID, network, dns string) string {
	var b strings.Builder
	b.WriteString("# Cisco IPSec (IKEv1 + XAUTH) - managed by KorisPanel\n")
	b.WriteString("conn cisco-ipsec\n")
	b.WriteString("    keyexchange=ikev1\n")
	b.WriteString("    authby=xauthpsk\n")
	b.WriteString("    xauth=server\n")
	b.WriteString("    left=%defaultroute\n")
	b.WriteString(fmt.Sprintf("    leftid=%s\n", leftID))
	b.WriteString("    leftsubnet=0.0.0.0/0\n")
	b.WriteString(fmt.Sprintf("    rightsourceip=%s\n", network))
	b.WriteString(fmt.Sprintf("    rightdns=%s\n", dns))
	b.WriteString("    right=%any\n")
	b.WriteString("    rightauth=eap-radius\n")
	b.WriteString("    auto=add\n")
	b.WriteString("    ike=aes256-sha256-modp2048,aes128-sha1-modp2048!\n")
	b.WriteString("    esp=aes256-sha256,aes128-sha1!\n")
	b.WriteString("    aggressive=yes\n")
	b.WriteString("    fragmentation=yes\n")
	b.WriteString("    dpdaction=clear\n")
	b.WriteString("    dpddelay=30s\n")
	b.WriteString("    dpdtimeout=120s\n")
	return b.String()
}

// generateCiscoSecrets builds the ipsec.secrets content for PSK authentication.
func generateCiscoSecrets(psk string) string {
	return fmt.Sprintf("# Cisco IPSec PSK - managed by KorisPanel\n: PSK \"%s\"\n", psk)
}

// writeEAPRadiusConf generates the strongSwan EAP-RADIUS plugin configuration file.
// This enables strongSwan to authenticate XAUTH users against a FreeRADIUS server
// and send RADIUS accounting records (Start/Interim/Stop) to the radacct table.
func writeEAPRadiusConf(radiusServer, radiusSecret string) error {
	// Ensure the directory exists
	dir := filepath.Dir(eapRadiusConfPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create charon dir: %w", err)
	}

	var b strings.Builder
	b.WriteString("# EAP-RADIUS plugin config - managed by KorisPanel (Cisco IPSec)\n")
	b.WriteString("eap-radius {\n")
	b.WriteString("    load = yes\n")
	b.WriteString("    accounting = yes\n")
	b.WriteString("    accounting_close_on_timeout = yes\n")
	b.WriteString("    accounting_interval = 300\n")
	b.WriteString("    class_group = yes\n")
	b.WriteString("    nas_identifier = cisco-ipsec\n")
	b.WriteString("\n")
	b.WriteString("    servers {\n")
	b.WriteString("        primary {\n")
	b.WriteString(fmt.Sprintf("            address = %s\n", radiusServer))
	b.WriteString(fmt.Sprintf("            secret = %s\n", radiusSecret))
	b.WriteString("            auth_port = 1812\n")
	b.WriteString("            acct_port = 1813\n")
	b.WriteString("        }\n")
	b.WriteString("    }\n")
	b.WriteString("}\n")

	return os.WriteFile(eapRadiusConfPath, []byte(b.String()), 0600)
}

// configureFreeRADIUSClient adds this node as a RADIUS NAS client in FreeRADIUS.
// This allows FreeRADIUS to accept authentication and accounting requests from strongSwan.
// Only applicable when FreeRADIUS runs on the same host (common in single-node setups).
func configureFreeRADIUSClient(radiusServer, radiusSecret, leftID string, log *logger.Logger) error {
	// Only configure local FreeRADIUS if the RADIUS server is localhost
	if radiusServer != "127.0.0.1" && radiusServer != "localhost" && radiusServer != "::1" {
		return nil
	}

	// Check if FreeRADIUS clients.conf.d directory exists (standard on Ubuntu/Debian)
	// Fall back to appending to /etc/freeradius/3.0/clients.conf
	clientConfDir := freeradiusClientDir
	mainClientsConf := "/etc/freeradius/3.0/clients.conf"

	var clientConf strings.Builder
	clientConf.WriteString("# Cisco IPSec NAS client - managed by KorisPanel\n")
	clientConf.WriteString("client cisco-ipsec-local {\n")
	clientConf.WriteString("    ipaddr = 127.0.0.1\n")
	clientConf.WriteString(fmt.Sprintf("    secret = %s\n", radiusSecret))
	clientConf.WriteString("    shortname = cisco-ipsec\n")
	clientConf.WriteString("    nastype = cisco-ipsec\n")
	clientConf.WriteString("    proto = udp\n")
	clientConf.WriteString("}\n")

	// Try writing to clients.conf.d/ first
	if err := os.MkdirAll(clientConfDir, 0755); err == nil {
		confFile := filepath.Join(clientConfDir, "cisco-ipsec-nas.conf")
		if err := os.WriteFile(confFile, []byte(clientConf.String()), 0640); err != nil {
			return fmt.Errorf("write freeradius client config: %w", err)
		}
		log.Info("freeradius client configured", map[string]any{"path": confFile})
	} else {
		// Fall back: append to main clients.conf if it exists
		if _, err := os.Stat(mainClientsConf); err != nil {
			return fmt.Errorf("freeradius not found at expected paths")
		}
		// Check if already configured
		data, err := os.ReadFile(mainClientsConf)
		if err != nil {
			return fmt.Errorf("read clients.conf: %w", err)
		}
		if strings.Contains(string(data), "cisco-ipsec-local") {
			return nil // Already configured
		}
		f, err := os.OpenFile(mainClientsConf, os.O_APPEND|os.O_WRONLY, 0640)
		if err != nil {
			return fmt.Errorf("open clients.conf: %w", err)
		}
		defer f.Close()
		if _, err := f.WriteString("\n" + clientConf.String()); err != nil {
			return fmt.Errorf("append clients.conf: %w", err)
		}
		log.Info("freeradius client appended to clients.conf", nil)
	}

	// Reload FreeRADIUS to pick up new client (best-effort)
	if out, err := exec.Command("systemctl", "reload", "freeradius").CombinedOutput(); err != nil {
		log.Warn("freeradius reload failed", map[string]any{"output": string(out), "error": err.Error()})
	}

	return nil
}
