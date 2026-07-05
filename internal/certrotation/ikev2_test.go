package certrotation

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// --- Mock implementations for testing ---

// mockIKEv2Store is a test double for IKEv2CertStore.
type mockIKEv2Store struct {
	bindings     []IKEv2DomainBinding
	certificates map[int64]*IKEv2Certificate // keyed by domain_id
	allCerts     []IKEv2Certificate
	createErr    error
	updateErr    error
	updated      []*IKEv2Certificate
}

func newMockStore() *mockIKEv2Store {
	return &mockIKEv2Store{
		certificates: make(map[int64]*IKEv2Certificate),
	}
}

func (m *mockIKEv2Store) ListIKEv2Domains(_ context.Context) ([]IKEv2DomainBinding, error) {
	return m.bindings, nil
}

func (m *mockIKEv2Store) GetCertificateByDomain(_ context.Context, domainID int64) (*IKEv2Certificate, error) {
	c, ok := m.certificates[domainID]
	if !ok {
		return nil, nil
	}
	return c, nil
}

func (m *mockIKEv2Store) CreateCertificate(_ context.Context, cert *IKEv2Certificate) error {
	if m.createErr != nil {
		return m.createErr
	}
	cert.ID = int64(len(m.certificates) + 1)
	m.certificates[cert.DomainID] = cert
	return nil
}

func (m *mockIKEv2Store) UpdateCertificate(_ context.Context, cert *IKEv2Certificate) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updated = append(m.updated, cert)
	m.certificates[cert.DomainID] = cert
	return nil
}

func (m *mockIKEv2Store) ListExpiringCertificates(_ context.Context, _ time.Duration) ([]IKEv2Certificate, error) {
	return m.allCerts, nil
}

// mockACMEIssuer is a test double for ACMEIssuer.
type mockACMEIssuer struct {
	shouldFail bool
	failErr    error
	issuedFor  []string
}

func (m *mockACMEIssuer) Issue(_ context.Context, domain string) (string, string, string, time.Time, error) {
	m.issuedFor = append(m.issuedFor, domain)
	if m.shouldFail {
		if m.failErr != nil {
			return "", "", "", time.Time{}, m.failErr
		}
		return "", "", "", time.Time{}, fmt.Errorf("ACME issuance failed for %s", domain)
	}
	expiry := time.Now().Add(90 * 24 * time.Hour)
	return "---CERT PEM---", "---KEY PEM---", "---CA CHAIN---", expiry, nil
}

// mockCertPusher is a test double for CertPusher.
type mockCertPusher struct {
	shouldFail bool
	pushed     []pushRecord
}

type pushRecord struct {
	NodeID   int64
	CoreType string
}

func (m *mockCertPusher) SetCertificates(_ context.Context, nodeID int64, coreType string, _, _, _ []byte) error {
	m.pushed = append(m.pushed, pushRecord{NodeID: nodeID, CoreType: coreType})
	if m.shouldFail {
		return fmt.Errorf("gRPC push failed")
	}
	return nil
}

// mockNotifier is a test double for Notifier.
type mockNotifier struct {
	events []notifyEvent
}

type notifyEvent struct {
	EventType string
	Title     string
	Detail    string
}

func (m *mockNotifier) SendEvent(eventType, title, detail string) {
	m.events = append(m.events, notifyEvent{eventType, title, detail})
}

// noopEventFn is a no-op event function for testing.
func noopEventFn(_, _, _, _ string) {}

// --- Unit Tests ---

func TestIKEv2Worker_NewBindingIssuesCertificate(t *testing.T) {
	store := newMockStore()
	store.bindings = []IKEv2DomainBinding{
		{DomainID: 1, DomainName: "vpn.example.com", NodeID: 10},
	}

	issuer := &mockACMEIssuer{}
	pusher := &mockCertPusher{}
	notifier := &mockNotifier{}

	worker := NewIKEv2Worker(store, issuer, pusher, notifier, noopEventFn)
	worker.Run(context.Background())

	// Should have attempted issuance
	if len(issuer.issuedFor) != 1 || issuer.issuedFor[0] != "vpn.example.com" {
		t.Errorf("expected issuance for vpn.example.com, got %v", issuer.issuedFor)
	}

	// Should have pushed to node
	if len(pusher.pushed) != 1 || pusher.pushed[0].NodeID != 10 {
		t.Errorf("expected push to node 10, got %v", pusher.pushed)
	}

	// Certificate should be stored as active
	cert := store.certificates[1]
	if cert == nil {
		t.Fatal("certificate not stored")
	}
	if cert.Status != "active" {
		t.Errorf("expected status 'active', got %q", cert.Status)
	}
}

func TestIKEv2Worker_ExistingCertSkipped(t *testing.T) {
	store := newMockStore()
	store.bindings = []IKEv2DomainBinding{
		{DomainID: 1, DomainName: "vpn.example.com", NodeID: 10},
	}
	// Pre-existing active certificate
	store.certificates[1] = &IKEv2Certificate{
		ID:       1,
		DomainID: 1,
		Status:   "active",
	}

	issuer := &mockACMEIssuer{}
	worker := NewIKEv2Worker(store, issuer, nil, nil, noopEventFn)
	worker.Run(context.Background())

	// Should not attempt issuance
	if len(issuer.issuedFor) != 0 {
		t.Errorf("should not issue for existing cert, got %v", issuer.issuedFor)
	}
}

