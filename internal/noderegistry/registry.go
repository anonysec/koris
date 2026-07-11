package noderegistry

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

// NodeRecord represents a node's connection credentials stored in the database.
type NodeRecord struct {
	ID            int64
	Name          string
	Address       string
	Port          int
	APIKeyEnc     []byte // AES-GCM encrypted (stored form)
	ClientCertPEM []byte // PEM-encoded client certificate
	ClientKeyEnc  []byte // AES-GCM encrypted (stored form) or plaintext PEM before Create
	CACertPEM     []byte // PEM-encoded CA certificate
	Enabled       bool
	Status        string
	LastSeenAt    sql.NullTime
	OwnerWorker   sql.NullString
	Domain        sql.NullString
	BackupDomains sql.NullString
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NodeInput holds the plaintext credentials for creating or updating a node record.
// The API key and client key are in plaintext and will be encrypted before storage.
type NodeInput struct {
	Name          string
	Address       string
	Port          int
	APIKey        []byte // plaintext — will be encrypted
	ClientCertPEM []byte // PEM
	ClientKeyPEM  []byte // plaintext PEM — will be encrypted
	CACertPEM     []byte // PEM
	Enabled       bool
}

// Registry manages node records in the database.
type Registry interface {
	Create(ctx context.Context, input *NodeInput) (int64, error)
	Update(ctx context.Context, id int64, input *NodeInput) error
	Delete(ctx context.Context, id int64) error
	Get(ctx context.Context, id int64) (*NodeRecord, error)
	ListEnabled(ctx context.Context) ([]*NodeRecord, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
}

// DBRegistry is a database-backed implementation of Registry.
type DBRegistry struct {
	db        *sql.DB
	encryptor *Encryptor
	validator Validator
}

// NewDBRegistry creates a new database-backed registry.
func NewDBRegistry(db *sql.DB, encryptor *Encryptor) *DBRegistry {
	return &DBRegistry{
		db:        db,
		encryptor: encryptor,
		validator: Validator{},
	}
}

// Create validates and persists a new node record. The API key and client key
// are encrypted before storage. Returns the new record ID.
func (r *DBRegistry) Create(ctx context.Context, input *NodeInput) (int64, error) {
	// Build a temporary NodeRecord for validation (PEM fields validated as plaintext)
	record := &NodeRecord{
		Name:          input.Name,
		Address:       input.Address,
		Port:          input.Port,
		ClientCertPEM: input.ClientCertPEM,
		ClientKeyEnc:  input.ClientKeyPEM, // validator checks PEM validity on the plaintext
		CACertPEM:     input.CACertPEM,
	}
	if err := r.validator.Validate(record); err != nil {
		return 0, fmt.Errorf("validation failed: %w", err)
	}
	if len(input.APIKey) == 0 {
		return 0, ErrEmptyAPIKey
	}

	// Encrypt sensitive fields
	apiKeyEnc, err := r.encryptor.Encrypt(input.APIKey)
	if err != nil {
		return 0, fmt.Errorf("encrypt api key: %w", err)
	}
	clientKeyEnc, err := r.encryptor.Encrypt(input.ClientKeyPEM)
	if err != nil {
		return 0, fmt.Errorf("encrypt client key: %w", err)
	}

	now := time.Now().UTC()
	var id int64
	err = r.db.QueryRowContext(ctx,
		`INSERT INTO knode_connections (name, address, grpc_port, api_key_enc, client_cert, client_key_enc, ca_cert, enabled, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'offline', $9, $10)
		 RETURNING id`,
		input.Name, input.Address, input.Port, apiKeyEnc, input.ClientCertPEM, clientKeyEnc, input.CACertPEM, input.Enabled, now, now,
	).Scan(&id)
	if err != nil {
		log.Printf("[noderegistry] Create failed for %q: %v", input.Name, err)
		return 0, fmt.Errorf("insert node: %w", err)
	}

	log.Printf("[noderegistry] Created node %q (id=%d) at %s:%d", input.Name, id, input.Address, input.Port)
	return id, nil
}

// Update validates and updates an existing node record. Credentials are re-encrypted.
func (r *DBRegistry) Update(ctx context.Context, id int64, input *NodeInput) error {
	// Load existing secret material so masked fields that the UI does not resend
	// (API key, client cert/key, CA cert) are preserved instead of being wiped.
	var (
		curAPIKeyEnc, curClientCert, curClientKeyEnc, curCACert string
	)
	err := r.db.QueryRowContext(ctx,
		`SELECT api_key_enc, client_cert, client_key_enc, ca_cert FROM knode_connections WHERE id = $1`,
		id,
	).Scan(&curAPIKeyEnc, &curClientCert, &curClientKeyEnc, &curCACert)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("node id=%d: %w", id, ErrNodeNotFound)
		}
		return fmt.Errorf("load node: %w", err)
	}

	// Resolve effective values for validation: keep existing when input is empty.
	// Note: client cert and CA cert are stored as plaintext PEM; the client key is
	// stored AES-GCM encrypted, so it must be blanked (not reused) for PEM validation.
	clientCert := input.ClientCertPEM
	if len(clientCert) == 0 {
		clientCert = []byte(curClientCert)
	}
	clientKey := input.ClientKeyPEM
	if len(clientKey) == 0 {
		clientKey = []byte{}
	}
	caCert := input.CACertPEM
	if len(caCert) == 0 {
		caCert = []byte(curCACert)
	}
	record := &NodeRecord{
		Name:          input.Name,
		Address:       input.Address,
		Port:          input.Port,
		ClientCertPEM: clientCert,
		ClientKeyEnc:  clientKey,
		CACertPEM:     caCert,
	}
	if err := r.validator.Validate(record); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Build the UPDATE dynamically: only overwrite secret columns that were provided.
	setClauses := make([]string, 0, 9)
	args := make([]any, 0, 9)
	n := 1
	add := func(clause string, val any) {
		n++
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", clause, n))
		args = append(args, val)
	}
	add("name", input.Name)
	add("address", input.Address)
	add("grpc_port", input.Port)
	add("enabled", input.Enabled)
	add("updated_at", time.Now().UTC())
	if len(input.APIKey) > 0 {
		enc, e := r.encryptor.Encrypt(input.APIKey)
		if e != nil {
			return fmt.Errorf("encrypt api key: %w", e)
		}
		add("api_key_enc", enc)
	}
	if len(input.ClientCertPEM) > 0 {
		add("client_cert", input.ClientCertPEM)
	}
	if len(input.ClientKeyPEM) > 0 {
		enc, e := r.encryptor.Encrypt(input.ClientKeyPEM)
		if e != nil {
			return fmt.Errorf("encrypt client key: %w", e)
		}
		add("client_key_enc", enc)
	}
	if len(input.CACertPEM) > 0 {
		add("ca_cert", input.CACertPEM)
	}

	query := fmt.Sprintf(`UPDATE knode_connections SET %s WHERE id = $1`, strings.Join(setClauses, ", "))
	args = append([]any{id}, args...)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		log.Printf("[noderegistry] Update failed for id=%d: %v", id, err)
		return fmt.Errorf("update node: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("node id=%d: %w", id, ErrNodeNotFound)
	}

	log.Printf("[noderegistry] Updated node id=%d (%q)", id, input.Name)
	return nil
}

