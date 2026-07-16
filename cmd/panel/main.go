package main

import (
	"github.com/anonysec/koris/internal/safepath"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/fs"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/anonysec/koris/internal/api"
	"github.com/anonysec/koris/internal/backup"
	"github.com/anonysec/koris/internal/certrotation"
	"github.com/anonysec/koris/internal/cli"
	"github.com/anonysec/koris/internal/config"
	"github.com/anonysec/koris/internal/db"
	"github.com/anonysec/koris/internal/grpcclient"
	"github.com/anonysec/koris/internal/notify"
	"github.com/anonysec/koris/internal/protocols"
	"github.com/anonysec/koris/internal/ratelimit"
	"github.com/anonysec/koris/internal/sessions"
	"github.com/anonysec/koris/internal/tui"
	"github.com/anonysec/koris/internal/worker"
	"github.com/anonysec/koris/web"

	"github.com/coreos/go-systemd/v22/daemon"
	"golang.org/x/crypto/acme/autocert"
)

// logger is the structured TUI logger used throughout the panel process.
// Initialized in main() before any other component starts.
var logger *tui.Logger

func dbNameFromDSN(dsn string) string {
	parts := strings.Split(dsn, "/")
	if len(parts) >= 2 {
		dbPart := parts[len(parts)-1]
		if i := strings.Index(dbPart, "?"); i != -1 {
			return dbPart[:i]
		}
		return dbPart
	}
	return ""
}

func startWorker(db *sql.DB) {
	notifier := notify.NewNotifier()
	ticker := time.NewTicker(time.Minute)
	var tickCount int
	go func() {
		for range ticker.C {
			tickCount++
			func() {
				defer func() {
					if r := recover(); r != nil {
						logger.Error("worker", "recovered from panic", map[string]any{"panic": r})
					}
				}()
				workerTick(db, notifier, tickCount)
			}()
		}
	}()
}

// startWatchdog parses the WATCHDOG_USEC environment variable set by systemd
// and, if present, starts a goroutine that sends WATCHDOG=1 notifications at
// half the configured interval as long as health checks (DB ping) pass.
// On non-Linux systems or when not running under systemd, this is a no-op.
func startWatchdog(database *sql.DB) {
	usecStr := os.Getenv("WATCHDOG_USEC")
	if usecStr == "" {
		return
	}
	usec, err := strconv.ParseInt(usecStr, 10, 64)
	if err != nil || usec <= 0 {
		return
	}

	interval := time.Duration(usec/2) * time.Microsecond
	logger.Info("watchdog", "starting systemd watchdog", map[string]any{"interval": interval.String()})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			if err := database.Ping(); err != nil {
				logger.Warn("watchdog", "health check failed, withholding watchdog", map[string]any{"error": err.Error()})
				continue
			}
			daemon.SdNotify(false, "WATCHDOG=1")
		}
	}()
}

