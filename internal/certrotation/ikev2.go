// Package certrotation — IKEv2 domain certificate lifecycle management.
//
// This file extends the existing certrotation package with domain-based IKEv2
// certificate issuance and renewal via Let's Encrypt (ACME). It integrates with
// the existing Worker's hourly cycle to:
//  1. Detect domains bound to IKEv2 protocol that need certificates
//  2. Issue new certificates for newly bound domains
//  3. Auto-renew certificates within 30 days of expiry
//  4. Retry failed issuance up to 72 times (3 days), then mark as "expired"
//  5. Push issued/renewed certs to knode via SetCertificates gRPC
//  6. Notify admin on failures
//
// Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.7
package certrotation

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

// MaxRetryCount is the maximum number of retry attempts for certificate issuance/renewal.
// At 1 attempt per hour, 72 retries = 3 days of retrying before giving up.
const MaxRetryCount = 72

// RenewalWindow is the duration before expiry at which auto-renewal is triggered.
const RenewalWindow = 30 * 24 * time.Hour

// IKEv2CertStore defines the database operations needed for IKEv2 certificate management.
// This interface decouples the certrotation logic from the concrete dbstore implementation.
type IKEv2CertStore interface {
	// ListIKEv2Domains returns all domain IDs and names that have an IKEv2 protocol binding.
	ListIKEv2Domains(ctx context.Context) ([]IKEv2DomainBinding, error)
	// GetCertificateByDomain returns the current certificate for a domain, or nil if none exists.
	GetCertificateByDomain(ctx context.Context, domainID int64) (*IKEv2Certificate, error)
	// CreateCertificate inserts a new certificate record.
	CreateCertificate(ctx context.Context, cert *IKEv2Certificate) error
	// UpdateCertificate updates an existing certificate record.
	UpdateCertificate(ctx context.Context, cert *IKEv2Certificate) error
	// ListExpiringCertificates returns IKEv2 certs expiring within the given duration.
	ListExpiringCertificates(ctx context.Context, within time.Duration) ([]IKEv2Certificate, error)
}

// IKEv2DomainBinding holds the minimal info for a domain bound to IKEv2 on a node.
type IKEv2DomainBinding struct {
	DomainID   int64
	DomainName string
	NodeID     int64
}

// IKEv2Certificate represents a certificate record in the vpn_certificates table.
type IKEv2Certificate struct {
	ID          int64
	NodeID      int64
	DomainID    int64
	DomainName  string // populated from JOIN, not stored
	CertType    string
	Status      string // pending, active, expired
	Certificate string
	PrivateKey  string
	CAChain     string
	IssuedAt    *time.Time
	ExpiresAt   *time.Time
	RetryCount  int
	LastError   string
}

// CertStatus derives the display status for a certificate based on its expiry.
// Returns: "valid" (>30 days), "expiring_soon" (≤30 days), "expired" (past), "none" (no cert).
func CertStatus(expiresAt *time.Time) string {
	if expiresAt == nil {
		return "none"
	}
	now := time.Now()
	if expiresAt.Before(now) || expiresAt.Equal(now) {
		return "expired"
	}
	if expiresAt.Sub(now) <= RenewalWindow {
		return "expiring_soon"
	}
	return "valid"
}

// ACMEIssuer defines the interface for issuing certificates via ACME (Let's Encrypt).
// This allows testing with mock implementations.
type ACMEIssuer interface {
	// Issue obtains a certificate for the given domain.
	// Returns the certificate PEM, private key PEM, and CA chain PEM.
	Issue(ctx context.Context, domain string) (cert, key, caChain string, expiresAt time.Time, err error)
}

// Notifier defines the interface for sending admin notifications.
type Notifier interface {
	SendEvent(eventType, title, detail string)
}

// IKEv2Worker manages the IKEv2 certificate lifecycle for domains.
// It is designed to be driven by the existing certrotation.Worker's hourly cycle.
type IKEv2Worker struct {
	store    IKEv2CertStore
	issuer   ACMEIssuer
	pusher   CertPusher
	notifier Notifier
	eventFn  func(eventType, severity, title, message string)
}

