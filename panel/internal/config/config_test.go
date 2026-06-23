package config

import (
	"os"
	"testing"
)

func TestLoad_DevMode_AllowsEmptySecrets(t *testing.T) {
	// In dev mode, empty secrets should use defaults without fatal
	os.Setenv("PANEL_DEV_MODE", "true")
	os.Unsetenv("PANEL_SESSION_SECRET")
	os.Unsetenv("PANEL_SETUP_KEY")
	os.Unsetenv("PANEL_DB_DSN")
	defer func() {
		os.Unsetenv("PANEL_DEV_MODE")
	}()

	cfg := Load()

	if cfg.SessionSecret == "" {
		t.Error("expected SessionSecret to have a default in dev mode")
	}
	if cfg.DBDSN == "" {
		t.Error("expected DBDSN to have a default in dev mode")
	}
}

func TestLoad_DevMode_UsesProvidedValues(t *testing.T) {
	os.Setenv("PANEL_DEV_MODE", "true")
	os.Setenv("PANEL_SESSION_SECRET", "my-custom-secret-that-is-long-enough")
	os.Setenv("PANEL_DB_DSN", "user:pass@tcp(localhost)/db")
	defer func() {
		os.Unsetenv("PANEL_DEV_MODE")
		os.Unsetenv("PANEL_SESSION_SECRET")
		os.Unsetenv("PANEL_DB_DSN")
	}()

	cfg := Load()

	if cfg.SessionSecret != "my-custom-secret-that-is-long-enough" {
		t.Errorf("expected SessionSecret to be 'my-custom-secret-that-is-long-enough', got %q", cfg.SessionSecret)
	}
	if cfg.DBDSN != "user:pass@tcp(localhost)/db" {
		t.Errorf("expected DBDSN to be 'user:pass@tcp(localhost)/db', got %q", cfg.DBDSN)
	}
}

func TestLoad_DevModeNotSet_WithValidConfig(t *testing.T) {
	// When PANEL_DEV_MODE is not "true" but required vars are set, should work fine
	os.Unsetenv("PANEL_DEV_MODE")
	os.Setenv("PANEL_SESSION_SECRET", "a-valid-session-secret-for-production-use")
	os.Setenv("PANEL_DB_DSN", "user:pass@tcp(localhost)/db")
	defer func() {
		os.Unsetenv("PANEL_SESSION_SECRET")
		os.Unsetenv("PANEL_DB_DSN")
	}()

	cfg := Load()

	if cfg.SessionSecret != "a-valid-session-secret-for-production-use" {
		t.Errorf("expected SessionSecret to be set, got %q", cfg.SessionSecret)
	}
	if cfg.DBDSN != "user:pass@tcp(localhost)/db" {
		t.Errorf("expected DBDSN to be set, got %q", cfg.DBDSN)
	}
}

