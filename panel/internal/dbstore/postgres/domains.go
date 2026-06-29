// Package postgres — domain management database operations.
// Implements CRUD for vpn_domains, vpn_domain_ip_history, vpn_protocol_bindings,
// and per-customer MTProto secret management.
package postgres

import (
	"context"
	"fmt"
	"time"

	"KorisPanel/panel/internal/dbstore"

	"github.com/jackc/pgx/v5"
)

// --- Domain CRUD ---

// CreateDomain inserts a new domain record and returns the created domain with ID and timestamps populated.
func (s *Store) CreateDomain(ctx context.Context, name, ipAddress string) (*dbstore.Domain, error) {
	var d dbstore.Domain
	err := s.pool.QueryRow(ctx, `
		INSERT INTO vpn_domains (name, ip_address, status, created_at, updated_at)
		VALUES ($1, $2, 'active', NOW(), NOW())
		RETURNING id, name, ip_address, status, created_at, updated_at
	`, name, ipAddress).Scan(&d.ID, &d.Name, &d.IPAddress, &d.Status, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, wrapPgError(err)
	}
	return &d, nil
}

// GetDomain retrieves a single domain by ID.
// Returns dbstore.ErrNotFound if the domain does not exist.
func (s *Store) GetDomain(ctx context.Context, id int64) (*dbstore.Domain, error) {
	var d dbstore.Domain
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, ip_address, status, created_at, updated_at
		FROM vpn_domains
		WHERE id = $1
	`, id).Scan(&d.ID, &d.Name, &d.IPAddress, &d.Status, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, dbstore.ErrNotFound
		}
		return nil, wrapPgError(err)
	}
	return &d, nil
}

// ListDomains returns all domains ordered by created_at DESC, with computed binding_count
// and cert_status from JOINs on vpn_protocol_bindings and vpn_certificates.
func (s *Store) ListDomains(ctx context.Context) ([]dbstore.Domain, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			d.id, d.name, d.ip_address, d.status, d.created_at, d.updated_at,
			COALESCE(b.binding_count, 0),
			COALESCE(
				CASE
					WHEN c.expires_at IS NULL THEN 'none'
					WHEN c.expires_at <= NOW() THEN 'expired'
					WHEN c.expires_at <= NOW() + INTERVAL '30 days' THEN 'expiring_soon'
					ELSE 'valid'
				END,
				'none'
			)
		FROM vpn_domains d
		LEFT JOIN (
			SELECT domain_id, COUNT(*) AS binding_count
			FROM vpn_protocol_bindings
			GROUP BY domain_id
		) b ON b.domain_id = d.id
		LEFT JOIN LATERAL (
			SELECT expires_at
			FROM vpn_certificates
			WHERE domain_id = d.id AND status IN ('active', 'pending')
			ORDER BY expires_at DESC
			LIMIT 1
		) c ON true
		ORDER BY d.created_at DESC
	`)
	if err != nil {
		return nil, wrapPgError(err)
	}
	defer rows.Close()

	var domains []dbstore.Domain
	for rows.Next() {
		var d dbstore.Domain
		if err := rows.Scan(
			&d.ID, &d.Name, &d.IPAddress, &d.Status, &d.CreatedAt, &d.UpdatedAt,
			&d.BindingCount, &d.CertStatus,
		); err != nil {
			return nil, fmt.Errorf("postgres: scan domain row: %w", err)
		}
		domains = append(domains, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: iterate domain rows: %w", err)
	}
	return domains, nil
}

// UpdateDomain updates the ip_address and/or status fields for a domain.
// Returns dbstore.ErrNotFound if the domain does not exist.
func (s *Store) UpdateDomain(ctx context.Context, id int64, ipAddress, status string) (*dbstore.Domain, error) {
	var d dbstore.Domain
	err := s.pool.QueryRow(ctx, `
		UPDATE vpn_domains
		SET ip_address = COALESCE(NULLIF($2, ''), ip_address),
		    status = COALESCE(NULLIF($3, ''), status),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, ip_address, status, created_at, updated_at
	`, id, ipAddress, status).Scan(&d.ID, &d.Name, &d.IPAddress, &d.Status, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, dbstore.ErrNotFound
		}
		return nil, wrapPgError(err)
	}
	return &d, nil
}

// DeleteDomain removes a domain by ID.
// Returns dbstore.ErrConstraintViolation if active protocol bindings reference it (ON DELETE RESTRICT).
// Returns dbstore.ErrNotFound if the domain does not exist.
func (s *Store) DeleteDomain(ctx context.Context, id int64) error {
	result, err := s.pool.Exec(ctx, `DELETE FROM vpn_domains WHERE id = $1`, id)
	if err != nil {
		return wrapPgError(err)
	}
	if result.RowsAffected() == 0 {
		return dbstore.ErrNotFound
	}
	return nil
}

