package certrotation

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// Worker monitors certificate expiry and triggers rotation.
type Worker struct {
	DB            *sql.DB
	CheckInterval time.Duration
	WarnBefore    time.Duration
	RenewBefore   time.Duration
}

// CertInfo holds certificate metadata for rotation checks.
type CertInfo struct {
	ID        int64
	Name      string
	Type      string
	NodeID    *int64
	ExpiresAt time.Time
}

// New creates a new certificate rotation worker with sensible defaults.
func New(db *sql.DB) *Worker {
	return &Worker{
		DB:            db,
		CheckInterval: 1 * time.Hour,
		WarnBefore:    30 * 24 * time.Hour,
		RenewBefore:   7 * 24 * time.Hour,
	}
}

// Start launches the background worker goroutine.
func (w *Worker) Start() {
	go func() {
		log.Println("[certrotation] worker started (interval:", w.CheckInterval, ")")
		ticker := time.NewTicker(w.CheckInterval)
		defer ticker.Stop()
		// Run once on startup
		w.check()
		for range ticker.C {
			w.check()
		}
	}()
}

func (w *Worker) check() {
	expiring, err := w.CheckExpiring()
	if err != nil {
		log.Printf("[certrotation] check error: %v", err)
		return
	}
	for _, cert := range expiring {
		daysLeft := time.Until(cert.ExpiresAt).Hours() / 24
		if daysLeft <= 7 {
			log.Printf("[certrotation] CRITICAL: cert %q (id=%d) expires in %.0f days — triggering regeneration", cert.Name, cert.ID, daysLeft)
			w.createEvent(cert, "critical", daysLeft)
			// Regeneration would call openssl/easy-rsa here
			// For now, log and create event
		} else if daysLeft <= 30 {
			log.Printf("[certrotation] WARNING: cert %q (id=%d) expires in %.0f days", cert.Name, cert.ID, daysLeft)
			w.createEvent(cert, "warning", daysLeft)
		}
	}
}

// CheckExpiring returns certificates that will expire within WarnBefore duration.
func (w *Worker) CheckExpiring() ([]CertInfo, error) {
	rows, err := w.DB.Query(`SELECT id, name, type, node_id, expires_at FROM vpn_certificates WHERE expires_at IS NOT NULL AND expires_at <= DATE_ADD(NOW(), INTERVAL ? DAY) ORDER BY expires_at ASC`, int(w.WarnBefore.Hours()/24))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var certs []CertInfo
	for rows.Next() {
		var c CertInfo
		var nodeID sql.NullInt64
		if err := rows.Scan(&c.ID, &c.Name, &c.Type, &nodeID, &c.ExpiresAt); err != nil {
			continue
		}
		if nodeID.Valid {
			c.NodeID = &nodeID.Int64
		}
		certs = append(certs, c)
	}
	return certs, rows.Err()
}

func (w *Worker) createEvent(cert CertInfo, severity string, daysLeft float64) {
	desc := fmt.Sprintf("Certificate %q (type: %s) expires in %.0f days", cert.Name, cert.Type, daysLeft)
	_, _ = w.DB.Exec(`INSERT INTO events(type, severity, title, description, actor, target) VALUES('certificate', ?, ?, ?, 'system', ?)`,
		severity,
		"Certificate expiring: "+cert.Name,
		desc,
		cert.Name,
	)
}