func workerTick(db *sql.DB, notifier *notify.Notifier, tickCount int) {
	// Find customers whose subscriptions have expired
	// First attempt auto-renewal for eligible customers
	autoRenewRows, _ := db.Query(`
		SELECT c.id, c.username, c.plan_id, p.price, p.duration_days, COALESCE(w.credit, 0) as credit
		FROM customers c
		JOIN (SELECT username, MAX(expires_at) as max_expires FROM subscriptions WHERE status='active' GROUP BY username) s ON c.username=s.username
		JOIN plans p ON p.id = c.plan_id
		LEFT JOIN wallets w ON w.username = c.username
		WHERE c.status = 'active' AND c.auto_renew = TRUE AND s.max_expires <= NOW()
		AND COALESCE(w.credit, 0) >= p.price`)
	if autoRenewRows != nil {
		for autoRenewRows.Next() {
			var cid, planID int64
			var username string
			var price, credit float64
			var durationDays int
			if autoRenewRows.Scan(&cid, &username, &planID, &price, &durationDays, &credit) == nil {
				// Deduct from wallet and create new subscription
				db.Exec(`UPDATE wallets SET credit = credit - $1 WHERE username = $2`, price, username)
				expires := time.Now().AddDate(0, 0, durationDays)
				db.Exec(`INSERT INTO subscriptions(customer_id, username, plan_id, expires_at, status) VALUES($1,$2,$3,$4,'active')`, cid, username, planID, expires)
				db.Exec(`INSERT INTO wallet_transactions(customer_id, username, amount, type, description, actor) VALUES($1,$2,$3,$4,$5,$6)`,
					cid, username, -price, "purchase", "Auto-renewal", "system")
				logger.Info("worker", "auto-renewed customer", map[string]any{"username": username, "plan": planID, "charged": price})
				notifier.SendEvent("renewal", fmt.Sprintf("🔄 Auto-renewed: %s", username), fmt.Sprintf("Plan renewed for %d days, charged $%.2f from wallet", durationDays, price))
			}
		}
		autoRenewRows.Close()
	}

	// Find remaining expired customers (not auto-renewed)
	expRows, expErr := db.Query(`SELECT c.id, c.username, COALESCE(p.grace_days, 0) as grace_days, s.max_expires
		FROM customers c
		JOIN (SELECT username, MAX(expires_at) as max_expires FROM subscriptions WHERE status='active' GROUP BY username) s ON c.username=s.username
		LEFT JOIN plans p ON p.id = c.plan_id
		WHERE c.status IN ('active', 'limited') AND s.max_expires <= NOW()`)
	var expiringCustomerIDs []int64
	if expErr == nil {
		for expRows.Next() {
			var cid int64
			var username string
			var graceDays int
			var maxExpires time.Time
			if expRows.Scan(&cid, &username, &graceDays, &maxExpires) == nil {
				graceEnd := maxExpires.AddDate(0, 0, graceDays)
				now := time.Now()

				if graceDays > 0 && now.Before(graceEnd) {
					// Within grace period → set to 'limited' (not expired yet)
					db.Exec(`UPDATE customers SET status='limited' WHERE id=$1 AND status='active'`, cid)
				} else {
					// Grace period over (or no grace days) → expire
					db.Exec(`UPDATE customers SET status='expired' WHERE id=$1 AND status IN ('active','limited')`, cid)
					expiringCustomerIDs = append(expiringCustomerIDs, cid)
				}
			}
		}
		expRows.Close()
	}

	// Auto-revoke WireGuard peers for fully expired customers
	for _, cid := range expiringCustomerIDs {
		api.AutoRevokeWireGuardPeersByDB(db, cid)
	}

	// Usage warnings: notify admin via Telegram when users hit thresholds (80%, 95%)
	warnRows, warnErr := db.Query(`
		SELECT c.username, CAST(r.value AS BIGINT) as max_bytes, a.used
		FROM customers c
		JOIN radcheck r ON c.username=r.username AND r.attribute='Max-Data'
		JOIN (SELECT username, COALESCE(SUM(acctinputoctets+acctoutputoctets),0) AS used FROM radacct GROUP BY username) a ON c.username=a.username
		WHERE c.status='active' AND CAST(r.value AS BIGINT) > 0`)
	if warnErr == nil {
		for warnRows.Next() {
			var username string
			var maxBytes, used int64
			if warnRows.Scan(&username, &maxBytes, &used) == nil && maxBytes > 0 {
				percent := int(float64(used) / float64(maxBytes) * 100)
				// Notify at 80% and 95% (check if not already notified via events)
				if percent >= 95 {
					var already int
					db.QueryRow(`SELECT COUNT(*) FROM events WHERE related=$1 AND type='data_warning' AND title LIKE '%95%' AND created_at > NOW() - INTERVAL '1 day'`, username).Scan(&already)
					if already == 0 {
						notifier.SendEvent("data_warning", fmt.Sprintf("⚠️ %s at 95%% data", username), fmt.Sprintf("User %s has used 95%% of their data limit", username))
						db.Exec(`INSERT INTO events(type,severity,title,message,actor,related) VALUES('data_warning','warning',$1,$2,$3,$4)`, fmt.Sprintf("%s at 95%% data", username), fmt.Sprintf("Used %d%% of data limit", percent), "system", username)
					}
				} else if percent >= 80 {
					var already int
					db.QueryRow(`SELECT COUNT(*) FROM events WHERE related=$1 AND type='data_warning' AND title LIKE '%80%' AND created_at > NOW() - INTERVAL '1 day'`, username).Scan(&already)
					if already == 0 {
						notifier.SendEvent("data_warning", fmt.Sprintf("📊 %s at 80%% data", username), fmt.Sprintf("User %s has used 80%% of their data limit", username))
						db.Exec(`INSERT INTO events(type,severity,title,message,actor,related) VALUES('data_warning','info',$1,$2,$3,$4)`, fmt.Sprintf("%s at 80%% data", username), fmt.Sprintf("Used %d%% of data limit", percent), "system", username)
					}
				}
			}
		}
		warnRows.Close()
	}

	if _, err := db.Exec(`UPDATE customers SET status='limited' FROM radcheck r JOIN (SELECT username, COALESCE(SUM(acctinputoctets+acctoutputoctets),0) AS used FROM radacct GROUP BY username) a ON r.username=a.username WHERE customers.username=r.username AND r.attribute='Max-Data' AND customers.status='active' AND CAST(r.value AS BIGINT) > 0 AND a.used >= CAST(r.value AS BIGINT)`); err != nil {
		logger.Error("worker", "data limit enforcement failed", map[string]any{"error": err.Error()})
	}
	_, _ = db.Exec(`UPDATE radacct SET acctstoptime=NOW(), acctterminatecause='Stalled session' WHERE acctstoptime IS NULL AND acctupdatetime < (NOW() - INTERVAL '5 minutes')`)

	// Mark nodes offline, record downtime for SLA tracking, and notify via Telegram
	rows, err := db.Query(`SELECT id, name, public_ip FROM nodes WHERE status IN('online','stale') AND last_seen_at < (NOW() - INTERVAL '5 minutes')`)
	if err == nil {
		for rows.Next() {
			var nodeID int64
			var name, ip string
			if rows.Scan(&nodeID, &name, &ip) == nil {
				api.RecordNodeDowntime(db, nodeID, "Node went offline (no push for 5+ minutes)")
				notifier.NotifyNodeOffline(name, ip)
			}
		}
		rows.Close()
	}
	_, _ = db.Exec(`UPDATE nodes SET status='offline' WHERE status IN('online','stale') AND last_seen_at < (NOW() - INTERVAL '5 minutes')`)

	// Data retention: prune old snapshots to prevent unbounded growth
	// Keep last 7 days of node_usage_snapshots, last 24h of user_bandwidth_snapshots
	_, _ = db.Exec(`DELETE FROM node_usage_snapshots WHERE created_at < NOW() - INTERVAL '7 days'`)
	_, _ = db.Exec(`DELETE FROM user_bandwidth_snapshots WHERE created_at < NOW() - INTERVAL '24 hours'`)

	// History retention: prune old radacct and wallet_transactions
	// Runs only at midnight (00:00) to avoid unnecessary load
	now := time.Now()
	if now.Hour() == 0 && now.Minute() == 0 {
		retentionDays := 45
		var retVal string
		if db.QueryRow(`SELECT setting_value FROM panel_settings WHERE setting_key='history_retention_days'`).Scan(&retVal) == nil {
			if d, err := strconv.Atoi(retVal); err == nil && d > 0 {
				retentionDays = d
			}
		}
		_, _ = db.Exec(`DELETE FROM radacct WHERE acctstoptime IS NOT NULL AND acctstoptime < NOW() - INTERVAL '1 day' * $1`, retentionDays)
		_, _ = db.Exec(`DELETE FROM wallet_transactions WHERE created_at < NOW() - INTERVAL '1 day' * $1`, retentionDays)
		logger.Info("worker", "history retention: purged old records", map[string]any{"retention_days": retentionDays})
	}

	// Node resource alerts: check CPU/RAM/disk against per-node thresholds
	api.CheckNodeAlerts(db, notifier.Send)

	// Bandwidth quota alerts: check usage against configured thresholds
	api.CheckBandwidthQuotas(db, notifier.Send)

	// Bandwidth quota reset: reset current_usage_gb on the configured reset_day
	api.ResetBandwidthQuotas(db)

	// Protocol health checks: TCP connect test for each enabled protocol per node
	protocols.CheckProtocolHealth(db)

	// Node bandwidth quotas (Server-level): check nodes table bandwidth columns
	api.CheckNodeBandwidthQuotas(db, notifier.Send)

	// Reset monthly bandwidth counters on 1st of month
	api.ResetMonthlyNodeBandwidth(db)

	// Pending update health: fail stale update_agent tasks
	api.CheckPendingUpdateHealth(db, notifier.Send)

	// Excluded-feature worker operations (billing, SLA, teleproxy, load balancing)
	// No-op in lite build.
	workerTickExcluded(db, notifier, tickCount)
}

