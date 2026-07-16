package postgres

import (
	"context"
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/anonysec/koris/internal/dbstore"
)

const testDSN = "postgres://test:test@localhost:5432/koris_test?sslmode=disable"

// Helper functions for pointer types
func stringPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func newTestPostgresStore(t *testing.T) *Store {
	t.Helper()
	ctx := context.Background()
	store, err := New(ctx, testDSN)
	if err != nil {
		t.Fatalf("failed to create postgres store: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	// Truncate all test tables to ensure clean state for each test.
// Note: schema_migrations is NOT truncated to allow migration idempotency
// testing within the same test.
	_, err = store.Pool().Exec(ctx, `
		TRUNCATE TABLE 
			panel_sessions,
			node_metrics_history,
			user_traffic_log,
			customers,
			vpn_certificates
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("failed to truncate test tables: %v", err)
	}

	return store
}

func TestMain(m *testing.M) {
	// Ensure the test database schema exists. The unit tests truncate tables
	// (panel_sessions, etc.) but never create them, so apply the real
	// migrations once against koris_test before running the suite.
	ctx := context.Background()
	store, err := New(ctx, testDSN)
	if err != nil {
		log.Fatalf("TestMain: connect: %v", err)
	}
	defer store.Close()

	dir := os.Getenv("KORIS_TEST_MIGRATIONS")
	if dir == "" {
		// repo root migrations/ relative to this package (internal/dbstore/postgres)
		dir = filepath.Join("..", "..", "..", "migrations")
	}
	if err := store.Migrate(ctx, dir); err != nil {
		log.Fatalf("TestMain: migrate %s: %v", dir, err)
	}

	os.Exit(m.Run())
}

func TestNew(t *testing.T) {
	store, err := New(context.Background(), testDSN)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer store.Close()

	if store.DB() == nil {
		t.Fatal("expected non-nil DB")
	}
}

func TestPing(t *testing.T) {
	store := newTestPostgresStore(t)
	if err := store.Ping(context.Background()); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}

func TestAcquireReleaseLock(t *testing.T) {
	store := newTestPostgresStore(t)
	ctx := context.Background()

	// Use unique lock IDs for this test to avoid collisions with other tests.
	lock1 := int64(100000 + time.Now().UnixNano()%10000)
	lock2 := lock1 + 1

	// Acquire a dedicated connection from the pool to ensure all lock operations
	// use the same session (advisory locks are session-scoped in PostgreSQL).
	conn, err := store.pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire connection: %v", err)
	}
	defer conn.Release()

	// First acquire should succeed.
	var acquired bool
	err = conn.Conn().QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", lock1).Scan(&acquired)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !acquired {
		t.Fatal("expected lock to be acquired")
	}

	// Second acquire of same lock should SUCCEED (advisory locks are reentrant in same session).
	err = conn.Conn().QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", lock1).Scan(&acquired)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !acquired {
		t.Fatal("expected lock re-acquisition to succeed (reentrant)")
	}

	// Different lock ID should succeed.
	err = conn.Conn().QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", lock2).Scan(&acquired)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !acquired {
		t.Fatal("expected different lock to be acquired")
	}

	// Release the first lock ONCE.
	_, err = conn.Conn().Exec(ctx, "SELECT pg_advisory_unlock($1)", lock1)
	if err != nil {
		t.Fatalf("release failed: %v", err)
	}

	// The lock is still held (reentrant count > 0). Release again to fully release.
	_, err = conn.Conn().Exec(ctx, "SELECT pg_advisory_unlock($1)", lock1)
	if err != nil {
		t.Fatalf("second release failed: %v", err)
	}

	// Now we can acquire it again.
	err = conn.Conn().QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", lock1).Scan(&acquired)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !acquired {
		t.Fatal("expected lock to be re-acquired after full release")
	}
}

func TestReleaseLock_NotHeld(t *testing.T) {
	store := newTestPostgresStore(t)
	ctx := context.Background()

	err := store.ReleaseLock(ctx, 9999)
	if err != dbstore.ErrLockNotAcquired {
		t.Fatalf("expected ErrLockNotAcquired, got %v", err)
	}
}

func TestSessionCRUD(t *testing.T) {
	store := newTestPostgresStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	sess := &dbstore.Session{
		Token:      "test-token-123",
		AdminID:    sql.NullInt64{Int64: 1, Valid: true},
		CustomerID: sql.NullInt64{},
		Data:       []byte(`{"role":"admin"}`),
		IPAddress:  "192.168.1.1",
		UserAgent:  "TestAgent/1.0",
		CreatedAt:  now,
		ExpiresAt:  now.Add(24 * time.Hour),
		LastSeen:   now,
	}

	// Save session.
	if err := store.SaveSession(ctx, sess); err != nil {
		t.Fatalf("save session failed: %v", err)
	}

	// Get session.
	got, err := store.GetSession(ctx, "test-token-123")
	if err != nil {
		t.Fatalf("get session failed: %v", err)
	}
	if got.Token != sess.Token {
		t.Errorf("token mismatch: got %q, want %q", got.Token, sess.Token)
	}
	if got.AdminID.Int64 != 1 || !got.AdminID.Valid {
		t.Errorf("admin_id mismatch: got %v", got.AdminID)
	}
	if got.IPAddress != "192.168.1.1" {
		t.Errorf("ip mismatch: got %q", got.IPAddress)
	}

	// Delete session.
	if err := store.DeleteSession(ctx, "test-token-123"); err != nil {
		t.Fatalf("delete session failed: %v", err)
	}

	// Get should fail.
	_, err = store.GetSession(ctx, "test-token-123")
	if err != dbstore.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetSession_Expired(t *testing.T) {
	store := newTestPostgresStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	sess := &dbstore.Session{
		Token:     "expired-token",
		CreatedAt: now.Add(-2 * time.Hour),
		ExpiresAt: now.Add(-1 * time.Hour), // already expired
		LastSeen:  now.Add(-2 * time.Hour),
	}

	if err := store.SaveSession(ctx, sess); err != nil {
		t.Fatalf("save session failed: %v", err)
	}

	_, err := store.GetSession(ctx, "expired-token")
	if err != dbstore.ErrSessionExpired {
		t.Fatalf("expected ErrSessionExpired, got %v", err)
	}
}

func TestGetSession_NotFound(t *testing.T) {
	store := newTestPostgresStore(t)
	ctx := context.Background()

	_, err := store.GetSession(ctx, "nonexistent")
	if err != dbstore.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCleanExpiredSessions(t *testing.T) {
	store := newTestPostgresStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)

	// Insert an expired session and a valid session.
	expired := &dbstore.Session{
		Token:     "expired",
		CreatedAt: now.Add(-2 * time.Hour),
		ExpiresAt: now.Add(-1 * time.Hour),
		LastSeen:  now.Add(-2 * time.Hour),
	}
	valid := &dbstore.Session{
		Token:     "valid",
		CreatedAt: now,
		ExpiresAt: now.Add(1 * time.Hour),
		LastSeen:  now,
	}

	store.SaveSession(ctx, expired)
	store.SaveSession(ctx, valid)

	if err := store.CleanExpiredSessions(ctx); err != nil {
		t.Fatalf("clean failed: %v", err)
	}

	// Valid session should still exist.
	_, err := store.GetSession(ctx, "valid")
	if err != nil {
		t.Fatalf("valid session should still exist: %v", err)
	}

	// Expired session should be gone.
	_, err = store.GetSession(ctx, "expired")
	if err != dbstore.ErrNotFound {
		t.Fatalf("expected expired session to be cleaned, got %v", err)
	}
}

func TestInsertAndQueryMetrics(t *testing.T) {
	store := newTestPostgresStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	m := &dbstore.MetricsRow{
		Time:           now,
		CPUPercent:     65.5,
		RAMPercent:     72.3,
		DiskPercent:    45.0,
		RxBPS:          1000000,
		TxBPS:          500000,
		ActiveSessions: 42,
		UptimeSeconds:  86400,
	}

	if err := store.InsertMetrics(ctx, 1, m); err != nil {
		t.Fatalf("insert metrics failed: %v", err)
	}

	results, err := store.QueryMetrics(ctx, 1, now.Add(-1*time.Minute), now.Add(1*time.Minute))
	if err != nil {
		t.Fatalf("query metrics failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 row, got %d", len(results))
	}

	got := results[0]
	if got.CPUPercent != 65.5 {
		t.Errorf("cpu mismatch: got %v", got.CPUPercent)
	}
	// Float comparison with tolerance
	if got.RAMPercent < 72.299 || got.RAMPercent > 72.301 {
		t.Errorf("ram mismatch: got %v", got.RAMPercent)
	}
	if got.ActiveSessions != 42 {
		t.Errorf("sessions mismatch: got %v", got.ActiveSessions)
	}
	if got.UptimeSeconds != 86400 {
		t.Errorf("uptime mismatch: got %v", got.UptimeSeconds)
	}
}

func TestInsertTrafficLog(t *testing.T) {
	store := newTestPostgresStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	entry := &dbstore.TrafficLogEntry{
		Time:    now,
		UserID:  10,
		NodeID:  1,
		RxBytes: 1048576,
		TxBytes: 524288,
	}

	if err := store.InsertTrafficLog(ctx, entry); err != nil {
		t.Fatalf("insert traffic log failed: %v", err)
	}

	// Verify via raw query.
	var rxBytes, txBytes int64
	err := store.pool.QueryRow(ctx, "SELECT rx_bytes, tx_bytes FROM user_traffic_log WHERE user_id = $1", 10).Scan(&rxBytes, &txBytes)
	if err != nil {
		t.Fatalf("query traffic log failed: %v", err)
	}
	if rxBytes != 1048576 || txBytes != 524288 {
		t.Errorf("traffic mismatch: rx=%d tx=%d", rxBytes, txBytes)
	}
}

func TestTransaction(t *testing.T) {
	store := newTestPostgresStore(t)
	ctx := context.Background()

	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatalf("begin failed: %v", err)
	}

	_, err = tx.Exec(ctx, `INSERT INTO user_traffic_log (time, user_id, node_id, rx_bytes, tx_bytes) VALUES ($1, $2, $3, $4, $5)`,
		time.Now().Format(time.RFC3339), 1, 1, 100, 200)
	if err != nil {
		t.Fatalf("exec in tx failed: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	// Verify committed.
	var count int
	store.pool.QueryRow(ctx, "SELECT COUNT(*) FROM user_traffic_log").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 row, got %d", count)
	}
}

func TestTransaction_Rollback(t *testing.T) {
	store := newTestPostgresStore(t)
	ctx := context.Background()

	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatalf("begin failed: %v", err)
	}

	_, err = tx.Exec(ctx, `INSERT INTO user_traffic_log (time, user_id, node_id, rx_bytes, tx_bytes) VALUES ($1, $2, $3, $4, $5)`,
		time.Now().Format(time.RFC3339), 1, 1, 100, 200)
	if err != nil {
		t.Fatalf("exec in tx failed: %v", err)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback failed: %v", err)
	}

	// Verify nothing committed.
	var count int
	store.pool.QueryRow(ctx, "SELECT COUNT(*) FROM user_traffic_log").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 rows after rollback, got %d", count)
	}
}

func TestMigrate(t *testing.T) {
	store := newTestPostgresStore(t)
	ctx := context.Background()

	// Create a temporary migrations directory.
	dir := t.TempDir()

	// Write a migration file.
	migration := `CREATE TABLE IF NOT EXISTS test_items (id BIGSERIAL PRIMARY KEY, name TEXT NOT NULL);`
	if err := os.WriteFile(filepath.Join(dir, "001_create_test.sql"), []byte(migration), 0644); err != nil {
		t.Fatalf("write migration file: %v", err)
	}

	// Run migrations.
	if err := store.Migrate(ctx, dir); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	// Table should exist now.
	_, err := store.pool.Exec(ctx, "INSERT INTO test_items (name) VALUES ('hello')")
	if err != nil {
		t.Fatalf("insert into migrated table failed: %v", err)
	}

	// Running again should be idempotent (skip already applied).
	if err := store.Migrate(ctx, dir); err != nil {
		t.Fatalf("second migrate failed: %v", err)
	}
}

func TestDeleteSession_NotFound(t *testing.T) {
	store := newTestPostgresStore(t)
	ctx := context.Background()

	err := store.DeleteSession(ctx, "nonexistent-token")
	if err != dbstore.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