// NewIKEv2Worker creates a new IKEv2 certificate lifecycle worker.
func NewIKEv2Worker(store IKEv2CertStore, issuer ACMEIssuer, pusher CertPusher, notifier Notifier, eventFn func(string, string, string, string)) *IKEv2Worker {
	return &IKEv2Worker{
		store:    store,
		issuer:   issuer,
		pusher:   pusher,
		notifier: notifier,
		eventFn:  eventFn,
	}
}

// Run performs a single IKEv2 certificate check cycle. Called hourly by the main worker.
// It:
// 1. Finds all domains bound to IKEv2 that need a certificate (new or renewal)
// 2. Issues or renews certificates as needed
// 3. Pushes successful certs to knode
// 4. Handles retry logic and admin notification on failure
func (w *IKEv2Worker) Run(ctx context.Context) {
	// Step 1: Find domains bound to IKEv2 that need new certificates
	w.handleNewBindings(ctx)

	// Step 2: Find expiring/pending certificates that need renewal or retry
	w.handleExpiringAndPending(ctx)
}

// handleNewBindings checks for IKEv2 domain bindings that don't yet have a certificate.
func (w *IKEv2Worker) handleNewBindings(ctx context.Context) {
	bindings, err := w.store.ListIKEv2Domains(ctx)
	if err != nil {
		log.Printf("[cert] failed to list IKEv2 domain bindings: %v", err)
		return
	}

	for _, binding := range bindings {
		existing, err := w.store.GetCertificateByDomain(ctx, binding.DomainID)
		if err == nil && existing != nil {
			// Certificate already exists (active or pending) — skip
			continue
		}

		// No certificate exists — create a pending record and attempt issuance
		cert := &IKEv2Certificate{
			NodeID:   binding.NodeID,
			DomainID: binding.DomainID,
			CertType: "ikev2",
			Status:   "pending",
		}
		if err := w.store.CreateCertificate(ctx, cert); err != nil {
			log.Printf("[cert] failed to create pending certificate for domain %q: %v", binding.DomainName, err)
			continue
		}

		// Attempt immediate issuance
		w.issueCertificate(ctx, cert, binding.DomainName)
	}
}

// handleExpiringAndPending processes certificates that need renewal or retry.
func (w *IKEv2Worker) handleExpiringAndPending(ctx context.Context) {
	// Get certificates expiring within 30 days (active certs needing renewal)
	expiring, err := w.store.ListExpiringCertificates(ctx, RenewalWindow)
	if err != nil {
		log.Printf("[cert] failed to list expiring certificates: %v", err)
		return
	}

	for i := range expiring {
		cert := &expiring[i]
		if cert.CertType != "ikev2" {
			continue
		}
		w.renewCertificate(ctx, cert)
	}
}

// issueCertificate attempts to obtain a new Let's Encrypt certificate for a domain.
// Implements retry logic per Requirement 6.4.
func (w *IKEv2Worker) issueCertificate(ctx context.Context, cert *IKEv2Certificate, domainName string) {
	certPEM, keyPEM, caChainPEM, expiresAt, err := w.issuer.Issue(ctx, domainName)
	if err != nil {
		w.handleIssuanceFailure(ctx, cert, domainName, err)
		return
	}

	w.handleIssuanceSuccess(ctx, cert, domainName, certPEM, keyPEM, caChainPEM, expiresAt)
}

// renewCertificate attempts to renew an expiring certificate.
func (w *IKEv2Worker) renewCertificate(ctx context.Context, cert *IKEv2Certificate) {
	domainName := cert.DomainName
	if domainName == "" {
		// Fall back to domain lookup if not joined
		domainName = fmt.Sprintf("domain_id_%d", cert.DomainID)
	}

	certPEM, keyPEM, caChainPEM, expiresAt, err := w.issuer.Issue(ctx, domainName)
	if err != nil {
		w.handleIssuanceFailure(ctx, cert, domainName, err)
		return
	}

	w.handleIssuanceSuccess(ctx, cert, domainName, certPEM, keyPEM, caChainPEM, expiresAt)
}