// loadBotConfigFromDB reads telegram_token and telegram_chat_id from the panel_settings table.
// Returns empty values if the table doesn't exist or the keys are not set.
func loadBotConfigFromDB(database *sql.DB) (token string, chatIDs []int64) {
	rows, err := database.Query(`SELECT setting_key, setting_value FROM panel_settings WHERE setting_key IN ('telegram_token', 'telegram_chat_id')`)
	if err != nil {
		// Table might not exist yet on first run
		return "", nil
	}
	defer rows.Close()
	for rows.Next() {
		var key, val string
		if err := rows.Scan(&key, &val); err != nil {
			continue
		}
		switch key {
		case "telegram_token":
			token = strings.TrimSpace(val)
		case "telegram_chat_id":
			for _, s := range strings.Split(val, ",") {
				s = strings.TrimSpace(s)
				if id, err := strconv.ParseInt(s, 10, 64); err == nil && id != 0 {
					chatIDs = append(chatIDs, id)
				}
			}
		}
	}
	return
}

// startSocketListener starts an HTTP server on a Unix domain socket for local
// CLI communication. On Windows this is a no-op. Returns the listener (for
// shutdown) and any error. The caller should serve in a goroutine.
func startSocketListener(handler http.Handler, socketPath string) (net.Listener, error) {
	if runtime.GOOS == "windows" {
		return nil, nil
	}

	// Remove stale socket file from a previous run.
	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("remove old socket: %w", err)
	}

	// Ensure the parent directory exists.
	if dir := filepath.Dir(socketPath); dir != "" {
		os.MkdirAll(dir, 0755)
	}

	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("listen unix %s: %w", socketPath, err)
	}

	// Set socket file permissions to 0660 (owner + group read/write).
	if err := os.Chmod(socketPath, 0660); err != nil {
		ln.Close()
		return nil, fmt.Errorf("chmod socket: %w", err)
	}

	return ln, nil
}

func parseCertInfo(certPath string) (expiry string, issuer string) {
	data, err := safepath.ReadFile(certPath)
	if err != nil {
		return "", ""
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return "", ""
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", ""
	}
	expiry = cert.NotAfter.Format(time.RFC3339)
	issuer = cert.Issuer.CommonName
	if issuer == "" && len(cert.Issuer.Organization) > 0 {
		issuer = cert.Issuer.Organization[0]
	}
	return
}

func isCLICommand(arg string) bool {
	commands := map[string]bool{
		"status":  true,
		"nodes":   true,
		"users":   true,
		"admin":   true,
		"cleanup": true,
		"workers": true,
		"logs":    true,
		"update":  true,
		"cert":    true,
		"help":    true,
		"--help":  true,
		"--json":  true,
	}
	return commands[arg]
}

// isLoopback reports whether the given address string (host:port or just IP)
// refers to a loopback interface (127.0.0.0/8 or ::1).
func isLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		// addr might not have a port; treat the whole string as host
		host = addr
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

