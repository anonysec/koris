package sessions

import (
	"database/sql"
	"log"
	"time"
)

type Enforcer struct {
	db   *sql.DB
	done chan struct{}
}

func NewEnforcer(db *sql.DB) *Enforcer {
	return &Enforcer{
		db:   db,
		done: make(chan struct{}),
	}
}

// EnforceConnLimit kills excess active sessions for users who exceed their connection limit.
// Called periodically by the worker goroutine.
func (e *Enforcer) EnforceConnLimit() {
	// Get users with Simultaneous-Use > 0 who have more active sessions than allowed
	rows, err := e.db.Query(`
		SELECT r.username, CAST(r.value AS BIGINT) AS conn_limit,
			(SELECT COUNT(*) FROM radacct WHERE username=r.username AND acctstoptime IS NULL) AS active
		FROM radcheck r
		JOIN customers c ON c.username = r.username AND c.status = 'active' AND c.deleted_at IS NULL
		WHERE r.attribute = 'Simultaneous-Use' AND CAST(r.value AS BIGINT) > 0
		HAVING active > conn_limit
	`)
	if err != nil {
		log.Printf("[enforcer] query: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var username string
		var limit, active int
		if err := rows.Scan(&username, &limit, &active); err != nil {
			continue
		}
		excess := active - limit
		if excess <= 0 {
			continue
		}
		// Kill oldest excess sessions (PostgreSQL doesn't support LIMIT on UPDATE)
		_, err := e.db.Exec(`
			UPDATE radacct SET acctstoptime=NOW(), acctterminatecause='Connection-Limit'
			WHERE ctid IN (SELECT ctid FROM radacct WHERE username=$1 AND acctstoptime IS NULL
			ORDER BY acctstarttime ASC LIMIT $2)
		`, username, excess)
		if err != nil {
			log.Printf("[enforcer] kill sessions for %s: %v", username, err)
		} else {
			log.Printf("[enforcer] killed %d excess sessions for %s (limit=%d)", excess, username, limit)
		}
	}
}

// Start runs the enforcer every 30 seconds.
// The goroutine exits when Stop() is called.
func (e *Enforcer) Start() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		// Run immediately on start
		e.EnforceConnLimit()
		for {
			select {
			case <-ticker.C:
				e.EnforceConnLimit()
			case <-e.done:
				return
			}
		}
	}()
}

// Stop signals the enforcer goroutine to exit.
func (e *Enforcer) Stop() {
	close(e.done)
}
