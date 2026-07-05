package autocert

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/crypto/acme/autocert"
)

// ListenAndServeTLS starts the appropriate server(s) based on the TLS configuration.
//
// If TLS is enabled:
//   - Starts HTTPS on port 443 with autocert.
//   - Starts HTTP on port 80 for ACME HTTP-01 challenges and HTTPS redirect.
//
// If TLS is disabled:
//   - Starts HTTP on the given addr (current behavior, backwards-compatible).
//
// This function blocks until the server exits.
func ListenAndServeTLS(handler http.Handler, cfg TLSConfig, addr string) error {
	if !cfg.Enabled {
		// TLS disabled — plain HTTP on the configured port (existing behavior)
		log.Printf("[autocert] TLS disabled, serving HTTP on %s", addr)
		return http.ListenAndServe(addr, handler)
	}

	// Create the autocert manager
	m, err := NewManager(cfg)
	if err != nil {
		return err
	}

	// Ensure cert cache directory exists
	certDir := cfg.CertDir
	if certDir == "" {
		certDir = DefaultCertDir
	}
	if err := os.MkdirAll(certDir, 0700); err != nil {
		return err
	}

	// Build TLS config from the autocert manager
	tlsCfg := m.TLSConfig()
	tlsCfg.MinVersion = tls.VersionTLS12

	// HTTPS server on :443
	tlsSrv := &http.Server{
		Addr:      ":443",
		Handler:   handler,
		TLSConfig: tlsCfg,
	}

	// Start HTTP listener on port 80 for challenges + redirect
	go func() {
		httpHandler := buildPort80Handler(m, cfg)
		httpSrv := &http.Server{
			Addr:    ":80",
			Handler: httpHandler,
		}
		log.Printf("[autocert] HTTP challenge/redirect listener on :80")
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[autocert] HTTP server error: %v", err)
		}
	}()

	log.Printf("[autocert] starting HTTPS on :443 for domain %s", cfg.Domain)
	return tlsSrv.ListenAndServeTLS("", "")
}

// buildPort80Handler creates an HTTP handler for port 80 that handles:
// 1. ACME HTTP-01 challenges (if HTTPChallenge is enabled)
// 2. Health check endpoint at /api/health
// 3. HTTPS redirect for all other requests
func buildPort80Handler(m *autocert.Manager, cfg TLSConfig) http.Handler {
	if cfg.HTTPChallenge {
		// Use autocert's HTTPHandler which handles challenges and redirects
		// Wrap it to also handle health checks
		acmeHandler := m.HTTPHandler(nil)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Health check bypasses ACME and redirect
			if r.URL.Path == "/api/health" {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"ok":true,"service":"panel","tls":true}`))
				return
			}
			acmeHandler.ServeHTTP(w, r)
		})
	}

	// No HTTP-01 challenge — just redirect + health check
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"ok":true,"service":"panel","tls":true}`))
			return
		}
		target := "https://" + stripPort(r.Host) + r.URL.RequestURI()
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})
}

// stripPort removes the port from a host:port string.
func stripPort(host string) string {
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		return host[:idx]
	}
	return host
}