func TestLoad_DevModeEnvParsing(t *testing.T) {
	// Only exact "true" should enable dev mode
	tests := []struct {
		value     string
		isDevMode bool
	}{
		{"true", true},
		{"TRUE", false},
		{"1", false},
		{"false", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run("PANEL_DEV_MODE="+tt.value, func(t *testing.T) {
			if tt.value == "" {
				os.Unsetenv("PANEL_DEV_MODE")
			} else {
				os.Setenv("PANEL_DEV_MODE", tt.value)
			}
			// Set valid config to avoid fatalf in non-dev mode
			os.Setenv("PANEL_SESSION_SECRET", "a-valid-session-secret-for-production-use")
			os.Setenv("PANEL_DB_DSN", "user:pass@tcp(localhost)/db")
			defer func() {
				os.Unsetenv("PANEL_DEV_MODE")
				os.Unsetenv("PANEL_SESSION_SECRET")
				os.Unsetenv("PANEL_DB_DSN")
			}()

			// Should not panic/fatal since config is provided
			cfg := Load()
			if cfg.SessionSecret == "" {
				t.Error("expected SessionSecret to be non-empty")
			}
		})
	}
}

func TestLoad_SecureCookies_FalseByDefault(t *testing.T) {
	os.Unsetenv("PANEL_DEV_MODE")
	os.Unsetenv("PANEL_SECURE_COOKIES")
	os.Setenv("PANEL_SESSION_SECRET", "a-valid-session-secret-for-production-use")
	os.Setenv("PANEL_DB_DSN", "user:pass@tcp(localhost)/db")
	defer func() {
		os.Unsetenv("PANEL_SESSION_SECRET")
		os.Unsetenv("PANEL_DB_DSN")
	}()

	cfg := Load()

	if cfg.SecureCookies {
		t.Error("expected SecureCookies to be false by default (reverse proxy compatibility)")
	}
}

func TestLoad_SecureCookies_TrueWhenExplicitlySet(t *testing.T) {
	os.Unsetenv("PANEL_DEV_MODE")
	os.Setenv("PANEL_SECURE_COOKIES", "true")
	os.Setenv("PANEL_SESSION_SECRET", "a-valid-session-secret-for-production-use")
	os.Setenv("PANEL_DB_DSN", "user:pass@tcp(localhost)/db")
	defer func() {
		os.Unsetenv("PANEL_SECURE_COOKIES")
		os.Unsetenv("PANEL_SESSION_SECRET")
		os.Unsetenv("PANEL_DB_DSN")
	}()

	cfg := Load()

	if !cfg.SecureCookies {
		t.Error("expected SecureCookies to be true when PANEL_SECURE_COOKIES=true")
	}
}

func TestLoad_SecureCookies_CaseInsensitive(t *testing.T) {
	os.Unsetenv("PANEL_DEV_MODE")
	os.Setenv("PANEL_SECURE_COOKIES", "TRUE")
	os.Setenv("PANEL_SESSION_SECRET", "a-valid-session-secret-for-production-use")
	os.Setenv("PANEL_DB_DSN", "user:pass@tcp(localhost)/db")
	defer func() {
		os.Unsetenv("PANEL_SECURE_COOKIES")
		os.Unsetenv("PANEL_SESSION_SECRET")
		os.Unsetenv("PANEL_DB_DSN")
	}()

	cfg := Load()

	if !cfg.SecureCookies {
		t.Error("expected SecureCookies to be true when PANEL_SECURE_COOKIES=TRUE (case insensitive)")
	}
}

func TestLoad_SecureCookies_FalseInDevMode(t *testing.T) {
	os.Setenv("PANEL_DEV_MODE", "true")
	os.Unsetenv("PANEL_SESSION_SECRET")
	os.Unsetenv("PANEL_DB_DSN")
	os.Unsetenv("PANEL_SECURE_COOKIES")
	defer func() {
		os.Unsetenv("PANEL_DEV_MODE")
	}()

	cfg := Load()

	if cfg.SecureCookies {
		t.Error("expected SecureCookies to be false in dev mode")
	}
}

func TestLoad_SecureCookies_FalseWhenExplicitlyDisabled(t *testing.T) {
	os.Unsetenv("PANEL_DEV_MODE")
	os.Setenv("PANEL_SECURE_COOKIES", "false")
	os.Setenv("PANEL_SESSION_SECRET", "a-valid-session-secret-for-production-use")
	os.Setenv("PANEL_DB_DSN", "user:pass@tcp(localhost)/db")
	defer func() {
		os.Unsetenv("PANEL_SECURE_COOKIES")
		os.Unsetenv("PANEL_SESSION_SECRET")
		os.Unsetenv("PANEL_DB_DSN")
	}()

	cfg := Load()

	if cfg.SecureCookies {
		t.Error("expected SecureCookies to be false when PANEL_SECURE_COOKIES=false")
	}
}

func TestLoad_TrustedProxies_Parsed(t *testing.T) {
	os.Setenv("PANEL_DEV_MODE", "true")
	os.Unsetenv("PANEL_SESSION_SECRET")
	os.Unsetenv("PANEL_DB_DSN")
	os.Setenv("PANEL_TRUSTED_PROXIES", "10.0.0.1, 192.168.1.0/24, 172.16.0.1")
	defer func() {
		os.Unsetenv("PANEL_DEV_MODE")
		os.Unsetenv("PANEL_TRUSTED_PROXIES")
	}()

	cfg := Load()

	expected := []string{"10.0.0.1", "192.168.1.0/24", "172.16.0.1"}
	if len(cfg.TrustedProxies) != len(expected) {
		t.Fatalf("expected %d trusted proxies, got %d", len(expected), len(cfg.TrustedProxies))
	}
	for i, v := range expected {
		if cfg.TrustedProxies[i] != v {
			t.Errorf("expected TrustedProxies[%d] = %q, got %q", i, v, cfg.TrustedProxies[i])
		}
	}
}

func TestLoad_TrustedProxies_EmptyWhenUnset(t *testing.T) {
	os.Setenv("PANEL_DEV_MODE", "true")
	os.Unsetenv("PANEL_SESSION_SECRET")
	os.Unsetenv("PANEL_DB_DSN")
	os.Unsetenv("PANEL_TRUSTED_PROXIES")
	defer func() {
		os.Unsetenv("PANEL_DEV_MODE")
	}()

	cfg := Load()

	if len(cfg.TrustedProxies) != 0 {
		t.Errorf("expected empty TrustedProxies, got %v", cfg.TrustedProxies)
	}
}

func TestLoad_AllowedOrigins_Parsed(t *testing.T) {
	os.Setenv("PANEL_DEV_MODE", "true")
	os.Unsetenv("PANEL_SESSION_SECRET")
	os.Unsetenv("PANEL_DB_DSN")
	os.Setenv("PANEL_ALLOWED_ORIGINS", "https://example.com, https://app.example.com")
	defer func() {
		os.Unsetenv("PANEL_DEV_MODE")
		os.Unsetenv("PANEL_ALLOWED_ORIGINS")
	}()

	cfg := Load()

	expected := []string{"https://example.com", "https://app.example.com"}
	if len(cfg.AllowedOrigins) != len(expected) {
		t.Fatalf("expected %d allowed origins, got %d", len(expected), len(cfg.AllowedOrigins))
	}
	for i, v := range expected {
		if cfg.AllowedOrigins[i] != v {
			t.Errorf("expected AllowedOrigins[%d] = %q, got %q", i, v, cfg.AllowedOrigins[i])
		}
	}
}

func TestLoad_AllowedOrigins_EmptyWhenUnset(t *testing.T) {
	os.Setenv("PANEL_DEV_MODE", "true")
	os.Unsetenv("PANEL_SESSION_SECRET")
	os.Unsetenv("PANEL_DB_DSN")
	os.Unsetenv("PANEL_ALLOWED_ORIGINS")
	defer func() {
		os.Unsetenv("PANEL_DEV_MODE")
	}()

	cfg := Load()

	if len(cfg.AllowedOrigins) != 0 {
		t.Errorf("expected empty AllowedOrigins, got %v", cfg.AllowedOrigins)
	}
}

func TestLoad_TrustedProxies_SkipsEmptyEntries(t *testing.T) {
	os.Setenv("PANEL_DEV_MODE", "true")
	os.Unsetenv("PANEL_SESSION_SECRET")
	os.Unsetenv("PANEL_DB_DSN")
	os.Setenv("PANEL_TRUSTED_PROXIES", "10.0.0.1,,  , 172.16.0.1")
	defer func() {
		os.Unsetenv("PANEL_DEV_MODE")
		os.Unsetenv("PANEL_TRUSTED_PROXIES")
	}()

	cfg := Load()

	expected := []string{"10.0.0.1", "172.16.0.1"}
	if len(cfg.TrustedProxies) != len(expected) {
		t.Fatalf("expected %d trusted proxies, got %d: %v", len(expected), len(cfg.TrustedProxies), cfg.TrustedProxies)
	}
	for i, v := range expected {
		if cfg.TrustedProxies[i] != v {
			t.Errorf("expected TrustedProxies[%d] = %q, got %q", i, v, cfg.TrustedProxies[i])
		}
	}
}

func TestLoad_SessionSecret_MinLength_ProductionWithValidSecret(t *testing.T) {
	// A 32+ character secret should be accepted in production mode
	os.Unsetenv("PANEL_DEV_MODE")
	os.Setenv("PANEL_SESSION_SECRET", "abcdefghijklmnopqrstuvwxyz123456") // exactly 32 chars
	os.Setenv("PANEL_DB_DSN", "user:pass@tcp(localhost)/db")
	defer func() {
		os.Unsetenv("PANEL_SESSION_SECRET")
		os.Unsetenv("PANEL_DB_DSN")
	}()

	cfg := Load()

	if cfg.SessionSecret != "abcdefghijklmnopqrstuvwxyz123456" {
		t.Errorf("expected SessionSecret to be accepted, got %q", cfg.SessionSecret)
	}
}

func TestLoad_SessionSecret_ShortSecret_DevModeAllowed(t *testing.T) {
	// In dev mode, short secrets should be allowed (no length check)
	os.Setenv("PANEL_DEV_MODE", "true")
	os.Setenv("PANEL_SESSION_SECRET", "short")
	os.Unsetenv("PANEL_DB_DSN")
	defer func() {
		os.Unsetenv("PANEL_DEV_MODE")
		os.Unsetenv("PANEL_SESSION_SECRET")
	}()

	cfg := Load()

	if cfg.SessionSecret != "short" {
		t.Errorf("expected SessionSecret to be 'short' in dev mode, got %q", cfg.SessionSecret)
	}
}

func TestLoad_TLS_Enabled(t *testing.T) {
	os.Setenv("PANEL_DEV_MODE", "true")
	os.Setenv("PANEL_TLS_ENABLED", "true")
	defer func() {
		os.Unsetenv("PANEL_DEV_MODE")
		os.Unsetenv("PANEL_TLS_ENABLED")
	}()

	cfg := Load()

	if !cfg.TLSEnabled {
		t.Error("expected TLSEnabled to be true when PANEL_TLS_ENABLED=true")
	}
}

func TestLoad_TLS_Disabled_ByDefault(t *testing.T) {
	os.Setenv("PANEL_DEV_MODE", "true")
	os.Unsetenv("PANEL_TLS_ENABLED")
	defer func() {
		os.Unsetenv("PANEL_DEV_MODE")
	}()

	cfg := Load()

	if cfg.TLSEnabled {
		t.Error("expected TLSEnabled to be false by default")
	}
}

func TestLoad_TLS_Domain(t *testing.T) {
	os.Setenv("PANEL_DEV_MODE", "true")
	os.Setenv("PANEL_TLS_DOMAIN", "panel.example.com")
	defer func() {
		os.Unsetenv("PANEL_DEV_MODE")
		os.Unsetenv("PANEL_TLS_DOMAIN")
	}()

	cfg := Load()

	if cfg.TLSDomain != "panel.example.com" {
		t.Errorf("expected TLSDomain to be 'panel.example.com', got %q", cfg.TLSDomain)
	}
}

func TestLoad_TLS_Email(t *testing.T) {
	os.Setenv("PANEL_DEV_MODE", "true")
	os.Setenv("PANEL_TLS_EMAIL", "admin@example.com")
	defer func() {
		os.Unsetenv("PANEL_DEV_MODE")
		os.Unsetenv("PANEL_TLS_EMAIL")
	}()

	cfg := Load()

	if cfg.TLSEmail != "admin@example.com" {
		t.Errorf("expected TLSEmail to be 'admin@example.com', got %q", cfg.TLSEmail)
	}
}

func TestLoad_TLS_CertDir_Default(t *testing.T) {
	os.Setenv("PANEL_DEV_MODE", "true")
	os.Unsetenv("PANEL_TLS_CERT_DIR")
	defer func() {
		os.Unsetenv("PANEL_DEV_MODE")
	}()

	cfg := Load()

	if cfg.TLSCertDir != "/etc/panel/certs" {
		t.Errorf("expected TLSCertDir default to be '/etc/panel/certs', got %q", cfg.TLSCertDir)
	}
}

func TestLoad_TLS_CertDir_Custom(t *testing.T) {
	os.Setenv("PANEL_DEV_MODE", "true")
	os.Setenv("PANEL_TLS_CERT_DIR", "/custom/certs")
	defer func() {
		os.Unsetenv("PANEL_DEV_MODE")
		os.Unsetenv("PANEL_TLS_CERT_DIR")
	}()

	cfg := Load()

	if cfg.TLSCertDir != "/custom/certs" {
		t.Errorf("expected TLSCertDir to be '/custom/certs', got %q", cfg.TLSCertDir)
	}
}

func TestLoad_TLS_Domain_EmptyByDefault(t *testing.T) {
	os.Setenv("PANEL_DEV_MODE", "true")
	os.Unsetenv("PANEL_TLS_DOMAIN")
	defer func() {
		os.Unsetenv("PANEL_DEV_MODE")
	}()

	cfg := Load()

	if cfg.TLSDomain != "" {
		t.Errorf("expected TLSDomain to be empty by default, got %q", cfg.TLSDomain)
	}
}

func TestLoad_TLS_Email_EmptyByDefault(t *testing.T) {
	os.Setenv("PANEL_DEV_MODE", "true")
	os.Unsetenv("PANEL_TLS_EMAIL")
	defer func() {
		os.Unsetenv("PANEL_DEV_MODE")
	}()

	cfg := Load()

	if cfg.TLSEmail != "" {
		t.Errorf("expected TLSEmail to be empty by default, got %q", cfg.TLSEmail)
	}
}
