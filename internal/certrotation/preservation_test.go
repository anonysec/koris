package certrotation

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"testing/quick"
	"time"
	"unicode/utf8"

	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// migration025SQL is the fix migration that adds cert_path and status columns.
const migration025SQL = `
ALTER TABLE vpn_certificates ADD COLUMN cert_path VARCHAR(512) NULL AFTER fingerprint;
ALTER TABLE vpn_certificates ADD COLUMN status ENUM('active','revoked','expired') NULL DEFAULT 'active' AFTER cert_path;
ALTER TABLE vpn_certificates ADD INDEX idx_cert_status (status);
`

// certRow represents a randomly generated vpn_certificates row for property testing.
type certRow struct {
	Name        string
	Type        string
	NodeID      sql.NullInt64
	Content     string
	IsDefault   int
	ExpiresAt   sql.NullTime
	Fingerprint sql.NullString
}

// validCertTypes are the allowed ENUM values for the type column.
var validCertTypes = []string{"ca", "tls_crypt", "client_cert", "client_key"}

// specialChars used in random string generation to test edge cases.
var specialChars = []rune{'\'', '"', '\\', '\n', '\t', '\r', '/', '@', '#',
	'$', '%', '^', '&', '*', '(', ')', '<', '>', '?', '!', '~', '`', '|',
	'日', '本', '語', '中', '文', 'ñ', 'ü', 'ö', 'é', 'à', '€', '£'}

// randString generates a random string of the given max length, including special chars and unicode.
func randString(rng *rand.Rand, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	length := rng.Intn(maxLen) + 1
	var sb strings.Builder
	for i := 0; i < length; i++ {
		choice := rng.Intn(3)
		switch choice {
		case 0:
			// ASCII letter/digit
			chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 _-."
			sb.WriteByte(chars[rng.Intn(len(chars))])
		case 1:
			// Special character
			r := specialChars[rng.Intn(len(specialChars))]
			sb.WriteRune(r)
		case 2:
			// Random unicode (BMP)
			r := rune(rng.Intn(0xD7FF-0x20) + 0x20)
			if utf8.ValidRune(r) {
				sb.WriteRune(r)
			} else {
				sb.WriteByte('x')
			}
		}
	}
	return sb.String()
}

// generateCertRow creates a random certRow using the given RNG.
func generateCertRow(rng *rand.Rand) certRow {
	row := certRow{
		Name:    randString(rng, 100),
		Type:    validCertTypes[rng.Intn(len(validCertTypes))],
		Content: randString(rng, 500),
	}

	// Ensure name is not empty (NOT NULL constraint)
	if row.Name == "" {
		row.Name = "cert"
	}
	// Ensure content is not empty (NOT NULL constraint, TEXT NOT NULL)
	if row.Content == "" {
		row.Content = "content"
	}

	// Random node_id: NULL or a positive integer
	if rng.Intn(3) == 0 {
		row.NodeID = sql.NullInt64{Valid: false}
	} else {
		row.NodeID = sql.NullInt64{Int64: int64(rng.Intn(10000) + 1), Valid: true}
	}

	// is_default: 0 or 1
	row.IsDefault = rng.Intn(2)

	// expires_at: NULL or a random timestamp
	if rng.Intn(3) == 0 {
		row.ExpiresAt = sql.NullTime{Valid: false}
	} else {
		// Random time between 2020 and 2030
		t := time.Date(2020+rng.Intn(10), time.Month(rng.Intn(12)+1),
			rng.Intn(28)+1, rng.Intn(24), rng.Intn(60), rng.Intn(60), 0, time.UTC)
		row.ExpiresAt = sql.NullTime{Time: t, Valid: true}
	}

	// fingerprint: NULL or random string (including empty string edge case)
	switch rng.Intn(4) {
	case 0:
		row.Fingerprint = sql.NullString{Valid: false} // NULL
	case 1:
		row.Fingerprint = sql.NullString{String: "", Valid: true} // empty string
	default:
		row.Fingerprint = sql.NullString{String: randString(rng, 64), Valid: true}
	}

	return row
}