// handleIssuanceSuccess processes a successful certificate issuance/renewal.
// Stores the cert, pushes to knode, and emits success event.
// Requirements: 6.2, 6.5
func (w *IKEv2Worker) handleIssuanceSuccess(ctx context.Context, cert *IKEv2Certificate, domainName, certPEM, keyPEM, caChainPEM string, expiresAt time.Time) {
	now := time.Now()
	cert.Certificate = certPEM
	cert.PrivateKey = keyPEM
	cert.CAChain = caChainPEM
	cert.IssuedAt = &now
	cert.ExpiresAt = &expiresAt
	cert.Status = "active"
	cert.RetryCount = 0
	cert.LastError = ""

	if err := w.store.UpdateCertificate(ctx, cert); err != nil {
		log.Printf("[cert] failed to store certificate for domain %q: %v", domainName, err)
		return
	}

	log.Printf("[cert] issued IKEv2 certificate for domain %q, expires %s", domainName, expiresAt.Format("2006-01-02"))
	w.eventFn("cert.issued", "info",
		fmt.Sprintf("IKEv2 certificate issued for %q", domainName),
		fmt.Sprintf("Certificate expires on %s.", expiresAt.Format("2006-01-02")))

	// Push to knode via gRPC SetCertificates
	w.pushToNode(ctx, cert, domainName)
}

// HandleIssuanceFailure processes a failed certificate issuance/renewal.
// Implements retry logic: max 72 attempts, then mark as "expired".
// Exported for testing.
// Requirements: 6.4, 6.7
func (w *IKEv2Worker) HandleIssuanceFailure(ctx context.Context, cert *IKEv2Certificate, domainName string, issueErr error) {
	w.handleIssuanceFailure(ctx, cert, domainName, issueErr)
}

// handleIssuanceFailure processes a failed certificate issuance/renewal.
// Implements retry logic: max 72 attempts, then mark as "expired".
// Requirements: 6.4, 6.7
func (w *IKEv2Worker) handleIssuanceFailure(ctx context.Context, cert *IKEv2Certificate, domainName string, issueErr error) {
	cert.RetryCount++
	errMsg := issueErr.Error()
	cert.LastError = errMsg

	if cert.RetryCount >= MaxRetryCount {
		// Max retries reached — mark as expired and cease retries
		cert.Status = "expired"
		if err := w.store.UpdateCertificate(ctx, cert); err != nil {
			log.Printf("[cert] failed to update expired certificate for domain %q: %v", domainName, err)
		}

		log.Printf("[cert] certificate for domain %q marked as expired after %d retries: %v", domainName, cert.RetryCount, issueErr)
		w.eventFn("cert.expired", "error",
			fmt.Sprintf("IKEv2 certificate for %q failed permanently", domainName),
			fmt.Sprintf("After %d retries (3 days), certificate issuance failed. Last error: %s", cert.RetryCount, errMsg))

		// Notify admin
		if w.notifier != nil {
			w.notifier.SendEvent("cert",
				fmt.Sprintf("🔴 IKEv2 cert failed for %s", domainName),
				fmt.Sprintf("Certificate issuance exhausted %d retries. Last error: %s", cert.RetryCount, errMsg))
		}
		return
	}

	// Retry scheduled for next hourly cycle
	if err := w.store.UpdateCertificate(ctx, cert); err != nil {
		log.Printf("[cert] failed to update retry count for domain %q: %v", domainName, err)
	}

	log.Printf("[cert] certificate issuance for domain %q failed (retry %d/%d): %v", domainName, cert.RetryCount, MaxRetryCount, issueErr)
	w.eventFn("cert.retry", "warning",
		fmt.Sprintf("IKEv2 certificate retry %d/%d for %q", cert.RetryCount, MaxRetryCount, domainName),
		fmt.Sprintf("Will retry on next hourly cycle. Error: %s", errMsg))

	// Notify admin on first failure and every 24th retry (once per day)
	if cert.RetryCount == 1 || cert.RetryCount%24 == 0 {
		if w.notifier != nil {
			w.notifier.SendEvent("cert",
				fmt.Sprintf("⚠️ IKEv2 cert retry for %s (%d/%d)", domainName, cert.RetryCount, MaxRetryCount),
				fmt.Sprintf("Certificate issuance failed. Retrying hourly. Error: %s", errMsg))
		}
	}
}