// GetDomainBindings returns all protocol bindings that reference a given domain.
// Used to check before deletion whether the domain is still in use.
func (s *Store) GetDomainBindings(ctx context.Context, domainID int64) ([]dbstore.ProtocolBinding, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, node_id, protocol, domain_id, position, created_at
		FROM vpn_protocol_bindings
		WHERE domain_id = $1
		ORDER BY node_id, protocol, position
	`, domainID)
	if err != nil {
		return nil, wrapPgError(err)
	}
	defer rows.Close()

	var bindings []dbstore.ProtocolBinding
	for rows.Next() {
		var b dbstore.ProtocolBinding
		if err := rows.Scan(&b.ID, &b.NodeID, &b.Protocol, &b.DomainID, &b.Position, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("postgres: scan binding row: %w", err)
		}
		bindings = append(bindings, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: iterate binding rows: %w", err)
	}
	return bindings, nil
}

// --- Domain IP History ---

// CreateDomainIPHistory inserts an IP rotation audit log entry.
func (s *Store) CreateDomainIPHistory(ctx context.Context, domainID int64, previousIP, newIP, adminUsername string) (*dbstore.DomainIPHistory, error) {
	var h dbstore.DomainIPHistory
	err := s.pool.QueryRow(ctx, `
		INSERT INTO vpn_domain_ip_history (domain_id, previous_ip, new_ip, admin_username, rotated_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id, domain_id, previous_ip, new_ip, admin_username, rotated_at
	`, domainID, previousIP, newIP, adminUsername).Scan(
		&h.ID, &h.DomainID, &h.PreviousIP, &h.NewIP, &h.AdminUsername, &h.RotatedAt,
	)
	if err != nil {
		return nil, wrapPgError(err)
	}
	return &h, nil
}

// ListDomainIPHistory returns all IP rotation records for a domain, ordered by rotated_at DESC.
func (s *Store) ListDomainIPHistory(ctx context.Context, domainID int64) ([]dbstore.DomainIPHistory, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, domain_id, previous_ip, new_ip, admin_username, rotated_at
		FROM vpn_domain_ip_history
		WHERE domain_id = $1
		ORDER BY rotated_at DESC
	`, domainID)
	if err != nil {
		return nil, wrapPgError(err)
	}
	defer rows.Close()

	var history []dbstore.DomainIPHistory
	for rows.Next() {
		var h dbstore.DomainIPHistory
		if err := rows.Scan(&h.ID, &h.DomainID, &h.PreviousIP, &h.NewIP, &h.AdminUsername, &h.RotatedAt); err != nil {
			return nil, fmt.Errorf("postgres: scan ip history row: %w", err)
		}
		history = append(history, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: iterate ip history rows: %w", err)
	}
	return history, nil
}

// --- Protocol Bindings ---

// CreateProtocolBinding creates a new protocol-to-domain binding for a node.
func (s *Store) CreateProtocolBinding(ctx context.Context, nodeID int64, protocol string, domainID int64, position int) (*dbstore.ProtocolBinding, error) {
	var b dbstore.ProtocolBinding
	err := s.pool.QueryRow(ctx, `
		INSERT INTO vpn_protocol_bindings (node_id, protocol, domain_id, position, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id, node_id, protocol, domain_id, position, created_at
	`, nodeID, protocol, domainID, position).Scan(
		&b.ID, &b.NodeID, &b.Protocol, &b.DomainID, &b.Position, &b.CreatedAt,
	)
	if err != nil {
		return nil, wrapPgError(err)
	}
	return &b, nil
}

// ListProtocolBindings returns all protocol bindings for a node, ordered by protocol and position,
// with joined domain details (name, IP, status).
func (s *Store) ListProtocolBindings(ctx context.Context, nodeID int64) ([]dbstore.ProtocolBinding, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT pb.id, pb.node_id, pb.protocol, pb.domain_id, pb.position, pb.created_at,
		       d.name, d.ip_address, d.status
		FROM vpn_protocol_bindings pb
		JOIN vpn_domains d ON d.id = pb.domain_id
		WHERE pb.node_id = $1
		ORDER BY pb.protocol, pb.position
	`, nodeID)
	if err != nil {
		return nil, wrapPgError(err)
	}
	defer rows.Close()

	var bindings []dbstore.ProtocolBinding
	for rows.Next() {
		var b dbstore.ProtocolBinding
		if err := rows.Scan(
			&b.ID, &b.NodeID, &b.Protocol, &b.DomainID, &b.Position, &b.CreatedAt,
			&b.DomainName, &b.DomainIP, &b.DomainStatus,
		); err != nil {
			return nil, fmt.Errorf("postgres: scan protocol binding row: %w", err)
		}
		bindings = append(bindings, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: iterate protocol binding rows: %w", err)
	}
	return bindings, nil
}

// DeleteProtocolBinding removes a binding by ID and re-sequences remaining positions
// for the same (node_id, protocol) combination to be contiguous starting at 1.
func (s *Store) DeleteProtocolBinding(ctx context.Context, bindingID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get the binding's node_id and protocol before deleting
	var nodeID int64
	var protocol string
	err = tx.QueryRow(ctx, `
		SELECT node_id, protocol FROM vpn_protocol_bindings WHERE id = $1
	`, bindingID).Scan(&nodeID, &protocol)
	if err != nil {
		if err == pgx.ErrNoRows {
			return dbstore.ErrNotFound
		}
		return wrapPgError(err)
	}

	// Delete the binding
	_, err = tx.Exec(ctx, `DELETE FROM vpn_protocol_bindings WHERE id = $1`, bindingID)
	if err != nil {
		return wrapPgError(err)
	}

	// Re-sequence remaining bindings for same (node_id, protocol) to be contiguous from 1
	_, err = tx.Exec(ctx, `
		WITH ranked AS (
			SELECT id, ROW_NUMBER() OVER (ORDER BY position) AS new_pos
			FROM vpn_protocol_bindings
			WHERE node_id = $1 AND protocol = $2
		)
		UPDATE vpn_protocol_bindings
		SET position = ranked.new_pos
		FROM ranked
		WHERE vpn_protocol_bindings.id = ranked.id
	`, nodeID, protocol)
	if err != nil {
		return wrapPgError(err)
	}

	return tx.Commit(ctx)
}

// ReorderProtocolBindings accepts a slice of binding IDs in the desired order and updates
// their positions atomically (1-based). All IDs must belong to the same (node_id, protocol).
func (s *Store) ReorderProtocolBindings(ctx context.Context, bindingIDs []int64) error {
	if len(bindingIDs) == 0 {
		return nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Verify all IDs belong to the same (node_id, protocol)
	var nodeID int64
	var protocol string
	err = tx.QueryRow(ctx, `
		SELECT node_id, protocol FROM vpn_protocol_bindings WHERE id = $1
	`, bindingIDs[0]).Scan(&nodeID, &protocol)
	if err != nil {
		if err == pgx.ErrNoRows {
			return dbstore.ErrNotFound
		}
		return wrapPgError(err)
	}

	// Temporarily set all positions to negative values to avoid unique constraint conflicts
	for i, id := range bindingIDs {
		_, err = tx.Exec(ctx, `
			UPDATE vpn_protocol_bindings
			SET position = $1
			WHERE id = $2 AND node_id = $3 AND protocol = $4
		`, -(i + 1), id, nodeID, protocol)
		if err != nil {
			return wrapPgError(err)
		}
	}

	// Now set the actual positions
	for i, id := range bindingIDs {
		_, err = tx.Exec(ctx, `
			UPDATE vpn_protocol_bindings
			SET position = $1
			WHERE id = $2 AND node_id = $3 AND protocol = $4
		`, i+1, id, nodeID, protocol)
		if err != nil {
			return wrapPgError(err)
		}
	}

	return tx.Commit(ctx)
}

// --- MTProto Secret Management ---

// MTProtoInfo holds customer MTProto secret details.
type MTProtoInfo struct {
	Secret    *string `json:"secret"`
	Enabled   bool    `json:"enabled"`
	ConnLimit int     `json:"conn_limit"`
}

// GetCustomerMTProtoSecret retrieves the MTProto secret, enabled status, and connection limit
// for a customer. Returns dbstore.ErrNotFound if the customer does not exist.
func (s *Store) GetCustomerMTProtoSecret(ctx context.Context, customerID int64) (*MTProtoInfo, error) {
	var info MTProtoInfo
	err := s.pool.QueryRow(ctx, `
		SELECT mtproto_secret, mtproto_enabled, mtproto_conn_limit
		FROM customers
		WHERE id = $1
	`, customerID).Scan(&info.Secret, &info.Enabled, &info.ConnLimit)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, dbstore.ErrNotFound
		}
		return nil, wrapPgError(err)
	}
	return &info, nil
}

// SetCustomerMTProtoSecret updates the MTProto secret and enabled status for a customer.
// Returns dbstore.ErrNotFound if the customer does not exist.
func (s *Store) SetCustomerMTProtoSecret(ctx context.Context, customerID int64, secret string, enabled bool) error {
	result, err := s.pool.Exec(ctx, `
		UPDATE customers
		SET mtproto_secret = $2, mtproto_enabled = $3, updated_at = NOW()
		WHERE id = $1
	`, customerID, secret, enabled)
	if err != nil {
		return wrapPgError(err)
	}
	if result.RowsAffected() == 0 {
		return dbstore.ErrNotFound
	}
	return nil
}

// RegenerateMTProtoSecret generates and stores a new MTProto secret for a customer.
// The secret parameter should be a pre-generated 64-char hex string (32 random bytes).
// Returns dbstore.ErrNotFound if the customer does not exist.
func (s *Store) RegenerateMTProtoSecret(ctx context.Context, customerID int64, newSecret string) error {
	result, err := s.pool.Exec(ctx, `
		UPDATE customers
		SET mtproto_secret = $2, mtproto_enabled = true, updated_at = NOW()
		WHERE id = $1
	`, customerID, newSecret)
	if err != nil {
		return wrapPgError(err)
	}
	if result.RowsAffected() == 0 {
		return dbstore.ErrNotFound
	}
	return nil
}

// --- VPN Certificates ---

// CreateVpnCertificate inserts a new certificate record.
func (s *Store) CreateVpnCertificate(ctx context.Context, cert *dbstore.VpnCertificate) (*dbstore.VpnCertificate, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO vpn_certificates (node_id, domain_id, cert_type, status, certificate, private_key, ca_chain, issued_at, expires_at, retry_count, last_error, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`, cert.NodeID, cert.DomainID, cert.CertType, cert.Status, cert.Certificate, cert.PrivateKey, cert.CAChain, cert.IssuedAt, cert.ExpiresAt, cert.RetryCount, cert.LastError).Scan(
		&cert.ID, &cert.CreatedAt, &cert.UpdatedAt,
	)
	if err != nil {
		return nil, wrapPgError(err)
	}
	return cert, nil
}