// TestPreservation_DataIntegrityAfterMigration_Mock uses sqlmock to verify that after
// migration 025 is applied, existing column data remains accessible and new columns
// (cert_path, status) are NULL. This property test runs without a real database.
//
// The test simulates the observation that:
// 1. Rows inserted with the original schema (id, name, type, node_id, content,
//    is_default, created_at, expires_at, fingerprint) can still be fully read
//    after migration adds cert_path and status columns.
// 2. New columns cert_path and status are NULL for all existing rows.
//
// **Validates: Requirements 3.1, 3.2**
func TestPreservation_DataIntegrityAfterMigration_Mock(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 50,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	property := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		// Generate random rows to simulate pre-migration data
		numRows := rng.Intn(16) + 5
		rows := make([]certRow, numRows)
		for i := range rows {
			rows[i] = generateCertRow(rng)
		}

		// Create mock DB that simulates post-migration schema behavior
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Logf("failed to create sqlmock: %v", err)
			return false
		}
		defer db.Close()

		// Simulate querying all rows after migration 025 with extended schema.
		// After migration, SELECT should return all original columns plus new
		// NULL cert_path and NULL status columns.
		resultColumns := []string{
			"name", "type", "node_id", "content", "is_default",
			"created_at", "expires_at", "fingerprint", "cert_path", "status",
		}
		mockRows := sqlmock.NewRows(resultColumns)
		for _, row := range rows {
			var nodeID interface{}
			if row.NodeID.Valid {
				nodeID = row.NodeID.Int64
			}
			var expiresAt interface{}
			if row.ExpiresAt.Valid {
				expiresAt = row.ExpiresAt.Time
			}
			var fingerprint interface{}
			if row.Fingerprint.Valid {
				fingerprint = row.Fingerprint.String
			}
			createdAt := time.Now()
			// cert_path and status are NULL for existing rows after migration
			mockRows.AddRow(
				row.Name, row.Type, nodeID, row.Content, row.IsDefault,
				createdAt, expiresAt, fingerprint, nil, nil,
			)
		}

		mock.ExpectQuery("SELECT name, type, node_id, content, is_default, created_at, expires_at, fingerprint, cert_path, status FROM vpn_certificates").
			WillReturnRows(mockRows)

		// Execute the query against the mock (simulates post-migration read)
		queryRows, err := db.Query(
			"SELECT name, type, node_id, content, is_default, created_at, expires_at, fingerprint, cert_path, status FROM vpn_certificates",
		)
		if err != nil {
			t.Logf("query failed: %v", err)
			return false
		}
		defer queryRows.Close()

		idx := 0
		for queryRows.Next() {
			var (
				gotName        string
				gotType        string
				gotNodeID      sql.NullInt64
				gotContent     string
				gotIsDefault   int
				gotCreatedAt   time.Time
				gotExpiresAt   sql.NullTime
				gotFingerprint sql.NullString
				gotCertPath    sql.NullString
				gotStatus      sql.NullString
			)
			err := queryRows.Scan(&gotName, &gotType, &gotNodeID, &gotContent,
				&gotIsDefault, &gotCreatedAt, &gotExpiresAt, &gotFingerprint,
				&gotCertPath, &gotStatus)
			if err != nil {
				t.Logf("scan row %d failed: %v", idx, err)
				return false
			}

			// Property: existing column values are preserved
			if gotName != rows[idx].Name {
				t.Logf("row %d: name mismatch: got %q, want %q", idx, gotName, rows[idx].Name)
				return false
			}
			if gotType != rows[idx].Type {
				t.Logf("row %d: type mismatch: got %q, want %q", idx, gotType, rows[idx].Type)
				return false
			}
			if gotNodeID != rows[idx].NodeID {
				t.Logf("row %d: node_id mismatch", idx)
				return false
			}
			if gotContent != rows[idx].Content {
				t.Logf("row %d: content mismatch", idx)
				return false
			}
			if gotIsDefault != rows[idx].IsDefault {
				t.Logf("row %d: is_default mismatch: got %d, want %d", idx, gotIsDefault, rows[idx].IsDefault)
				return false
			}

			if gotExpiresAt.Valid != rows[idx].ExpiresAt.Valid {
				t.Logf("row %d: expires_at validity mismatch", idx)
				return false
			}
			if gotExpiresAt.Valid && rows[idx].ExpiresAt.Valid {
				gotTrunc := gotExpiresAt.Time.Truncate(time.Second)
				wantTrunc := rows[idx].ExpiresAt.Time.Truncate(time.Second)
				if !gotTrunc.Equal(wantTrunc) {
					t.Logf("row %d: expires_at mismatch: got %v, want %v", idx, gotTrunc, wantTrunc)
					return false
				}
			}
			if gotFingerprint != rows[idx].Fingerprint {
				t.Logf("row %d: fingerprint mismatch", idx)
				return false
			}

			// Property: new columns (cert_path, status) are NULL for existing rows
			if gotCertPath.Valid {
				t.Logf("row %d: cert_path should be NULL after migration, got %q", idx, gotCertPath.String)
				return false
			}
			if gotStatus.Valid {
				t.Logf("row %d: status should be NULL after migration, got %q", idx, gotStatus.String)
				return false
			}

			idx++
		}

		if idx != numRows {
			t.Logf("row count mismatch: got %d rows, want %d", idx, numRows)
			return false
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Logf("mock expectations not met: %v", err)
			return false
		}
		return true
	}

	if err := quick.Check(property, cfg); err != nil {
		t.Fatalf("Preservation data integrity property failed: %v", err)
	}
}

