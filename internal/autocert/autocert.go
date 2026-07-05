// Package autocert provides automatic TLS certificate management using Let's Encrypt
// via the ACME protocol. It supports both TLS-ALPN-01 and HTTP-01 challenges.
package autocert

import (
	"errors"
	"log"
	"net/http"

	"golang.org/x/crypto/acme/autocert"
)

// DefaultCertDir is the default directory for storing certificates.
const DefaultCertDir = "/opt/koris/certs/autocert/"

// TLSConfig holds configuration for automatic TLS certificate management.
type TLSConfig struct {
	// Enabled controls whether automatic TLS is active.
	Enabled bool

	// Domain is the domain name for the certificate (e.g., "panel.example.com").
	Domain string

	// Email is the contact email for Let's Encrypt registration.
	Email string

	// CertDir is the directory for caching certificates.
	// Defaults to DefaultCertDir if empty.
	CertDir string

	// HTTPChallenge enables the HTTP-01 challenge solver on port 80.
	// When false, only TLS-ALPN-01 is used.
	HTTPChallenge bool
}

// NewManager creates and configures an autocert.Manager based on the provided TLSConfig.
// Returns an error if the config is invalid (e.g., empty domain when enabled).
func NewManager(cfg TLSConfig) (*autocert.Manager, error) {
	if !cfg.Enabled {
		return nil, errors.New("autocert: TLS is not enabled")
	}
	if cfg.Domain == "" {
		return nil, errors.New("autocert: domain is required")
	}

	certDir := cfg.CertDir
	if certDir == "" {
		certDir = DefaultCertDir
	}

	m := &autocert.Manager{
		Cache:      autocert.DirCache(certDir),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(cfg.Domain),
	}

	if cfg.Email != "" {
		m.Email = cfg.Email
	}

	log.Printf("[autocert] manager created: domain=%s, certDir=%s, email=%s, httpChallenge=%v",
		cfg.Domain, certDir, cfg.Email, cfg.HTTPChallenge)

	return m, nil
}

// HTTPChallengeHandler returns an http.Handler that responds to ACME HTTP-01
// challenges and redirects all other traffic to HTTPS. This handler should be
// mounted on port 80.
//
// If fallback is nil, non-challenge requests are redirected to HTTPS.
// If fallback is provided, non-challenge requests are passed to it.
func HTTPChallengeHandler(m *autocert.Manager, fallback http.Handler) http.Handler {
	return m.HTTPHandler(fallback)
}