// redirectToHTTPS is a middleware that redirects non-loopback HTTP requests
// to their HTTPS equivalent with a 301 Moved Permanently status.
// Requests from loopback addresses (127.0.0.1, ::1) are served normally
// without redirect, allowing local tools and health checks to use HTTP.
func redirectToHTTPS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil && !isLoopback(r.RemoteAddr) {
			host := r.Host
			// Strip port from host if present for clean HTTPS URL
			if idx := strings.LastIndex(host, ":"); idx != -1 {
				host = host[:idx]
			}
			target := "https://" + host + r.URL.RequestURI()
			http.Redirect(w, r, target, http.StatusMovedPermanently)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// startTLSListener starts HTTPS (and loopback HTTP) listeners based on config.
// It supports three TLS modes via the TLS Manager:
//   - Manual: user-provided cert/key files with validation at startup
//   - ACME (Let's Encrypt): automatic cert provisioning, requires PANEL_TLS_DOMAIN
//   - SelfSigned: auto-generated cert (365 days), logs a warning
//
// HTTP listener binds to 127.0.0.1:8080 (loopback only) with redirectToHTTPS
// middleware for defense-in-depth. HTTPS listener binds to 0.0.0.0:443 with
// minimum TLS 1.2.
//
// This function blocks (runs the HTTPS server). It should be called from a goroutine
// or as the final blocking call in main.
func startTLSListener(handler http.Handler, cfg config.Config) {
	domain := os.Getenv("PANEL_DOMAIN")

	// Default install mode: plaintext HTTP bound to 127.0.0.1 only. No cert is
	// generated or required. The operator installs a cert (selfsigned/acme/manual)
	// to switch to public HTTPS. Public plaintext is never served.
	if cfg.TLSMode == "disabled" {
		serveLoopbackHTTP(handler, cfg.TLSAddr)
		return
	}

	// Dev-only self-signed mode: generate a self-signed cert if one isn't present.
	// Browsers will warn — this is for local/dev use only. Production must use a
	// real cert (manual cert files, or ACME for a domain).
	if cfg.TLSMode == "selfsigned" && (!safepath.Exists(cfg.TLSCert) || !safepath.Exists(cfg.TLSKey)) {
		logger.Warn("tls", "SELF-SIGNED certificate (DEV ONLY) — browsers will warn; use a real cert (domain/IP/custom) in production", map[string]any{"cert": cfg.TLSCert})
		if err := generateSelfSignedCert(cfg.TLSCert, cfg.TLSKey, domain); err != nil {
			logger.Error("tls", "failed to generate self-signed certificate", map[string]any{"error": err.Error()})
			logger.Warn("tls", "falling back to loopback-only HTTP so an admin can fix the certificate", map[string]any{"addr": "127.0.0.1"})
			serveLoopbackHTTP(handler, cfg.TLSAddr)
			return
		}
	}

	// Mode 1: Custom cert/key files provided
	if safepath.Exists(cfg.TLSCert) && safepath.Exists(cfg.TLSKey) {
		logger.Info("tls", "starting HTTPS with custom cert/key", map[string]any{
			"cert": cfg.TLSCert,
			"key":  cfg.TLSKey,
			"addr": cfg.TLSAddr,
		})

		// Single-port model: HTTPS occupies the one port, so no separate loopback
		// HTTP listener here. Plain HTTP is started only as the cert-failure
		// fallback (serveLoopbackHTTP) further below.
		tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
		tlsCfg.Certificates = make([]tls.Certificate, 1)
		var err error
		tlsCfg.Certificates[0], err = tls.LoadX509KeyPair(cfg.TLSCert, cfg.TLSKey)
		if err != nil {
			logger.Error("tls", "failed to load TLS certificate", map[string]any{"error": err.Error()})
			logger.Warn("tls", "falling back to loopback-only HTTP so an admin can fix the certificate")
			serveLoopbackHTTP(handler, cfg.TLSAddr)
			return
		}

		httpsListener, err := tls.Listen("tcp", cfg.TLSAddr, tlsCfg)
		if err != nil {
			logger.Error("tls", "failed to bind HTTPS listener", map[string]any{"addr": cfg.TLSAddr, "error": err.Error()})
			logger.Warn("tls", "falling back to loopback-only HTTP so an admin can fix the certificate")
			serveLoopbackHTTP(handler, cfg.TLSAddr)
			return
		}
		logger.Info("tls", "HTTPS listener started", map[string]any{"addr": cfg.TLSAddr})

		customSrv := &http.Server{
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      60 * time.Second,
			IdleTimeout:       120 * time.Second,
		}
		if err := customSrv.Serve(httpsListener); err != nil {
			logger.Error("tls", "HTTPS server failed (custom cert)", map[string]any{"error": err.Error()})
		}
		return
	}

	// Mode 2: Autocert (Let's Encrypt / ZeroSSL) — only when explicitly
	// requested (acme) OR when a REAL domain is configured. A bare "localhost"
	// or empty domain must NOT trigger ACME (it can't issue for localhost),
	// so it falls through to the loopback-only HTTP fallback instead.
	isAcmeMode := cfg.TLSMode == "acme" ||
		(cfg.TLSMode != "selfsigned" && cfg.TLSMode != "manual" && domain != "" && domain != "localhost")
	if !isAcmeMode {
		if cfg.TLSMode == "manual" {
			logger.Error("tls", "manual TLS mode but PANEL_TLS_CERT/PANEL_TLS_KEY not found", map[string]any{"cert": cfg.TLSCert, "key": cfg.TLSKey})
		}
		logger.Error("tls", "no usable certificate and no ACME domain — cannot start public HTTPS")
		logger.Warn("tls", "falling back to loopback-only HTTP so an admin can fix the certificate", map[string]any{"addr": "127.0.0.1"})
		serveLoopbackHTTP(handler, cfg.TLSAddr)
		return
	}

	// Ensure cert cache directory exists
	certDir := cfg.TLSCertDir
	if err := os.MkdirAll(certDir, 0700); err != nil {
		logger.Error("tls", "failed to create cert cache dir", map[string]any{"dir": certDir, "error": err.Error()})
		return
	}

	m := &autocert.Manager{
		Cache:      autocert.DirCache(certDir),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(domain),
	}

	tlsCfg := m.TLSConfig()
	tlsCfg.MinVersion = tls.VersionTLS12

	tlsSrv := &http.Server{
		Addr:              cfg.TLSAddr,
		Handler:           handler,
		TLSConfig:         tlsCfg,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	logger.Info("tls", "starting HTTPS with Let's Encrypt autocert", map[string]any{
		"domain":   domain,
		"addr":     cfg.TLSAddr,
		"cert_dir": certDir,
	})

	// Start HTTP challenge handler on :80 for ACME HTTP-01 challenges.
	// This must listen on all interfaces for Let's Encrypt validation.
	go func() {
		httpSrv := &http.Server{
			Addr:              ":80",
			Handler:           m.HTTPHandler(nil),
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      60 * time.Second,
			IdleTimeout:       120 * time.Second,
		}
		if err := httpSrv.ListenAndServe(); err != nil {
			logger.Error("tls", "HTTP challenge/redirect server error", map[string]any{"error": err.Error()})
		}
	}()

	// Start loopback-only HTTP listener for local access (skip when HTTP and HTTPS
	// share one port — HTTPS already occupies it).
	if !samePort(cfg.Addr, cfg.TLSAddr) {
		go startHTTPLoopback(handler, cfg.Addr)
	}

	if err := tlsSrv.ListenAndServeTLS("", ""); err != nil {
		logger.Error("tls", "HTTPS server failed (autocert)", map[string]any{"error": err.Error()})
	}
}

// serveLoopbackHTTP is the mandatory-HTTPS fallback: when no certificate can be
// loaded, public HTTPS is unavailable and this serves plain HTTP bound to loopback
// (127.0.0.1) ONLY on the panel port. Plaintext HTTP is never exposed off-host —
// an admin reaches it locally (on a bare-metal host directly; in Docker via
// `docker exec` or the koris CLI) to fix the certificate. This blocks (final call).
func serveLoopbackHTTP(handler http.Handler, addr string) {
	_, port, err := net.SplitHostPort(addr)
	if err != nil || port == "" {
		port = "2096"
	}
	httpAddr := "127.0.0.1:" + port
	logger.Warn("http", "serving loopback-only HTTP fallback on the panel port (cert unavailable) — local access only; fix the certificate and restart", map[string]any{"addr": httpAddr})
	srv := &http.Server{
		Addr:              httpAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("http", "loopback HTTP fallback server failed", map[string]any{"error": err.Error()})
	}
}

// generateSelfSignedCert writes a self-signed ECDSA cert/key pair to certPath and
// keyPath (creating parent dirs). DEV ONLY — browsers will warn. The cert includes
// localhost, 127.0.0.1 and ::1, plus the given host (domain or IP) if provided.
func generateSelfSignedCert(certPath, keyPath, host string) error {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generate key: %w", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("serial: %w", err)
	}
	tmpl := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "koris-panel (self-signed, dev)"},
		NotBefore:             notBeforeEpoch(),
		NotAfter:              notBeforeEpoch().AddDate(1, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
	}
	if host != "" && host != "localhost" {
		if ip := net.ParseIP(host); ip != nil {
			tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
		} else {
			tmpl.DNSNames = append(tmpl.DNSNames, host)
		}
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		return fmt.Errorf("create cert: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(certPath), 0o755); err != nil {
		return fmt.Errorf("mkdir cert dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o755); err != nil {
		return fmt.Errorf("mkdir key dir: %w", err)
	}
	certOut, err := os.Create(certPath)
	if err != nil {
		return fmt.Errorf("create cert file: %w", err)
	}
	defer certOut.Close()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		return fmt.Errorf("write cert: %w", err)
	}
	keyBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return fmt.Errorf("marshal key: %w", err)
	}
	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create key file: %w", err)
	}
	defer keyOut.Close()
	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return fmt.Errorf("write key: %w", err)
	}
	return nil
}