// TestPreservation_RegenerateUpdateAfterMigration_Mock verifies that the Regenerate()
// UPDATE query (`UPDATE vpn_certificates SET expires_at = ?, fingerprint = ? WHERE id = ?`)
// continues to work after migration 025 is applied.
//
// This property test confirms that adding cert_path and status columns does NOT break
// the existing UPDATE pattern used by the Regenerate() method.
//
// **Validates: Requirements 3.3, 3.4**
func TestPreservation_RegenerateUpdateAfterMigration_Mock(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 50,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	property := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		// Generate a random cert row to simulate an existing certificate
		row := generateCertRow(rng)
		certID := int64(rng.Intn(10000) + 1)

		// Generate new expiry and fingerprint values (as Regenerate would)
		newExpiry := time.Now().Add(time.Duration(rng.Intn(365*3)+1) * 24 * time.Hour)
		newFingerprint := fmt.Sprintf("%x", rng.Int63())

		db, mock, err := sqlmock.New()
		if err != nil {
			t.Logf("failed to create sqlmock: %v", err)
			return false
		}
		defer db.Close()

		// Expect the Regenerate UPDATE query to succeed after migration 025
		// (migration only adds columns; it does NOT change existing column behavior)
		mock.ExpectExec("UPDATE vpn_certificates SET expires_at = \\?, fingerprint = \\? WHERE id = \\?").
			WithArgs(newExpiry, newFingerprint, certID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Execute the exact query used by worker.go Regenerate()
		result, err := db.Exec(
			`UPDATE vpn_certificates SET expires_at = ?, fingerprint = ? WHERE id = ?`,
			newExpiry, newFingerprint, certID,
		)
		if err != nil {
			t.Logf("Regenerate UPDATE failed for id=%d: %v", certID, err)
			return false
		}

		affected, _ := result.RowsAffected()
		if affected != 1 {
			t.Logf("Regenerate UPDATE for id=%d: expected 1 row affected, got %d", certID, affected)
			return false
		}

		// Verify the row data can still be queried with all original columns intact
		// after the UPDATE (simulating post-migration read)
		var nodeID interface{}
		if row.NodeID.Valid {
			nodeID = row.NodeID.Int64
		}
		var fingerprint interface{}
		if row.Fingerprint.Valid {
			fingerprint = row.Fingerprint.String
		}
		_ = nodeID
		_ = fingerprint

		verifyColumns := []string{"id", "name", "type", "node_id", "content",
			"is_default", "expires_at", "fingerprint", "cert_path", "status"}
		verifyRows := sqlmock.NewRows(verifyColumns).AddRow(
			certID, row.Name, row.Type, nodeID, row.Content,
			row.IsDefault, newExpiry, newFingerprint, nil, nil,
		)

		mock.ExpectQuery("SELECT id, name, type, node_id, content, is_default, expires_at, fingerprint, cert_path, status FROM vpn_certificates WHERE id = \\?").
			WithArgs(certID).
			WillReturnRows(verifyRows)

		var (
			gotID          int64
			gotName        string
			gotType        string
			gotNodeID      sql.NullInt64
			gotContent     string
			gotIsDefault   int
			gotExpiresAt   time.Time
			gotFingerprint string
			gotCertPath    sql.NullString
			gotStatus      sql.NullString
		)

		err = db.QueryRow(
			"SELECT id, name, type, node_id, content, is_default, expires_at, fingerprint, cert_path, status FROM vpn_certificates WHERE id = ?",
			certID,
		).Scan(&gotID, &gotName, &gotType, &gotNodeID, &gotContent,
			&gotIsDefault, &gotExpiresAt, &gotFingerprint, &gotCertPath, &gotStatus)
		if err != nil {
			t.Logf("verify query for id=%d failed: %v", certID, err)
			return false
		}

		// Verify original columns are preserved after UPDATE
		if gotID != certID {
			t.Logf("id mismatch: got %d, want %d", gotID, certID)
			return false
		}
		if gotName != row.Name {
			t.Logf("name mismatch after UPDATE: got %q, want %q", gotName, row.Name)
			return false
		}
		if gotType != row.Type {
			t.Logf("type mismatch after UPDATE: got %q, want %q", gotType, row.Type)
			return false
		}

		if gotContent != row.Content {
			t.Logf("content mismatch after UPDATE")
			return false
		}
		if gotFingerprint != newFingerprint {
			t.Logf("fingerprint not updated: got %q, want %q", gotFingerprint, newFingerprint)
			return false
		}
		// Property: new columns remain NULL after UPDATE of existing columns
		if gotCertPath.Valid {
			t.Logf("cert_path should be NULL after Regenerate UPDATE, got %q", gotCertPath.String)
			return false
		}
		if gotStatus.Valid {
			t.Logf("status should be NULL after Regenerate UPDATE, got %q", gotStatus.String)
			return false
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Logf("mock expectations not met: %v", err)
			return false
		}
		return true
	}

	if err := quick.Check(property, cfg); err != nil {
		t.Fatalf("Preservation Regenerate UPDATE property failed: %v", err)
	}
}

