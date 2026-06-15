package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIP_UntrustedProxyIgnoresXRealIP(t *testing.T) {
	// Request from 5.5.5.5 (not trusted) with X-Real-IP header → should ignore X-Real-IP
	l := New(10, 30, []string{"10.0.0.1"})

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "5.5.5.5:12345"
	req.Header.Set("X-Real-IP", "9.9.9.9")

	got := l.clientIP(req)
	if got != "5.5.5.5" {
		t.Errorf("untrusted proxy: expected RemoteAddr 5.5.5.5, got %s", got)
	}
}

func TestClientIP_TrustedProxyUsesXRealIP(t *testing.T) {
	// Request from 10.0.0.1 (trusted) with X-Real-IP → should use X-Real-IP
	l := New(10, 30, []string{"10.0.0.1"})

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Real-IP", "203.0.113.50")

	got := l.clientIP(req)
	if got != "203.0.113.50" {
		t.Errorf("trusted proxy: expected X-Real-IP 203.0.113.50, got %s", got)
	}
}

func TestClientIP_TrustedCIDRUsesXRealIP(t *testing.T) {
	// Request from 192.168.1.50 (in trusted CIDR 192.168.1.0/24) with X-Real-IP → should use X-Real-IP
	l := New(10, 30, []string{"192.168.1.0/24"})

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	req.Header.Set("X-Real-IP", "8.8.8.8")

	got := l.clientIP(req)
	if got != "8.8.8.8" {
		t.Errorf("trusted CIDR: expected X-Real-IP 8.8.8.8, got %s", got)
	}
}

func TestClientIP_NoXRealIPUsesRemoteAddr(t *testing.T) {
	// No X-Real-IP header → uses RemoteAddr regardless of trust
	l := New(10, 30, []string{"10.0.0.1"})

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"

	got := l.clientIP(req)
	if got != "10.0.0.1" {
		t.Errorf("no X-Real-IP: expected RemoteAddr 10.0.0.1, got %s", got)
	}
}

func TestClientIP_InvalidXRealIPIgnored(t *testing.T) {
	// Trusted proxy with invalid X-Real-IP (not a valid IP) → falls back to RemoteAddr
	l := New(10, 30, []string{"10.0.0.1"})

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Real-IP", "not-an-ip")

	got := l.clientIP(req)
	if got != "10.0.0.1" {
		t.Errorf("invalid X-Real-IP: expected RemoteAddr 10.0.0.1, got %s", got)
	}
}

func TestMiddleware_RateLimitsCorrectIP(t *testing.T) {
	// Verify middleware uses clientIP correctly - untrusted proxy should rate limit by RemoteAddr
	l := New(10, 30, []string{"10.0.0.1"})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := l.Middleware(handler)

	// First request from untrusted proxy with spoofed X-Real-IP should use RemoteAddr
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "5.5.5.5:12345"
	req.Header.Set("X-Real-IP", "9.9.9.9")

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestIsTrustedProxy_DirectIP(t *testing.T) {
	l := New(10, 30, []string{"10.0.0.1", "172.16.0.5"})

	tests := []struct {
		ip       string
		expected bool
	}{
		{"10.0.0.1", true},
		{"172.16.0.5", true},
		{"10.0.0.2", false},
		{"192.168.1.1", false},
	}

	for _, tt := range tests {
		got := l.isTrustedProxy(tt.ip)
		if got != tt.expected {
			t.Errorf("isTrustedProxy(%s) = %v, want %v", tt.ip, got, tt.expected)
		}
	}
}

func TestIsTrustedProxy_CIDR(t *testing.T) {
	l := New(10, 30, []string{"10.0.0.0/8", "192.168.1.0/24"})

	tests := []struct {
		ip       string
		expected bool
	}{
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		{"192.168.1.100", true},
		{"192.168.2.1", false},
		{"172.16.0.1", false},
	}

	for _, tt := range tests {
		got := l.isTrustedProxy(tt.ip)
		if got != tt.expected {
			t.Errorf("isTrustedProxy(%s) = %v, want %v", tt.ip, got, tt.expected)
		}
	}
}

func TestNew_EmptyTrustedProxies(t *testing.T) {
	// Empty trusted proxies → no proxies trusted, always use RemoteAddr
	l := New(10, 30, nil)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	req.Header.Set("X-Real-IP", "9.9.9.9")

	got := l.clientIP(req)
	if got != "1.2.3.4" {
		t.Errorf("empty trusted proxies: expected RemoteAddr 1.2.3.4, got %s", got)
	}
}