// GetVpnCertificateByDomain retrieves the most recent active/pending certificate for a domain.
func (s *Store) GetVpnCertificateByDomain(ctx context.Context, domainID int64) (*dbstore.VpnCertificate, error) {
	var c dbstore.VpnCertificate
	err := s.pool.QueryRow(ctx, `
		SELECT id, node_id, domain_id, cert_type, status, certificate, private_key, ca_chain,
		       issued_at, expires_at, retry_count, last_error, created_at, updated_at
		FROM vpn_certificates
		WHERE domain_id = $1 AND status IN ('active', 'pending')
		ORDER BY created_at DESC
		LIMIT 1
	`, domainID).Scan(
		&c.ID, &c.NodeID, &c.DomainID, &c.CertType, &c.Status, &c.Certificate, &c.PrivateKey, &c.CAChain,
		&c.IssuedAt, &c.ExpiresAt, &c.RetryCount, &c.LastError, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, dbstore.ErrNotFound
		}
		return nil, wrapPgError(err)
	}
	return &c, nil
}

// UpdateVpnCertificate updates a certificate record (status, cert data, retry info).
func (s *Store) UpdateVpnCertificate(ctx context.Context, cert *dbstore.VpnCertificate) error {
	result, err := s.pool.Exec(ctx, `
		UPDATE vpn_certificates
		SET status = $2, certificate = $3, private_key = $4, ca_chain = $5,
		    issued_at = $6, expires_at = $7, retry_count = $8, last_error = $9, updated_at = NOW()
		WHERE id = $1
	`, cert.ID, cert.Status, cert.Certificate, cert.PrivateKey, cert.CAChain,
		cert.IssuedAt, cert.ExpiresAt, cert.RetryCount, cert.LastError)
	if err != nil {
		return wrapPgError(err)
	}
	if result.RowsAffected() == 0 {
		return dbstore.ErrNotFound
	}
	return nil
}

// ListExpiringCertificates returns certificates that are active and expire within the given duration.
func (s *Store) ListExpiringCertificates(ctx context.Context, within time.Duration) ([]dbstore.VpnCertificate, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, node_id, domain_id, cert_type, status, certificate, private_key, ca_chain,
		       issued_at, expires_at, retry_count, last_error, created_at, updated_at
		FROM vpn_certificates
		WHERE status = 'active' AND expires_at <= NOW() + $1::interval
		ORDER BY expires_at ASC
	`, within.String())
	if err != nil {
		return nil, wrapPgError(err)
	}
	defer rows.Close()

	var certs []dbstore.VpnCertificate
	for rows.Next() {
		var c dbstore.VpnCertificate
		if err := rows.Scan(
			&c.ID, &c.NodeID, &c.DomainID, &c.CertType, &c.Status, &c.Certificate, &c.PrivateKey, &c.CAChain,
			&c.IssuedAt, &c.ExpiresAt, &c.RetryCount, &c.LastError, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("postgres: scan certificate row: %w", err)
		}
		certs = append(certs, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: iterate certificate rows: %w", err)
	}
	return certs, nil
}