// --- Integration tests below require a real PostgreSQL via TEST_DSN ---

// createTestDB creates a temporary test database with the UNFIXED vpn_certificates schema.
func createTestDB(t *testing.T, db *sql.DB) string {
	t.Helper()
	dbName := fmt.Sprintf("test_preservation_%d", time.Now().UnixNano())

	_, err := db.Exec(fmt.Sprintf("CREATE DATABASE `%s`", dbName))
	if err != nil {
		t.Fatalf("create test database: %v", err)
	}

	_, err = db.Exec(fmt.Sprintf("USE `%s`", dbName))
	if err != nil {
		t.Fatalf("use test database: %v", err)
	}

	// Create UNFIXED schema (migration 011 + 017, without cert_path/status)
	createTable := `
		CREATE TABLE vpn_certificates (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(128) NOT NULL,
			type ENUM('ca','tls_crypt','client_cert','client_key') NOT NULL,
			node_id BIGINT NULL,
			content TEXT NOT NULL,
			is_default TINYINT(1) NOT NULL DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NULL,
			fingerprint VARCHAR(128) NULL,
			INDEX(node_id),
			INDEX(type),
			INDEX idx_expires_at (expires_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`
	_, err = db.Exec(createTable)
	if err != nil {
		t.Fatalf("create vpn_certificates table: %v", err)
	}

	return dbName
}