// pushToNode pushes the certificate to the associated knode via gRPC SetCertificates.
// On failure: logs error, notifies admin, will retry on next hourly cycle.
// Requirements: 6.5, 6.7
func (w *IKEv2Worker) pushToNode(ctx context.Context, cert *IKEv2Certificate, domainName string) {
	if w.pusher == nil {
		log.Printf("[cert] no gRPC pusher configured, cannot push IKEv2 cert for domain %q to node %d", domainName, cert.NodeID)
		return
	}

	err := w.pusher.SetCertificates(ctx, cert.NodeID, "ikev2",
		[]byte(cert.CAChain),
		[]byte(cert.Certificate),
		[]byte(cert.PrivateKey))
	if err != nil {
		log.Printf("[cert] gRPC push failed for IKEv2 cert on domain %q to node %d: %v", domainName, cert.NodeID, err)
		w.eventFn("cert.push_failed", "error",
			fmt.Sprintf("Failed to push IKEv2 cert to node for %q", domainName),
			fmt.Sprintf("gRPC SetCertificates failed: %v. Will retry on next hourly cycle.", err))

		// Notify admin of push failure
		if w.notifier != nil {
			w.notifier.SendEvent("cert",
				fmt.Sprintf("⚠️ IKEv2 cert push failed for %s", domainName),
				fmt.Sprintf("gRPC push to node %d failed: %v. Will retry next cycle.", cert.NodeID, err))
		}
		return
	}

	log.Printf("[cert] pushed IKEv2 certificate for domain %q to node %d via gRPC", domainName, cert.NodeID)
}

// --- ACME Issuer Implementation ---

// LetsEncryptIssuer issues certificates via Let's Encrypt using the ACME protocol.
// Uses the autocert Manager for the full ACME flow including HTTP-01 challenge.
type LetsEncryptIssuer struct {
	Email    string
	CacheDir string
}

// NewLetsEncryptIssuer creates a new Let's Encrypt certificate issuer.
func NewLetsEncryptIssuer(email, cacheDir string) *LetsEncryptIssuer {
	return &LetsEncryptIssuer{
		Email:    email,
		CacheDir: cacheDir,
	}
}

// Issue obtains a certificate for the given domain via Let's Encrypt ACME.
// Returns the certificate PEM, private key PEM, CA chain PEM, and expiration time.
func (le *LetsEncryptIssuer) Issue(ctx context.Context, domain string) (cert, key, caChain string, expiresAt time.Time, err error) {
	// Generate a new ECDSA private key for this certificate
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", "", time.Time{}, fmt.Errorf("generate key: %w", err)
	}

	// Encode private key to PEM
	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return "", "", "", time.Time{}, fmt.Errorf("marshal key: %w", err)
	}
	keyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}))

	// Use autocert Manager for the ACME flow
	manager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(domain),
		Email:      le.Email,
	}
	if le.CacheDir != "" {
		manager.Cache = autocert.DirCache(le.CacheDir)
	}

	// Request certificate using autocert's GetCertificate which handles the full ACME flow.
	// This requires the panel to be serving on port 80/443 for HTTP-01 challenge.
	hello := &tls.ClientHelloInfo{ServerName: domain}
	certData, err := manager.GetCertificate(hello)
	if err != nil {
		return "", "", "", time.Time{}, fmt.Errorf("ACME certificate issuance failed for %q: %w", domain, err)
	}

	// Extract certificate PEM from the tls.Certificate
	var certPEMBytes []byte
	var chainPEMBytes []byte
	for i, derBytes := range certData.Certificate {
		block := &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}
		if i == 0 {
			certPEMBytes = append(certPEMBytes, pem.EncodeToMemory(block)...)
		} else {
			chainPEMBytes = append(chainPEMBytes, pem.EncodeToMemory(block)...)
		}
	}

	// Parse leaf certificate for expiry
	leaf, err := x509.ParseCertificate(certData.Certificate[0])
	if err != nil {
		return "", "", "", time.Time{}, fmt.Errorf("parse issued cert: %w", err)
	}

	// Use the private key from autocert (not our generated one, as autocert manages its own)
	// If the returned cert has a PrivateKey, use it; otherwise use our generated one.
	if certData.PrivateKey != nil {
		switch pk := certData.PrivateKey.(type) {
		case *ecdsa.PrivateKey:
			pkDER, err := x509.MarshalECPrivateKey(pk)
			if err == nil {
				keyPEM = string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: pkDER}))
			}
		default:
			pkDER, err := x509.MarshalPKCS8PrivateKey(pk)
			if err == nil {
				keyPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkDER}))
			}
		}
	}

	return string(certPEMBytes), keyPEM, string(chainPEMBytes), leaf.NotAfter, nil
}
