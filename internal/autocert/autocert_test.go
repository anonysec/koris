package autocert

import (
	"net/http"
	"testing"
)

func TestNewManager_Disabled(t *testing.T) {
	cfg := TLSConfig{
		Enabled: false,
		Domain:  "example.com",
	}
	m, err := NewManager(cfg)
	if err == nil {
		t.Fatal("expected error when TLS is disabled, got nil")
	}
	if m != nil {
		t.Fatal("expected nil manager when TLS is disabled")
	}
	if err.Error() != "autocert: TLS is not enabled" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewManager_EmptyDomain(t *testing.T) {
	cfg := TLSConfig{
		Enabled: true,
		Domain:  "",
	}
	m, err := NewManager(cfg)
	if err == nil {
		t.Fatal("expected error when domain is empty, got nil")
	}
	if m != nil {
		t.Fatal("expected nil manager when domain is empty")
	}
	if err.Error() != "autocert: domain is required" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewManager_ValidConfig(t *testing.T) {
	cfg := TLSConfig{
		Enabled:       true,
		Domain:        "panel.example.com",
		Email:         "admin@example.com",
		CertDir:       t.TempDir(),
		HTTPChallenge: true,
	}
	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil manager")
	}
	if m.Email != "admin@example.com" {
		t.Errorf("expected email 'admin@example.com', got %q", m.Email)
	}
}

func TestNewManager_DefaultCertDir(t *testing.T) {
	cfg := TLSConfig{
		Enabled: true,
		Domain:  "panel.example.com",
		CertDir: "", // should use default
	}
	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil manager")
	}
	// Verify the cache is set (DirCache uses the default path)
	if m.Cache == nil {
		t.Error("expected cache to be set")
	}
}

func TestNewManager_CustomCertDir(t *testing.T) {
	dir := t.TempDir()
	cfg := TLSConfig{
		Enabled: true,
		Domain:  "panel.example.com",
		CertDir: dir,
	}
	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestNewManager_NoEmail(t *testing.T) {
	cfg := TLSConfig{
		Enabled: true,
		Domain:  "panel.example.com",
		CertDir: t.TempDir(),
	}
	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Email != "" {
		t.Errorf("expected empty email, got %q", m.Email)
	}
}

func TestNewManager_HostPolicy(t *testing.T) {
	cfg := TLSConfig{
		Enabled: true,
		Domain:  "panel.example.com",
		CertDir: t.TempDir(),
	}
	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// HostPolicy should accept the configured domain
	if err := m.HostPolicy(nil, "panel.example.com"); err != nil {
		t.Errorf("expected HostPolicy to accept configured domain, got error: %v", err)
	}
	// HostPolicy should reject other domains
	if err := m.HostPolicy(nil, "other.example.com"); err == nil {
		t.Error("expected HostPolicy to reject other domain, got nil error")
	}
}

func TestTLSConfig_Defaults(t *testing.T) {
	cfg := TLSConfig{}
	if cfg.Enabled {
		t.Error("expected Enabled to default to false")
	}
	if cfg.Domain != "" {
		t.Error("expected Domain to default to empty")
	}
	if cfg.Email != "" {
		t.Error("expected Email to default to empty")
	}
	if cfg.CertDir != "" {
		t.Error("expected CertDir to default to empty")
	}
	if cfg.HTTPChallenge {
		t.Error("expected HTTPChallenge to default to false")
	}
}

func TestHTTPChallengeHandler_NotNil(t *testing.T) {
	cfg := TLSConfig{
		Enabled:       true,
		Domain:        "panel.example.com",
		CertDir:       t.TempDir(),
		HTTPChallenge: true,
	}
	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	h := HTTPChallengeHandler(m, nil)
	if h == nil {
		t.Error("expected non-nil handler from HTTPChallengeHandler")
	}
}

// --- Additional coverage tests (task 11.6) ---

func TestNewManager_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		cfg         TLSConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "very long domain",
			cfg: TLSConfig{
				Enabled: true,
				Domain:  "a-very-long-subdomain-that-goes-on-and-on.another-level.deep.example.com",
				CertDir: t.TempDir(),
			},
			wantErr: false,
		},
		{
			name: "domain with hyphens",
			cfg: TLSConfig{
				Enabled: true,
				Domain:  "my-panel-server-01.vpn-service.io",
				CertDir: t.TempDir(),
			},
			wantErr: false,
		},
		{
			name: "domain with numbers",
			cfg: TLSConfig{
				Enabled: true,
				Domain:  "panel123.example456.com",
				CertDir: t.TempDir(),
			},
			wantErr: false,
		},
		{
			name: "single label domain",
			cfg: TLSConfig{
				Enabled: true,
				Domain:  "localhost",
				CertDir: t.TempDir(),
			},
			wantErr: false,
		},
		{
			name: "whitespace-only domain",
			cfg: TLSConfig{
				Enabled: true,
				Domain:  "   ",
				CertDir: t.TempDir(),
			},
			wantErr: false, // NewManager doesn't validate domain format, just emptiness
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m, err := NewManager(tc.cfg)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errContains != "" && !contains(err.Error(), tc.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tc.errContains)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if m == nil {
					t.Fatal("expected non-nil manager")
				}
			}
		})
	}
}