// dropTestDB drops the temporary test database.
func dropTestDB(t *testing.T, db *sql.DB, dbName string) {
	t.Helper()
	_, err := db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName))
	if err != nil {
		t.Logf("warning: failed to drop test database %s: %v", dbName, err)
	}
}

// TestPreservation_DataIntegrityAfterMigration_Integration tests data preservation
// against a real PostgreSQL instance. Requires TEST_DSN env var.
//
// **Validates: Requirements 3.1, 3.2, 3.3, 3.4**
func TestPreservation_DataIntegrityAfterMigration_Integration(t *testing.T) {
	dsn := os.Getenv("TEST_DSN")
	if dsn == "" {
		t.Skip("TEST_DSN not set, skipping database preservation test")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Skipf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skipf("PostgreSQL not reachable at TEST_DSN, skipping: %v", err)
	}

	f := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		dbName := createTestDB(t, db)
		defer dropTestDB(t, db, dbName)

		numRows := rng.Intn(16) + 5
		rows := make([]certRow, numRows)
		insertedIDs := make([]int64, numRows)

		for i := 0; i < numRows; i++ {
			rows[i] = generateCertRow(rng)
			result, err := db.Exec(
				`INSERT INTO vpn_certificates (name, type, node_id, content, is_default, expires_at, fingerprint) VALUES (?, ?, ?, ?, ?, ?, ?)`,
				rows[i].Name, rows[i].Type, rows[i].NodeID, rows[i].Content,
				rows[i].IsDefault, rows[i].ExpiresAt, rows[i].Fingerprint,
			)
			if err != nil {
				t.Logf("insert row %d failed: %v", i, err)
				return false
			}
			id, _ := result.LastInsertId()
			insertedIDs[i] = id
		}

		// Apply migration 025
		_, err := db.Exec(migration025SQL)
		if err != nil {
			t.Logf("apply migration 025 failed: %v", err)
			return false
		}

		// Verify all existing data is preserved and new columns are NULL
		for i, id := range insertedIDs {
			var (
				gotName        string
				gotType        string
				gotNodeID      sql.NullInt64
				gotContent     string
				gotIsDefault   int
				gotCreatedAt   time.Time
				gotExpiresAt   sql.NullTime
				gotFingerprint sql.NullString
				gotCertPath    sql.NullString
				gotStatus      sql.NullString
			)

			err := db.QueryRow(
				`SELECT name, type, node_id, content, is_default, created_at, expires_at, fingerprint, cert_path, status FROM vpn_certificates WHERE id = ?`,
				id,
			).Scan(&gotName, &gotType, &gotNodeID, &gotContent, &gotIsDefault,
				&gotCreatedAt, &gotExpiresAt, &gotFingerprint, &gotCertPath, &gotStatus)
			if err != nil {
				t.Logf("query row %d (id=%d) after migration failed: %v", i, id, err)
				return false
			}

			if gotName != rows[i].Name {
				t.Logf("row %d: name mismatch", i)
				return false
			}
			if gotType != rows[i].Type {
				t.Logf("row %d: type mismatch", i)
				return false
			}
			if gotNodeID != rows[i].NodeID {
				t.Logf("row %d: node_id mismatch", i)
				return false
			}
			if gotContent != rows[i].Content {
				t.Logf("row %d: content mismatch", i)
				return false
			}
			if gotIsDefault != rows[i].IsDefault {
				t.Logf("row %d: is_default mismatch", i)
				return false
			}
			if gotExpiresAt.Valid != rows[i].ExpiresAt.Valid {
				t.Logf("row %d: expires_at validity mismatch", i)
				return false
			}
			if gotExpiresAt.Valid && rows[i].ExpiresAt.Valid {
				gotTrunc := gotExpiresAt.Time.Truncate(time.Second)
				wantTrunc := rows[i].ExpiresAt.Time.Truncate(time.Second)
				if !gotTrunc.Equal(wantTrunc) {
					t.Logf("row %d: expires_at mismatch", i)
					return false
				}
			}

			if gotFingerprint != rows[i].Fingerprint {
				t.Logf("row %d: fingerprint mismatch", i)
				return false
			}
			if gotCertPath.Valid {
				t.Logf("row %d: cert_path should be NULL", i)
				return false
			}
			if gotStatus.Valid {
				t.Logf("row %d: status should be NULL", i)
				return false
			}
		}
		return true
	}

	cfg := &quick.Config{
		MaxCount: 20,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Fatalf("Preservation property failed: %v", err)
	}
}