// notBeforeEpoch returns a fixed backdated NotBefore. time.Now via the injected
// clock isn't available here; use the process start reference minus a day of skew.
func notBeforeEpoch() time.Time { return time.Now().Add(-24 * time.Hour) }

// portOf returns the port portion of an address (":2096" -> "2096",
// "0.0.0.0:2096" -> "2096"), or "" if it can't be parsed.
func portOf(addr string) string {
	if _, port, err := net.SplitHostPort(addr); err == nil {
		return port
	}
	return ""
}

// samePort reports whether two listen addresses resolve to the same port.
func samePort(a, b string) bool { return portOf(a) != "" && portOf(a) == portOf(b) }

// startHTTPLoopback starts an HTTP server bound to loopback only (127.0.0.1)
// with the redirectToHTTPS middleware applied. This ensures:
// - Local tools and health checks can use HTTP without redirect
// - Any non-loopback request (if it somehow reaches this listener) gets a 301 to HTTPS
func startHTTPLoopback(handler http.Handler, addr string) {
	// Ensure the HTTP listener binds to loopback only
	httpAddr := addr
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		port = "8080"
	}
	// Force loopback binding regardless of configured address
	httpAddr = "127.0.0.1:" + port

	httpListener, err := net.Listen("tcp", httpAddr)
	if err != nil {
		logger.Error("http", "failed to bind HTTP loopback listener", map[string]any{"addr": httpAddr, "error": err.Error()})
		return
	}

	logger.Info("http", "HTTP listener started (loopback only)", map[string]any{"addr": httpAddr})

	// Apply redirectToHTTPS middleware as defense-in-depth
	httpHandler := redirectToHTTPS(handler)
	loopbackSrv := &http.Server{
		Handler:           httpHandler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	if err := loopbackSrv.Serve(httpListener); err != nil {
		logger.Error("http", "HTTP loopback server error", map[string]any{"error": err.Error()})
	}
}

