package cli

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/anonysec/koris/internal/auth"
	"github.com/anonysec/koris/internal/db"
)

// cliHTTPClient talks to the panel on loopback. It skips TLS verification
// because the panel's certificate is often self-signed in development; the
// CLI only ever connects to 127.0.0.1, so this is an acceptable local trust.
var cliHTTPClient = &http.Client{
	Timeout: 15 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

// cliPort resolves the panel's local port from the environment, defaulting to
// 2096 (the single-panel port). PANEL_PORT is not always injected into the
// container, so fall back to PANEL_TLS_ADDR, then the default.
func cliPort() string {
	if p := os.Getenv("PANEL_PORT"); p != "" {
		return p
	}
	if ta := os.Getenv("PANEL_TLS_ADDR"); ta != "" {
		if i := strings.LastIndex(ta, ":"); i >= 0 && ta[i+1:] != "" {
			return ta[i+1:]
		}
	}
	return "2096"
}

// adminSessionCookie returns a Cookie header carrying a freshly minted admin
// session, so the CLI can call admin-only endpoints without an interactive
// login. It works because the CLI runs with the panel's own environment
// (PANEL_SESSION_SECRET + PANEL_PG_DSN) — i.e. as root or inside the panel
// container — which is exactly the local trust boundary we want. Anyone able
// to read those secrets can already control the panel, so this adds no new
// privilege, only convenience.
func adminSessionCookie() (string, error) {
	secret := os.Getenv("PANEL_SESSION_SECRET")
	if secret == "" {
		return "", fmt.Errorf("PANEL_SESSION_SECRET not set; run this via 'koris' or inside the panel container")
	}
	dsn := os.Getenv("PANEL_PG_DSN")
	if dsn == "" {
		dsn = os.Getenv("PANEL_DB_DSN")
	}
	if dsn == "" {
		return "", fmt.Errorf("PANEL_PG_DSN not set; run this via 'koris' or inside the panel container")
	}
	d, err := db.Open(dsn)
	if err != nil {
		return "", fmt.Errorf("open db: %w", err)
	}
	defer d.Close()

	var username string
	err = d.QueryRow(
		`SELECT username FROM admins WHERE is_active AND role IN ('owner','admin') ORDER BY (role='owner') DESC, id ASC LIMIT 1`,
	).Scan(&username)
	if err != nil {
		return "", fmt.Errorf("find admin: %w", err)
	}
	tok, err := auth.MakeSession(username, secret, time.Hour)
	if err != nil {
		return "", fmt.Errorf("mint session: %w", err)
	}
	return auth.AdminCookieName + "=" + tok, nil
}

// doRequest performs an HTTP request against the local panel, trying HTTPS
// first and falling back to plain HTTP (the panel serves loopback HTTP when
// its certificate fails to load). The admin session cookie is attached so
// requireAdmin endpoints accept the request.
func (c *CLI) doRequest(method, path string, body []byte) (*http.Response, error) {
	cookie, err := adminSessionCookie()
	if err != nil {
		return nil, err
	}
	port := cliPort()
	for _, scheme := range []string{"https", "http"} {
		url := scheme + "://127.0.0.1:" + port + path
		var req *http.Request
		if body != nil {
			req, err = http.NewRequest(method, url, bytes.NewReader(body))
		} else {
			req, err = http.NewRequest(method, url, nil)
		}
		if err != nil {
			return nil, err
		}
		req.Header.Set("Cookie", cookie)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := cliHTTPClient.Do(req)
		if err != nil {
			// Fall back only on connection-level failures (TLS or TCP), not on
			// HTTP error statuses — a 401/404 must reach the caller.
			if isConnFailure(err) {
				continue
			}
			return nil, err
		}
		return resp, nil
	}
	return nil, fmt.Errorf("cannot connect to panel on 127.0.0.1:%s (tried https then http): connection refused", port)
}

// isConnFailure reports whether err is a low-level TLS/TCP connection failure
// (as opposed to a successfully-established HTTP exchange that returned an
// error status).
func isConnFailure(err error) bool {
	if err == nil {
		return false
	}
	if ne, ok := err.(net.Error); ok && !ne.Timeout() {
		return true
	}
	msg := err.Error()
	for _, s := range []string{"connection refused", "tls:", "no such host", "certificate", "handshake"} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

// makeRequest performs a GET/POST request to the running panel and returns the
// response. It authenticates as a local admin (see adminSessionCookie).
func (c *CLI) makeRequest(method, path string) (*http.Response, error) {
	return c.doRequest(method, path, nil)
}

// makeRequestWithBody performs a request with a JSON body.
func (c *CLI) makeRequestWithBody(method, path string, body []byte) (*http.Response, error) {
	return c.doRequest(method, path, body)
}

// drainAndClose reads and discards the response body to allow connection reuse.
func drainAndClose(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}
