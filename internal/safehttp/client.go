// Package safehttp provides an HTTP client with SSRF protections.
//
// All outbound HTTP requests in the codebase should use this client
// instead of http.DefaultClient or http.Get/http.Post.
// It blocks requests to private, loopback, link-local, and unspecified IPs.
package safehttp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

// ErrBlockedAddress is returned when a request targets a blocked IP range.
var ErrBlockedAddress = errors.New("request blocked: target resolves to private/loopback/link-local address")

// NewClient creates an HTTP client with SSRF protections:
//   - Custom dialer that blocks private/loopback/link-local IPs
//   - Configurable timeout
//   - Redirect limit (max 5)
func NewClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, fmt.Errorf("invalid address %q: %w", addr, err)
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, fmt.Errorf("DNS lookup failed for %q: %w", host, err)
			}
			for _, ipAddr := range ips {
				ip := ipAddr.IP
				if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
					ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
					return nil, fmt.Errorf("%w: %s resolved to %s", ErrBlockedAddress, host, ip)
				}
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(host, port))
		},
		TLSHandshakeTimeout: 10 * time.Second,
		MaxIdleConns:        10,
		IdleConnTimeout:     60 * time.Second,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
}

// AllowlistClient creates a client that only allows requests to specific hostnames.
// This is the strictest option — use for payment/API integrations with known endpoints.
func AllowlistClient(allowedHosts []string, timeout time.Duration) *http.Client {
	hostSet := make(map[string]bool, len(allowedHosts))
	for _, h := range allowedHosts {
		hostSet[h] = true
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			if !hostSet[host] {
				return nil, fmt.Errorf("host %q not in allowlist", host)
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(host, port))
		},
		TLSHandshakeTimeout: 10 * time.Second,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // no redirects for payment APIs
		},
	}
}
