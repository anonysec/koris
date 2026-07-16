package certrotation

import (
	"context"
	"database/sql"
	"time"
)

// DBStore implements IKEv2CertStore using a *sql.DB connection.
// It queries the vpn_certificates table directly (joined on node id).
type DBStore struct {
	db *sql.DB
}

// NewDBStore creates a new database-backed IKEv2CertStore.
func NewDBStore(db *sql.DB) *DBStore {
	return &DBStore{db: db}
}

// ListIKEv2Domains returns all nodes that have an active IKEv2 certificate.
// (Domain mapping was removed; certs are now keyed by node.)
func (s *DBStore) ListIKEv2Domains(ctx context.Context) ([]IKEv2DomainBinding, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT node_id
		FROM vpn_certificates
		WHERE cert_type = 'ikev2' AND status = 'active'
		GROUP BY node_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bindings []IKEv2DomainBinding
	for rows.Next() {
		var b IKEv2DomainBinding
		if err := rows.Scan(&b.NodeID); err != nil {
			return nil, err
		}
		bindings = append(bindings, b)
	}
	return bindings, rows.Err()
}

// GetCertificateByDomain retrieves the most recent active/pending IKEv2 certificate for a domain.
// Returns nil, nil if no certificate exists.
func (s *DBStore) GetCertificateByDomain(ctx context.Context, domainID int64) (*IKEv2Certificate, error) {
	var c IKEv2Certificate
	var certificate, privateKey, caChain, lastError sql.NullString
	var issuedAt, expiresAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, node_id, domain_id, cert_type, status, certificate, private_key, ca_chain,
		       issued_at, expires_at, retry_count, last_error
		FROM vpn_certificates
		WHERE domain_id = $1 AND cert_type = 'ikev2' AND status IN ('active', 'pending')
		ORDER BY created_at DESC
		LIMIT 1
	`, domainID).Scan(
		&c.ID, &c.NodeID, &c.DomainID, &c.CertType, &c.Status,
		&certificate, &privateKey, &caChain,
		&issuedAt, &expiresAt, &c.RetryCount, &lastError,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if certificate.Valid {
		c.Certificate = certificate.String
	}
	if privateKey.Valid {
		c.PrivateKey = privateKey.String
	}
	if caChain.Valid {
		c.CAChain = caChain.String
	}
	if issuedAt.Valid {
		c.IssuedAt = &issuedAt.Time
	}
	if expiresAt.Valid {
		c.ExpiresAt = &expiresAt.Time
	}
	if lastError.Valid {
		c.LastError = lastError.String
	}

	return &c, nil
}

// CreateCertificate inserts a new certificate record into vpn_certificates.
func (s *DBStore) CreateCertificate(ctx context.Context, cert *IKEv2Certificate) error {
	var certificate, privateKey, caChain, lastError *string
	if cert.Certificate != "" {
		certificate = &cert.Certificate
	}
	if cert.PrivateKey != "" {
		privateKey = &cert.PrivateKey
	}
	if cert.CAChain != "" {
		caChain = &cert.CAChain
	}
	if cert.LastError != "" {
		lastError = &cert.LastError
	}

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO vpn_certificates (node_id, domain_id, cert_type, status, certificate, private_key, ca_chain, issued_at, expires_at, retry_count, last_error, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		RETURNING id
	`, cert.NodeID, cert.DomainID, cert.CertType, cert.Status,
		certificate, privateKey, caChain,
		cert.IssuedAt, cert.ExpiresAt, cert.RetryCount, lastError,
	).Scan(&cert.ID)
	return err
}

// UpdateCertificate updates an existing certificate record (status, cert data, retry info).
func (s *DBStore) UpdateCertificate(ctx context.Context, cert *IKEv2Certificate) error {
	var certificate, privateKey, caChain, lastError *string
	if cert.Certificate != "" {
		certificate = &cert.Certificate
	}
	if cert.PrivateKey != "" {
		privateKey = &cert.PrivateKey
	}
	if cert.CAChain != "" {
		caChain = &cert.CAChain
	}
	if cert.LastError != "" {
		lastError = &cert.LastError
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE vpn_certificates
		SET status = $2, certificate = $3, private_key = $4, ca_chain = $5,
		    issued_at = $6, expires_at = $7, retry_count = $8, last_error = $9, updated_at = NOW()
		WHERE id = $1
	`, cert.ID, cert.Status, certificate, privateKey, caChain,
		cert.IssuedAt, cert.ExpiresAt, cert.RetryCount, lastError,
	)
	return err
}

// ListExpiringCertificates returns IKEv2 certificates that are active and expire within
// the given duration. Includes domain_name via JOIN for display purposes.
func (s *DBStore) ListExpiringCertificates(ctx context.Context, within time.Duration) ([]IKEv2Certificate, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT vc.id, vc.node_id, vc.domain_id, vc.cert_type, vc.status,
		       vc.certificate, vc.private_key, vc.ca_chain,
		       vc.issued_at, vc.expires_at, vc.retry_count, vc.last_error
		FROM vpn_certificates vc
		WHERE vc.cert_type = 'ikev2'
		  AND vc.status = 'active'
		  AND vc.expires_at <= NOW() + $1::interval
		ORDER BY vc.expires_at ASC
	`, within.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var certs []IKEv2Certificate
	for rows.Next() {
		var c IKEv2Certificate
		var certificate, privateKey, caChain, lastError sql.NullString
		var issuedAt, expiresAt sql.NullTime

		if err := rows.Scan(
			&c.ID, &c.NodeID, &c.DomainID, &c.CertType, &c.Status,
			&certificate, &privateKey, &caChain,
			&issuedAt, &expiresAt, &c.RetryCount, &lastError,
		); err != nil {
			return nil, err
		}

		if certificate.Valid {
			c.Certificate = certificate.String
		}
		if privateKey.Valid {
			c.PrivateKey = privateKey.String
		}
		if caChain.Valid {
			c.CAChain = caChain.String
		}
		if issuedAt.Valid {
			c.IssuedAt = &issuedAt.Time
		}
		if expiresAt.Valid {
			c.ExpiresAt = &expiresAt.Time
		}
		if lastError.Valid {
			c.LastError = lastError.String
		}

		certs = append(certs, c)
	}
	return certs, rows.Err()
}
