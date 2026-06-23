package api

import (
	"regexp"
	"strings"
	"testing"
)

func TestGenerateUUID_Format(t *testing.T) {
	// UUID format: 8-4-4-4-12 hex characters (lowercase)
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

	tests := []struct {
		name string
	}{
		{"first call"},
		{"second call"},
		{"third call"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			uuid := generateUUID()
			if !uuidRegex.MatchString(uuid) {
				t.Errorf("generateUUID() = %q, does not match UUID format 8-4-4-4-12 hex", uuid)
			}
		})
	}
}

func TestGenerateUUID_Uniqueness(t *testing.T) {
	seen := make(map[string]struct{})
	for i := 0; i < 100; i++ {
		uuid := generateUUID()
		if _, exists := seen[uuid]; exists {
			t.Fatalf("generateUUID() produced duplicate: %s", uuid)
		}
		seen[uuid] = struct{}{}
	}
}

func TestGenerateUUID_Lowercase(t *testing.T) {
	uuid := generateUUID()
	if uuid != strings.ToLower(uuid) {
		t.Errorf("generateUUID() = %q, expected all lowercase", uuid)
	}
}

func TestMobileconfigProfile_RequiredKeys(t *testing.T) {
	// Generate a UUID to use in the profile (simulates what the handler does)
	uuidPayload := generateUUID()
	uuidProfile := generateUUID()

	tests := []struct {
		name     string
		contains string
	}{
		{"PayloadType vpn.managed", "<string>com.apple.vpn.managed</string>"},
		{"PayloadType Configuration", "<string>Configuration</string>"},
		{"VPNType IPSec", "<string>IPSec</string>"},
		{"IPSec dict key", "<key>IPSec</key>"},
		{"SharedSecret key", "<key>SharedSecret</key>"},
		{"AuthenticationMethod SharedSecret", "<string>SharedSecret</string>"},
		{"XAuthEnabled", "<key>XAuthEnabled</key>"},
		{"PayloadUUID payload", uuidPayload},
		{"PayloadUUID profile", uuidProfile},
		{"PayloadVersion", "<integer>1</integer>"},
		{"PayloadContent array", "<key>PayloadContent</key>"},
	}

	// Build a minimal profile using the same template format as the handler
	profile := buildTestMobileconfig("testuser", uuidPayload, "vpn.example.com", "dGVzdHBzaw==", "testuser", "testpass", uuidProfile)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(profile, tc.contains) {
				t.Errorf("profile missing expected content %q", tc.contains)
			}
		})
	}
}

func TestMobileconfigProfile_XMLStructure(t *testing.T) {
	profile := buildTestMobileconfig("user1", generateUUID(), "10.0.0.1", "cHNr", "user1", "pass1", generateUUID())

	tests := []struct {
		name     string
		contains string
	}{
		{"XML declaration", `<?xml version="1.0" encoding="UTF-8"?>`},
		{"plist DOCTYPE", `<!DOCTYPE plist`},
		{"plist root", `<plist version="1.0">`},
		{"closing plist", `</plist>`},
		{"RemoteAddress", "<string>10.0.0.1</string>"},
		{"XAuthName", "<string>user1</string>"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(profile, tc.contains) {
				t.Errorf("profile missing expected content %q", tc.contains)
			}
		})
	}
}

// buildTestMobileconfig replicates the profile template from the handler for testing purposes.
func buildTestMobileconfig(username, uuidPayload, host, pskData, xauthName, xauthPass, uuidProfile string) string {
	return strings.Join([]string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">`,
		`<plist version="1.0">`,
		`<dict>`,
		`	<key>PayloadContent</key>`,
		`	<array>`,
		`		<dict>`,
		`			<key>PayloadDescription</key>`,
		`			<string>Configures Cisco IPSec VPN</string>`,
		`			<key>PayloadDisplayName</key>`,
		`			<string>KorisPanel VPN</string>`,
		`			<key>PayloadIdentifier</key>`,
		`			<string>com.korispanel.vpn.cisco-ipsec.` + username + `</string>`,
		`			<key>PayloadType</key>`,
		`			<string>com.apple.vpn.managed</string>`,
		`			<key>PayloadUUID</key>`,
		`			<string>` + uuidPayload + `</string>`,
		`			<key>PayloadVersion</key>`,
		`			<integer>1</integer>`,
		`			<key>UserDefinedName</key>`,
		`			<string>KorisPanel VPN</string>`,
		`			<key>VPNType</key>`,
		`			<string>IPSec</string>`,
		`			<key>IPSec</key>`,
		`			<dict>`,
		`				<key>AuthenticationMethod</key>`,
		`				<string>SharedSecret</string>`,
		`				<key>LocalIdentifierType</key>`,
		`				<string>KeyID</string>`,
		`				<key>RemoteAddress</key>`,
		`				<string>` + host + `</string>`,
		`				<key>SharedSecret</key>`,
		`				<data>` + pskData + `</data>`,
		`				<key>XAuthEnabled</key>`,
		`				<integer>1</integer>`,
		`				<key>XAuthName</key>`,
		`				<string>` + xauthName + `</string>`,
		`				<key>XAuthPassword</key>`,
		`				<string>` + xauthPass + `</string>`,
		`			</dict>`,
		`		</dict>`,
		`	</array>`,
		`	<key>PayloadDisplayName</key>`,
		`	<string>KorisPanel VPN</string>`,
		`	<key>PayloadIdentifier</key>`,
		`	<string>com.korispanel.vpn.cisco-ipsec.profile.` + username + `</string>`,
		`	<key>PayloadRemovalDisallowed</key>`,
		`	<false/>`,
		`	<key>PayloadType</key>`,
		`	<string>Configuration</string>`,
		`	<key>PayloadUUID</key>`,
		`	<string>` + uuidProfile + `</string>`,
		`	<key>PayloadVersion</key>`,
		`	<integer>1</integer>`,
		`</dict>`,
		`</plist>`,
	}, "\n")
}