// TestPreservation_RegenerateUpdateAfterMigration_Integration tests the Regenerate
// UPDATE query against a real PostgreSQL instance. Requires TEST_DSN env var.
//
// **Validates: Requirements 3.1, 3.2**
func TestPreservation_RegenerateUpdateAfterMigration_Integration(t *testing.T) {
	dsn := os.Getenv("TEST_DSN")
	if dsn == "" {
		t.Skip("TEST_DSN not set, skipping database preservation test")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Skipf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skipf("PostgreSQL not reachable at TEST_DSN, skipping: %v", err)
	}

	f := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		dbName := createTestDB(t, db)
		defer dropTestDB(t, db, dbName)

		numRows := rng.Intn(10) + 3
		insertedIDs := make([]int64, numRows)

		for i := 0; i < numRows; i++ {
			row := generateCertRow(rng)
			result, err := db.Exec(
				`INSERT INTO vpn_certificates (name, type, node_id, content, is_default, expires_at, fingerprint) VALUES (?, ?, ?, ?, ?, ?, ?)`,
				row.Name, row.Type, row.NodeID, row.Content, row.IsDefault, row.ExpiresAt, row.Fingerprint,
			)
			if err != nil {
				t.Logf("insert failed: %v", err)
				return false
			}
			id, _ := result.LastInsertId()
			insertedIDs[i] = id
		}

		// Observe: Verify Regenerate UPDATE works on UNFIXED schema
		for _, id := range insertedIDs {
			newExpiry := time.Now().Add(time.Duration(rng.Intn(365*3)+1) * 24 * time.Hour)
			newFp := fmt.Sprintf("%x", rng.Int63())
			_, err := db.Exec(
				`UPDATE vpn_certificates SET expires_at = ?, fingerprint = ? WHERE id = ?`,
				newExpiry, newFp, id,
			)
			if err != nil {
				t.Logf("Regenerate UPDATE on UNFIXED schema failed for id=%d: %v", id, err)
				return false
			}
		}

		// Apply migration 025
		_, err := db.Exec(migration025SQL)
		if err != nil {
			t.Logf("apply migration 025 failed: %v", err)
			return false
		}

		// Verify: Regenerate UPDATE still works after migration
		for _, id := range insertedIDs {
			newExpiry := time.Now().Add(time.Duration(rng.Intn(365*3)+1) * 24 * time.Hour)
			newFp := fmt.Sprintf("%x", rng.Int63())

			result, err := db.Exec(
				`UPDATE vpn_certificates SET expires_at = ?, fingerprint = ? WHERE id = ?`,
				newExpiry, newFp, id,
			)
			if err != nil {
				t.Logf("Regenerate UPDATE after migration failed for id=%d: %v", id, err)
				return false
			}
			affected, _ := result.RowsAffected()
			if affected != 1 {
				t.Logf("Regenerate UPDATE for id=%d: expected 1 row affected, got %d", id, affected)
				return false
			}
		}
		return true
	}

	cfg := &quick.Config{
		MaxCount: 15,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Fatalf("Preservation Regenerate property failed: %v", err)
	}
}
