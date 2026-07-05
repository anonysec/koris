package autocert

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStripPort(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"example.com:443", "example.com"},
		{"example.com:80", "example.com"},
		{"example.com", "example.com"},
		{"[::1]:443", "[::1]"},
		{"localhost:8080", "localhost"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stripPort(tt.input)
			if got != tt.expected {
				t.Errorf("stripPort(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestBuildPort80Handler_NoHTTPChallenge_Redirect(t *testing.T) {
	cfg := TLSConfig{
		Enabled:       true,
		Domain:        "panel.example.com",
		CertDir:       t.TempDir(),
		HTTPChallenge: false,
	}
	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	handler := buildPort80Handler(m, cfg)

	// Test redirect
	req := httptest.NewRequest(http.MethodGet, "http://panel.example.com/dashboard/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Errorf("expected status 301, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc != "https://panel.example.com/dashboard/" {
		t.Errorf("expected redirect to https://panel.example.com/dashboard/, got %q", loc)
	}
}

func TestBuildPort80Handler_NoHTTPChallenge_HealthCheck(t *testing.T) {
	cfg := TLSConfig{
		Enabled:       true,
		Domain:        "panel.example.com",
		CertDir:       t.TempDir(),
		HTTPChallenge: false,
	}
	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	handler := buildPort80Handler(m, cfg)

	// Test health check endpoint
	req := httptest.NewRequest(http.MethodGet, "http://panel.example.com/api/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	body := w.Body.String()
	if body != `{"ok":true,"service":"panel","tls":true}` {
		t.Errorf("unexpected health response: %s", body)
	}
}

func TestBuildPort80Handler_WithHTTPChallenge_HealthCheck(t *testing.T) {
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

	handler := buildPort80Handler(m, cfg)

	// Health check should still work even with HTTP challenge enabled
	req := httptest.NewRequest(http.MethodGet, "http://panel.example.com/api/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	body := w.Body.String()
	if body != `{"ok":true,"service":"panel","tls":true}` {
		t.Errorf("unexpected health response: %s", body)
	}
}

func TestBuildPort80Handler_WithHTTPChallenge_NonACME_Redirect(t *testing.T) {
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

	handler := buildPort80Handler(m, cfg)

	// Non-ACME, non-health request should get redirected
	req := httptest.NewRequest(http.MethodGet, "http://panel.example.com/some-page", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// autocert's HTTPHandler redirects to HTTPS for non-challenge paths
	if w.Code != http.StatusMovedPermanently && w.Code != http.StatusFound {
		t.Errorf("expected redirect status (301 or 302), got %d", w.Code)
	}
}

func TestListenAndServeTLS_EnabledRequiresDomain(t *testing.T) {
	// When TLS is enabled but domain is empty, should return an error
	cfg := TLSConfig{
		Enabled: true,
		Domain:  "",
	}

	err := ListenAndServeTLS(http.DefaultServeMux, cfg, ":0")
	if err == nil {
		t.Fatal("expected error when domain is empty, got nil")
	}
	if err.Error() != "autocert: domain is required" {
		t.Errorf("unexpected error: %v", err)
	}
}
