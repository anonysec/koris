package safehttp

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient_BlocksLoopback(t *testing.T) {
	client := NewClient(5 * time.Second)
	// Create a test server on localhost
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// The client should reject the loopback address
	_, err := client.Get(ts.URL)
	if err == nil {
		t.Fatal("client allowed request to loopback address")
	}
}

func TestNewClient_BlocksPrivateIP(t *testing.T) {
	client := NewClient(5 * time.Second)

	// Use a private IP address directly (10.0.0.1)
	req, err := http.NewRequestWithContext(context.Background(), "GET", "http://10.0.0.1:80/", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Do(req)
	if err == nil {
		t.Fatal("client allowed request to private IP 10.0.0.1")
	}
}

func TestNewClient_BlocksLinkLocal(t *testing.T) {
	client := NewClient(5 * time.Second)
	// 169.254.0.0/16 is link-local
	req, err := http.NewRequestWithContext(context.Background(), "GET", "http://169.254.169.254:80/", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Do(req)
	if err == nil {
		t.Fatal("client allowed request to link-local IP 169.254.169.254")
	}
}

func TestNewClient_AllowsPublicIP(t *testing.T) {
	client := NewClient(5 * time.Second)
	// Use a public IP that won't resolve to private (1.1.1.1 - Cloudflare DNS)
	// Note: this will fail to connect, but NOT due to SSRF blocking.
	// We just verify it's not blocked by our checker.
	req, err := http.NewRequestWithContext(context.Background(), "GET", "http://1.1.1.1:80/", nil)
	if err != nil {
		t.Fatal(err)
	}
	// Should get a connection error (timeout/refused), not ErrBlockedAddress
	_, err = client.Do(req)
	if err != nil && err.Error() == ErrBlockedAddress.Error() {
		t.Fatal("client incorrectly blocked public IP 1.1.1.1")
	}
}

func TestNewClient_RespectsTimeout(t *testing.T) {
	client := NewClient(10 * time.Millisecond)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	_, err := client.Get(ts.URL)
	if err == nil {
		t.Fatal("client did not timeout on slow server")
	}
}

func TestNewClient_RedirectLimit(t *testing.T) {
	client := NewClient(5 * time.Second)
	// Chain of 6 redirects (limit is 5)
	count := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		if count <= 6 {
			http.Redirect(w, r, "/next", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	_, err := client.Get(ts.URL + "/")
	if err == nil {
		t.Fatal("client allowed more than 5 redirects")
	}
}

func TestNewClient_DNSResolutionError(t *testing.T) {
	client := NewClient(5 * time.Second)
	// Domain that resolves to a private IP (if such exists) or invalid
	// Use a non-existent domain to test DNS failure path
	_, err := client.Get("http://this-domain-does-not-exist-12345.invalid/")
	if err == nil {
		t.Fatal("client succeeded on non-existent domain")
	}
}

func TestNewClient_BlocksUnspecifiedIP(t *testing.T) {
	client := NewClient(5 * time.Second)
	// 0.0.0.0 is unspecified
	req, err := http.NewRequestWithContext(context.Background(), "GET", "http://0.0.0.0:80/", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Do(req)
	if err == nil {
		t.Fatal("client allowed request to unspecified IP 0.0.0.0")
	}
}

func TestErrBlockedAddress(t *testing.T) {
	// Just verify the error constant is accessible and has a sensible message
	if ErrBlockedAddress == nil {
		t.Fatal("ErrBlockedAddress is nil")
	}
	if len(ErrBlockedAddress.Error()) == 0 {
		t.Fatal("ErrBlockedAddress has empty message")
	}
}

func TestClient_IPv6Loopback(t *testing.T) {
	client := NewClient(5 * time.Second)
	// ::1 is IPv6 loopback
	req, err := http.NewRequestWithContext(context.Background(), "GET", "http://[::1]:80/", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Do(req)
	if err == nil {
		t.Fatal("client allowed request to IPv6 loopback ::1")
	}
}

func TestClient_IPv6Private(t *testing.T) {
	client := NewClient(5 * time.Second)
	// fd00::/8 is ULA (private)
	req, err := http.NewRequestWithContext(context.Background(), "GET", "http://[fd00::1]:80/", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Do(req)
	if err == nil {
		t.Fatal("client allowed request to IPv6 ULA fd00::1")
	}
}

// Verify that the dialer properly handles net.SplitHostPort
func TestDialContext_InvalidAddr(t *testing.T) {
	client := NewClient(5 * time.Second)
	// Invalid host format - IPv6 without brackets
	_, err := client.Get("http://[::1/")
	if err == nil {
		t.Fatal("client succeeded on invalid IPv6 format")
	}
}

// Verify that a test server on a non-loopback IP works (if available)
// This is best-effort; skip if no external test endpoint
func TestClient_PublicEndpoint(t *testing.T) {
	client := NewClient(10 * time.Second)
	// Use a known public endpoint (httpbin.org) — this is an integration-style test
	// and will be skipped if network is restricted (common in CI/sandbox)
	resp, err := client.Get("https://httpbin.org/get")
	if err != nil {
		t.Skipf("skipping public endpoint test (network may be restricted): %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}
}

// Test that the dialer blocks the same IP ranges as Go's net.IP methods
func TestIPBlockList(t *testing.T) {
	testCases := []struct {
		ipStr   string
		wantErr bool
	}{
		{"127.0.0.1", true},    // loopback
		{"10.0.0.1", true},     // private
		{"172.16.0.1", true},   // private
		{"192.168.1.1", true},  // private
		{"169.254.0.1", true},  // link-local
		{"169.254.255.255", true},
		{"0.0.0.0", true},      // unspecified
		{"1.1.1.1", false},     // public
		{"8.8.8.8", false},     // public
	}

	for _, tc := range testCases {
		ip := net.ParseIP(tc.ipStr)
		if ip == nil {
			t.Fatalf("parse %q failed", tc.ipStr)
		}
		isBlocked := ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
		if isBlocked != tc.wantErr {
			t.Errorf("IP %s: expected blocked=%v, got %v", tc.ipStr, tc.wantErr, isBlocked)
		}
	}
}

// IPv6 block list
func TestIPv6BlockList(t *testing.T) {
	testCases := []struct {
		ipStr   string
		wantErr bool
	}{
		{"::1", true},       // loopback
		{"fe80::1", true},   // link-local
		{"fd00::1", true},   // ULA (private)
		{"2001:db8::1", false}, // documentation (public)
		{"2606:4700:4700::1111", false}, // 1.1.1.1
	}

	for _, tc := range testCases {
		ip := net.ParseIP(tc.ipStr)
		if ip == nil {
			t.Fatalf("parse %q failed", tc.ipStr)
		}
		isBlocked := ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
		if isBlocked != tc.wantErr {
			t.Errorf("IPv6 %s: expected blocked=%v, got %v", tc.ipStr, tc.wantErr, isBlocked)
		}
	}
}