func TestIKEv2Worker_RetryIncrements(t *testing.T) {
	store := newMockStore()
	store.bindings = []IKEv2DomainBinding{
		{DomainID: 1, DomainName: "vpn.example.com", NodeID: 10},
	}

	issuer := &mockACMEIssuer{shouldFail: true}
	notifier := &mockNotifier{}
	worker := NewIKEv2Worker(store, issuer, nil, notifier, noopEventFn)
	worker.Run(context.Background())

	// Certificate should exist with retry_count = 1
	cert := store.certificates[1]
	if cert == nil {
		t.Fatal("certificate not stored")
	}
	if cert.RetryCount != 1 {
		t.Errorf("expected retry_count = 1, got %d", cert.RetryCount)
	}
	if cert.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", cert.Status)
	}

	// Admin should be notified on first failure
	if len(notifier.events) == 0 {
		t.Error("expected admin notification on first failure")
	}
}

func TestIKEv2Worker_MaxRetriesMarksExpired(t *testing.T) {
	store := newMockStore()
	notifier := &mockNotifier{}
	issuer := &mockACMEIssuer{shouldFail: true}

	worker := NewIKEv2Worker(store, issuer, nil, notifier, noopEventFn)

	// Simulate a certificate at retry 71 (one more failure will hit 72)
	cert := &IKEv2Certificate{
		ID:         1,
		NodeID:     10,
		DomainID:   1,
		CertType:   "ikev2",
		Status:     "pending",
		RetryCount: 71,
	}

	worker.HandleIssuanceFailure(context.Background(), cert, "vpn.example.com", fmt.Errorf("rate limited"))

	if cert.Status != "expired" {
		t.Errorf("expected status 'expired' after 72 retries, got %q", cert.Status)
	}
	if cert.RetryCount != 72 {
		t.Errorf("expected retry_count = 72, got %d", cert.RetryCount)
	}
}

func TestIKEv2Worker_BelowMaxRetryDoesNotExpire(t *testing.T) {
	store := newMockStore()
	notifier := &mockNotifier{}
	issuer := &mockACMEIssuer{shouldFail: true}

	worker := NewIKEv2Worker(store, issuer, nil, notifier, noopEventFn)

	cert := &IKEv2Certificate{
		ID:         1,
		NodeID:     10,
		DomainID:   1,
		CertType:   "ikev2",
		Status:     "pending",
		RetryCount: 50,
	}

	worker.HandleIssuanceFailure(context.Background(), cert, "vpn.example.com", fmt.Errorf("timeout"))

	if cert.Status == "expired" {
		t.Error("should not mark as expired when retry_count < 72")
	}
	if cert.RetryCount != 51 {
		t.Errorf("expected retry_count = 51, got %d", cert.RetryCount)
	}
}

func TestIKEv2Worker_PushFailureNotifiesAdmin(t *testing.T) {
	store := newMockStore()
	store.bindings = []IKEv2DomainBinding{
		{DomainID: 1, DomainName: "vpn.example.com", NodeID: 10},
	}

	issuer := &mockACMEIssuer{}
	pusher := &mockCertPusher{shouldFail: true}
	notifier := &mockNotifier{}

	worker := NewIKEv2Worker(store, issuer, pusher, notifier, noopEventFn)
	worker.Run(context.Background())

	// Certificate should still be stored as active
	cert := store.certificates[1]
	if cert == nil {
		t.Fatal("certificate not stored")
	}
	if cert.Status != "active" {
		t.Errorf("expected status 'active' even when push fails, got %q", cert.Status)
	}

	// Admin should be notified of push failure
	found := false
	for _, e := range notifier.events {
		if e.EventType == "cert" && contains(e.Title, "push failed") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected admin notification about push failure")
	}
}

func TestIKEv2Worker_RenewalForExpiringCerts(t *testing.T) {
	store := newMockStore()
	// No new bindings
	store.bindings = nil

	// Expiring certificate
	expiresIn25Days := time.Now().Add(25 * 24 * time.Hour)
	store.allCerts = []IKEv2Certificate{
		{
			ID:         1,
			NodeID:     10,
			DomainID:   1,
			DomainName: "vpn.example.com",
			CertType:   "ikev2",
			Status:     "active",
			ExpiresAt:  &expiresIn25Days,
		},
	}

	issuer := &mockACMEIssuer{}
	pusher := &mockCertPusher{}
	worker := NewIKEv2Worker(store, issuer, pusher, nil, noopEventFn)
	worker.Run(context.Background())

	// Should have attempted renewal
	if len(issuer.issuedFor) != 1 || issuer.issuedFor[0] != "vpn.example.com" {
		t.Errorf("expected renewal for vpn.example.com, got %v", issuer.issuedFor)
	}
}

func TestCertStatus_Derivation(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt *time.Time
		want      string
	}{
		{
			name:      "nil returns none",
			expiresAt: nil,
			want:      "none",
		},
		{
			name:      "past returns expired",
			expiresAt: timePtr(time.Now().Add(-24 * time.Hour)),
			want:      "expired",
		},
		{
			name:      "within 30 days returns expiring_soon",
			expiresAt: timePtr(time.Now().Add(15 * 24 * time.Hour)),
			want:      "expiring_soon",
		},
		{
			name:      "more than 30 days returns valid",
			expiresAt: timePtr(time.Now().Add(60 * 24 * time.Hour)),
			want:      "valid",
		},
		{
			name:      "exactly now returns expired",
			expiresAt: timePtr(time.Now()),
			want:      "expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CertStatus(tt.expiresAt)
			if got != tt.want {
				t.Errorf("CertStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- Helpers ---

func timePtr(t time.Time) *time.Time {
	return &t
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