func TestNewManager_HostPolicy_EdgeCaseDomains(t *testing.T) {
	tests := []struct {
		name         string
		configDomain string
		checkDomain  string
		wantErr      bool
	}{
		{
			name:         "exact match",
			configDomain: "panel.example.com",
			checkDomain:  "panel.example.com",
			wantErr:      false,
		},
		{
			name:         "subdomain mismatch",
			configDomain: "panel.example.com",
			checkDomain:  "www.panel.example.com",
			wantErr:      true,
		},
		{
			name:         "different TLD",
			configDomain: "panel.example.com",
			checkDomain:  "panel.example.org",
			wantErr:      true,
		},
		{
			name:         "case insensitive match",
			configDomain: "Panel.Example.COM",
			checkDomain:  "panel.example.com",
			wantErr:      false, // HostWhitelist is case-insensitive
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := TLSConfig{
				Enabled: true,
				Domain:  tc.configDomain,
				CertDir: t.TempDir(),
			}
			m, err := NewManager(cfg)
			if err != nil {
				t.Fatalf("NewManager error: %v", err)
			}
			err = m.HostPolicy(nil, tc.checkDomain)
			if tc.wantErr && err == nil {
				t.Error("expected HostPolicy to reject domain, got nil error")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected HostPolicy to accept domain, got error: %v", err)
			}
		})
	}
}

func TestConfigToManagerToTLSConfig_Flow(t *testing.T) {
	// Complete flow: TLSConfig → NewManager → extract TLS config
	cfg := TLSConfig{
		Enabled:       true,
		Domain:        "flow-test.example.com",
		Email:         "admin@example.com",
		CertDir:       t.TempDir(),
		HTTPChallenge: true,
	}

	// Step 1: Create manager from config
	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager error: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil manager")
	}

	// Step 2: Extract TLS config from manager
	tlsCfg := m.TLSConfig()
	if tlsCfg == nil {
		t.Fatal("expected non-nil TLS config from manager")
	}

	// Step 3: Verify TLS config has GetCertificate set (autocert manages certs)
	if tlsCfg.GetCertificate == nil {
		t.Error("TLS config should have GetCertificate function set by autocert")
	}

	// Step 4: Verify NextProtos includes acme-tls/1 for ALPN challenges
	hasACME := false
	for _, proto := range tlsCfg.NextProtos {
		if proto == "acme-tls/1" {
			hasACME = true
			break
		}
	}
	if !hasACME {
		t.Error("TLS config NextProtos should include 'acme-tls/1' for ALPN challenges")
	}

	// Step 5: Verify HTTP challenge handler can be built
	h := HTTPChallengeHandler(m, nil)
	if h == nil {
		t.Error("HTTPChallengeHandler should return non-nil handler")
	}
}

func TestListenAndServeTLS_DisabledConfig(t *testing.T) {
	// When TLS is disabled, ListenAndServeTLS should start a plain HTTP server.
	// We can't easily test the blocking behavior, but we verify it doesn't error
	// on manager creation (it goes straight to HTTP).
	// Instead, test that enabling with valid config creates the manager successfully.
	cfg := TLSConfig{
		Enabled: false,
		Domain:  "disabled.example.com",
	}

	// With TLS disabled, ListenAndServeTLS would try to listen on addr.
	// We use an invalid addr to quickly trigger an error (proving it reached HTTP path).
	err := ListenAndServeTLS(http.NewServeMux(), cfg, "invalid-addr-that-will-fail:99999999")
	if err == nil {
		t.Fatal("expected error with invalid address, got nil")
	}
	// The error should be from net.Listen, not from autocert manager creation
	if err.Error() == "autocert: TLS is not enabled" {
		t.Error("should not get autocert error when TLS is disabled")
	}
}

// contains is a helper to check substring presence (avoids import of strings in test)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