// Delete removes a node record and all associated credentials from the database.
func (r *DBRegistry) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM knode_connections WHERE id = $1`, id)
	if err != nil {
		log.Printf("[noderegistry] Delete failed for id=%d: %v", id, err)
		return fmt.Errorf("delete node: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("node id=%d: %w", id, ErrNodeNotFound)
	}

	log.Printf("[noderegistry] Deleted node id=%d", id)
	return nil
}

// Get retrieves a single node record by ID.
func (r *DBRegistry) Get(ctx context.Context, id int64) (*NodeRecord, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, address, grpc_port, api_key_enc, client_cert, client_key_enc, ca_cert, enabled, status, last_seen_at, owner_worker, domain, backup_domains, created_at, updated_at
		 FROM knode_connections WHERE id = $1`, id)

	rec, err := scanNodeRecord(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNodeNotFound
		}
		return nil, fmt.Errorf("get node id=%d: %w", id, err)
	}
	return rec, nil
}

// ListEnabled returns all node records where enabled = true.
func (r *DBRegistry) ListEnabled(ctx context.Context) ([]*NodeRecord, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, address, grpc_port, api_key_enc, client_cert, client_key_enc, ca_cert, enabled, status, last_seen_at, owner_worker, domain, backup_domains, created_at, updated_at
		 FROM knode_connections WHERE enabled = $1`, true)
	if err != nil {
		return nil, fmt.Errorf("list enabled nodes: %w", err)
	}
	defer rows.Close()

	var records []*NodeRecord
	for rows.Next() {
		rec, err := scanNodeRecordRows(rows)
		if err != nil {
			return nil, fmt.Errorf("scan node: %w", err)
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return records, nil
}

// UpdateStatus sets the status field (online, offline, stale) and last_seen_at for a node.
func (r *DBRegistry) UpdateStatus(ctx context.Context, id int64, status string) error {
	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx,
		`UPDATE knode_connections SET status = $1, last_seen_at = $2, updated_at = $3 WHERE id = $4`,
		status, now, now, id)
	if err != nil {
		return fmt.Errorf("update status for id=%d: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("node id=%d: %w", id, ErrNodeNotFound)
	}
	return nil
}

// DecryptAPIKey decrypts the stored API key for a NodeRecord.
func (r *DBRegistry) DecryptAPIKey(rec *NodeRecord) ([]byte, error) {
	return r.encryptor.Decrypt(rec.APIKeyEnc)
}

// DecryptClientKey decrypts the stored client private key for a NodeRecord.
func (r *DBRegistry) DecryptClientKey(rec *NodeRecord) ([]byte, error) {
	return r.encryptor.Decrypt(rec.ClientKeyEnc)
}

// ErrNodeNotFound indicates the requested node does not exist.
var ErrNodeNotFound = errors.New("node not found")

// scanNodeRecord scans a single row into a NodeRecord.
func scanNodeRecord(row *sql.Row) (*NodeRecord, error) {
	rec := &NodeRecord{}
	err := row.Scan(
		&rec.ID, &rec.Name, &rec.Address, &rec.Port,
		&rec.APIKeyEnc, &rec.ClientCertPEM, &rec.ClientKeyEnc, &rec.CACertPEM,
		&rec.Enabled, &rec.Status, &rec.LastSeenAt, &rec.OwnerWorker,
		&rec.Domain, &rec.BackupDomains,
		&rec.CreatedAt, &rec.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return rec, nil
}

// scanNodeRecordRows scans a rows result into a NodeRecord.
func scanNodeRecordRows(rows *sql.Rows) (*NodeRecord, error) {
	rec := &NodeRecord{}
	err := rows.Scan(
		&rec.ID, &rec.Name, &rec.Address, &rec.Port,
		&rec.APIKeyEnc, &rec.ClientCertPEM, &rec.ClientKeyEnc, &rec.CACertPEM,
		&rec.Enabled, &rec.Status, &rec.LastSeenAt, &rec.OwnerWorker,
		&rec.Domain, &rec.BackupDomains,
		&rec.CreatedAt, &rec.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return rec, nil
}