func main() {
	// Check for CLI mode first — before any heavy initialization.
	if len(os.Args) > 1 && isCLICommand(os.Args[1]) {
		c := cli.New(cli.WithOutput(os.Stdout))
		cli.RegisterDefaultCommands(c)
		if err := c.Execute(os.Args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Multi-worker manager mode detection.
	// If PANEL_WORKERS is set to a value > 1 (or "auto") and we are NOT a
	// child worker process, this process becomes the manager that forks and
	// monitors worker children. Worker children fall through to normal startup.
	if panelWorkers := os.Getenv("PANEL_WORKERS"); panelWorkers != "" && panelWorkers != "1" {
		isChild, _ := worker.IsWorkerProcess()
		if !isChild {
			// We are the master/manager process — fork workers and monitor.
			numWorkers := 0 // 0 = auto (resolved by Config.ResolvedWorkers)
			if n, err := strconv.Atoi(panelWorkers); err == nil && n > 1 {
				numWorkers = n
			}
			port := os.Getenv("PANEL_PORT")
			if port == "" {
				port = "8088"
			}
			graceSec := 30
			if gs := os.Getenv("PANEL_GRACEFUL_WAIT"); gs != "" {
				if v, err := strconv.Atoi(gs); err == nil && v > 0 {
					graceSec = v
				}
			}
			cfg := worker.Config{
				NumWorkers:   numWorkers,
				Addr:         ":" + port,
				GracefulWait: time.Duration(graceSec) * time.Second,
				MaxRestarts:  5,
			}
			mgr := worker.NewManager(cfg)
			ctx, cancel := context.WithCancel(context.Background())

			// Handle SIGINT/SIGTERM for the manager process.
			go func() {
				sigCh := make(chan os.Signal, 1)
				signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
				<-sigCh
				cancel()
				mgr.Stop()
			}()

			if err := mgr.Start(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "manager error: %v\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		}
		// If we ARE a worker child, fall through to normal server startup.
	}

	// Runtime resource limits (GOMAXPROCS, GOGC, GOMEMLIMIT) are intentionally
	// left at the Go runtime defaults. On Go 1.25 the runtime automatically
	// honors the container's cgroup CPU and memory limits, so overriding them
	// here would ignore that awareness and either over-subscribe CPUs or force
	// needless GC pressure. Set GOMAXPROCS/GOGC/GOMEMLIMIT in the environment to
	// override per-host if required.

	cfg := config.Load()

	// Initialize structured TUI logger early — all subsequent logging uses this.
	logger = tui.New(tui.WithLevel(tui.LevelInfo))

	// Log worker configuration at startup.
	logger.Info("worker", "worker configuration", map[string]any{
		"workers":   cfg.Workers,
		"worker_id": cfg.WorkerID,
	})

	database, err := db.Open(cfg.DBDSN)
	if err != nil {
		logger.Error("main", "failed to open database", map[string]any{"error": err.Error()})
		os.Exit(1)
	}
	migDir := os.Getenv("PANEL_MIGRATIONS")
	if err := db.Migrate(database, migDir); err != nil {
		logger.Error("main", "database migration failed", map[string]any{"error": err.Error()})
		os.Exit(1)
	}
	// Start background ticker. In multi-worker mode only the leader worker
	// runs the ticker to avoid duplicate billing/cleanup. Leader election is
	// via an exclusive file lock — only one worker process can acquire it.
	if isChild, _ := worker.IsWorkerProcess(); isChild {
		ll := worker.NewLeaderLock("")
		if ll.TryAcquire() {
			logger.Info("worker", "acquired leader lock — running background ticker", map[string]any{"pid": os.Getpid()})
			startWorker(database)
		} else {
			logger.Info("worker", "not leader — skipping background ticker", map[string]any{"pid": os.Getpid()})
		}
	} else {
		// Single-process mode: always run the ticker.
		startWorker(database)
	}

	// Initialize backup service
	backupCfg := backup.LoadConfigFromDB(database)
	backupService := backup.New(database, backupCfg)
	backupService.StartScheduler()

	// ─── gRPC Client Subsystem ─────────────────────────────────────────────
	// Initialize the gRPC connection pool, node registry, status monitor,
	// metrics consumer, user sync, traffic collector, and alerter.
	// Failed node connections do NOT block boot.
	grpcCtx, grpcCancel := context.WithCancel(context.Background())
	grpcSub, grpcErr := initGRPCSubsystem(grpcCtx, database, cfg, logger)
	if grpcErr != nil {
		logger.Warn("grpc-client", "gRPC subsystem initialization failed (continuing without gRPC)", map[string]any{"error": grpcErr.Error()})
	} else {
		logger.Info("grpc-client", "gRPC subsystem ready", map[string]any{
			"nodes": len(grpcSub.Pool.All()),
		})
	}

	// Start certificate rotation worker
	certEventFn := func(eventType, severity, title, message string) {
		_, _ = database.Exec(`INSERT INTO events(type,severity,title,message,actor,related) VALUES($1,$2,$3,$4,$5,$6)`,
			eventType, severity, title, message, "system", "")
	}
	certWorker := certrotation.New(database, certEventFn)
	certWorker.Start()

	// Wire gRPC cert pusher into the cert rotation worker if gRPC subsystem is available.
	// This allows cert rotation to use direct gRPC SetCertificates calls.
	if grpcSub != nil {
		certWorker.SetPusher(grpcclient.NewCertManager(grpcSub.Pool))
	}

	// Start session enforcer (kills excess connections every 30s)
	enforcer := sessions.NewEnforcer(database)
	enforcer.Start()
	logger.Info("main", "session enforcer started")

	srv := api.New(database, cfg)
	// Embed pre-built frontend assets into the binary — no external www/ needed
	adminFS, _ := fs.Sub(web.AdminFS, "admin/www")
	portalFS, _ := fs.Sub(web.PortalFS, "portal/www")
	landingFS, _ := fs.Sub(web.LandingFS, "landing/www")
	srv.AdminEmbedFS = adminFS
	srv.PortalEmbedFS = portalFS
	srv.LandingEmbedFS = landingFS
	srv.BackupService = backupService

	// Wire gRPC subsystem into the API server if it initialized successfully.
	if grpcSub != nil {
		srv.GRPCPool = grpcSub.Pool
		srv.NodeRegistry = grpcSub.Registry
		srv.FirewallMgr = grpcclient.NewFirewallManager(grpcSub.Pool)
		srv.CoreMgr = grpcclient.NewCoreManager(grpcSub.Pool, grpcSub.Store)
		srv.TunnelMgr = grpcclient.NewTunnelManager(grpcSub.Pool, grpcSub.Store)
		srv.CertMgr = grpcclient.NewCertManager(grpcSub.Pool)
		srv.ClientCertSvc = grpcclient.NewClientCertService(grpcSub.Pool)
		srv.SessionMgr = grpcclient.NewSessionManager(grpcSub.Pool)
		srv.UserSync = grpcSub.UserSync
		srv.TrafficCollector = grpcSub.TrafficCollector
		srv.GRPCStore = grpcSub.Store

		// Wire WebSocket metrics broadcasting into pool status changes
		srv.RegisterWSMetricsBroadcast()
	}

	// Initialize excluded services (billing, support, teleproxy, antidpi, payment)
	// No-op in lite build.
	initExcludedServices(srv, database)
	srv.LogEntries = func(n int) []tui.LogEntry {
		return logger.LastEntries(n)
	}

	mux := srv.Routes()

	// Start Telegram bot (no-op in lite build)
	startBot(database, srv, mux)

	// Certificate status endpoint
	mux.HandleFunc("/api/admin/cert-status", srv.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		certPath := cfg.TLSCert
		keyPath := cfg.TLSKey
		certExists := safepath.Exists(certPath)
		keyExists := safepath.Exists(keyPath)
		tlsActive := certExists && keyExists && r.TLS != nil
		result := map[string]any{
			"ok":          true,
			"cert_exists": certExists,
			"key_exists":  keyExists,
			"tls_active":  tlsActive,
			"tls_addr":    cfg.TLSAddr,
			"cert_path":   certPath,
			"key_path":    keyPath,
			"expiry":      "",
			"issuer":      "",
		}
		if certExists {
			expiry, issuer := parseCertInfo(certPath)
			result["expiry"] = expiry
			result["issuer"] = issuer
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}))

	// Certificate upload endpoint
	mux.HandleFunc("/api/admin/cert-upload", srv.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"ok":false,"error":"invalid multipart form"}`))
			return
		}
		certFile, _, err := r.FormFile("cert")
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"ok":false,"error":"cert file required"}`))
			return
		}
		defer certFile.Close()
		keyFile, _, err := r.FormFile("key")
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"ok":false,"error":"key file required"}`))
			return
		}
		defer keyFile.Close()

		certData, _ := io.ReadAll(certFile)
		keyData, _ := io.ReadAll(keyFile)

		// Validate that cert and key form a valid TLS pair
		if _, err := tls.X509KeyPair(certData, keyData); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "invalid certificate/key pair: " + err.Error()})
			return
		}

		// Save to the configured paths
		certPath := cfg.TLSCert
		keyPath := cfg.TLSKey
		os.MkdirAll(filepath.Dir(certPath), 0755)
		if err := os.WriteFile(certPath, certData, 0600); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"ok":false,"error":"failed to save cert"}`))
			return
		}
		if err := os.WriteFile(keyPath, keyData, 0600); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"ok":false,"error":"failed to save key"}`))
			return
		}

		// Parse cert info for response
		expiry, issuer := parseCertInfo(certPath)

		logger.Info("tls", "new certificate uploaded — restart required for HTTPS", map[string]any{"expiry": expiry, "issuer": issuer})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":               true,
			"message":          "Certificate saved. Restart the panel service to enable HTTPS.",
			"restart_required": true,
			"expiry":           expiry,
			"issuer":           issuer,
			"tls_addr":         cfg.TLSAddr,
		})
	}))

	// ─── Ready Message ─────────────────────────────────────────────────────
	nodeCount := 0
	if grpcSub != nil {
		nodeCount = len(grpcSub.Pool.All())
	}
	logger.Info("ready", "panel startup complete", map[string]any{
		"version": cfg.Version,
		"addr":    cfg.Addr,
		"nodes":   nodeCount,
		"workers": cfg.Workers,
	})

	// Notify systemd that the service is ready (no-op on non-Linux or without systemd).
	daemon.SdNotify(false, daemon.SdNotifyReady)
	logger.Info("main", "sent sd_notify ready")

	// Start systemd watchdog heartbeat if WATCHDOG_USEC is configured.
	startWatchdog(database)

	// Rate limiter: 30 requests/sec per IP, burst 60
	limiter := ratelimit.New(30, 60, cfg.TrustedProxies)

	// Apply no-cache middleware on API responses
	handler := api.SecurityHeadersMiddleware(api.NoCacheMiddleware(mux))

	// ─── Unix Socket Listener (Linux only, for local CLI) ──────────────────
	socketPath := os.Getenv("PANEL_SOCKET_PATH")
	if socketPath == "" {
		socketPath = "/var/run/panel.sock"
	}

	socketLn, sockErr := startSocketListener(handler, socketPath)
	if sockErr != nil {
		logger.Warn("main", "unix socket listener failed (CLI will use HTTP fallback)", map[string]any{"error": sockErr.Error(), "path": socketPath})
	} else if socketLn != nil {
		logger.Info("main", "unix socket listener started", map[string]any{"path": socketPath})
		go func() {
			socketSrv := &http.Server{
				Handler:           handler,
				ReadHeaderTimeout: 10 * time.Second,
				ReadTimeout:       30 * time.Second,
				WriteTimeout:      60 * time.Second,
				IdleTimeout:       120 * time.Second,
			}
			if err := socketSrv.Serve(socketLn); err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
				logger.Error("main", "unix socket serve error", map[string]any{"error": err.Error()})
			}
		}()
	}

	// Graceful shutdown: clean up socket file on SIGINT/SIGTERM.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logger.Info("main", "shutting down...")
		if grpcSub != nil {
			stopGRPCSubsystem(grpcSub)
			grpcCancel()
		}
		if socketLn != nil {
			socketLn.Close()
			os.Remove(socketPath)
		}
		limiter.Stop()
		os.Exit(0)
	}()

	// Start server: use TLS if explicitly enabled via PANEL_TLS_ENABLED=true,
	// OR if cert and key files exist AND the panel is NOT behind a reverse proxy.
	// Detection: if PANEL_ADDR is bound to loopback (127.0.0.1), assume a reverse proxy handles TLS.
	// To force direct TLS even on loopback, set PANEL_TLS_DIRECT=true.
	if cfg.TLSEnabled {
		// New built-in TLS mode: autocert or custom cert/key
		logger.Info("tls", "PANEL_TLS_ENABLED=true — starting built-in TLS", map[string]any{"addr": cfg.TLSAddr})

		// Start loopback-only HTTP listener with redirect middleware (skip when HTTP
		// and HTTPS share one port — startTLSListener owns that port for HTTPS).
		if !samePort(cfg.Addr, cfg.TLSAddr) {
			go startHTTPLoopback(limiter.Middleware(handler), cfg.Addr)
		}

		// startTLSListener blocks — it handles autocert or custom cert mode
		startTLSListener(limiter.Middleware(handler), cfg)
	} else {
		tlsCert := cfg.TLSCert
		tlsKey := cfg.TLSKey
		behindProxy := strings.HasPrefix(cfg.Addr, "127.") || strings.HasPrefix(cfg.Addr, "localhost")
		forceTLS := strings.ToLower(os.Getenv("PANEL_TLS_DIRECT")) == "true"

		// Default install mode: plaintext HTTP bound to 127.0.0.1 only, even if
		// a cert happens to exist on disk. The operator installs a cert to switch
		// to public HTTPS. Public plaintext is never served.
		if cfg.TLSMode == "disabled" {
			serveLoopbackHTTP(limiter.Middleware(handler), cfg.TLSAddr)
			return
		}

		if safepath.Exists(tlsCert) && safepath.Exists(tlsKey) && (!behindProxy || forceTLS) {
			logger.Info("tls", "TLS enabled (legacy mode)", map[string]any{"cert": tlsCert, "key": tlsKey, "addr": cfg.TLSAddr})
			logger.Info("tls", "HTTP loopback with redirect configured", map[string]any{"addr": "127.0.0.1:" + strings.TrimPrefix(cfg.Addr, ":")})

			// Start loopback-only HTTP server with redirect middleware (skip on shared port)
			if !samePort(cfg.Addr, cfg.TLSAddr) {
				go startHTTPLoopback(limiter.Middleware(handler), cfg.Addr)
			}

			// Start HTTPS server on all interfaces with min TLS 1.2
			tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
			tlsCfg.Certificates = make([]tls.Certificate, 1)
			var certErr error
			tlsCfg.Certificates[0], certErr = tls.LoadX509KeyPair(tlsCert, tlsKey)
			if certErr != nil {
				logger.Error("tls", "TLS server failed to load cert", map[string]any{"error": certErr.Error()})
				logger.Warn("tls", "falling back to plain HTTP — fix your certificate and restart", map[string]any{"addr": cfg.Addr})
				if httpErr := func() error { s := &http.Server{Addr: cfg.Addr, Handler: limiter.Middleware(handler), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 120 * time.Second}; return s.ListenAndServe() }(); httpErr != nil {
					logger.Error("main", "HTTP server failed", map[string]any{"error": httpErr.Error()})
					os.Exit(1)
				}
				return
			}

			httpsListener, listenErr := tls.Listen("tcp", cfg.TLSAddr, tlsCfg)
			if listenErr != nil {
				logger.Error("tls", "TLS server failed to bind", map[string]any{"addr": cfg.TLSAddr, "error": listenErr.Error()})
				logger.Warn("tls", "falling back to plain HTTP — fix your certificate and restart", map[string]any{"addr": cfg.Addr})
				if httpErr := func() error { s := &http.Server{Addr: cfg.Addr, Handler: limiter.Middleware(handler), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 120 * time.Second}; return s.ListenAndServe() }(); httpErr != nil {
					logger.Error("main", "HTTP server failed", map[string]any{"error": httpErr.Error()})
					os.Exit(1)
				}
				return
			}

			logger.Info("tls", "HTTPS server running", map[string]any{"addr": cfg.TLSAddr})
			if err := func() error { s := &http.Server{Handler: limiter.Middleware(handler), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 120 * time.Second}; return s.Serve(httpsListener) }(); err != nil {
				logger.Error("tls", "HTTPS server failed", map[string]any{"error": err.Error()})
				os.Exit(1)
			}
		} else {
			if safepath.Exists(tlsCert) && safepath.Exists(tlsKey) && behindProxy {
				logger.Info("tls", "TLS available but behind reverse proxy", map[string]any{"addr": cfg.Addr})
				logger.Info("tls", "set PANEL_TLS_DIRECT=true to serve TLS directly from Go")
			} else if !safepath.Exists(tlsCert) || !safepath.Exists(tlsKey) {
				logger.Info("tls", "TLS disabled: cert/key not found", map[string]any{"cert": tlsCert, "key": tlsKey})
			}
			if httpErr := func() error { s := &http.Server{Addr: cfg.Addr, Handler: limiter.Middleware(handler), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 120 * time.Second}; return s.ListenAndServe() }(); httpErr != nil {
				logger.Error("main", "HTTP server failed", map[string]any{"error": httpErr.Error()})
				os.Exit(1)
			}
		}
	}
}
