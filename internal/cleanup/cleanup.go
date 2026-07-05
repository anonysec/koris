// Package cleanup provides batch data cleanup operations for purging old
// sessions, expired subscriptions, orphaned records, and log rotation.
package cleanup

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// CleanupTarget identifies a category of data that can be cleaned up.
type CleanupTarget string

const (
	TargetStaleSessions   CleanupTarget = "stale_sessions"
	TargetExpiredSubs     CleanupTarget = "expired_subscriptions"
	TargetOrphanedRecords CleanupTarget = "orphaned_records"
	TargetOldRadAcct      CleanupTarget = "old_radacct"
	TargetOldWalletTxns   CleanupTarget = "old_wallet_transactions"
	TargetOldAuditLogs    CleanupTarget = "old_audit_logs"
	TargetOldEvents       CleanupTarget = "old_events"
	TargetOldSnapshots    CleanupTarget = "old_snapshots"
)

// AllTargets returns all supported cleanup targets.
func AllTargets() []CleanupTarget {
	return []CleanupTarget{
		TargetStaleSessions,
		TargetExpiredSubs,
		TargetOrphanedRecords,
		TargetOldRadAcct,
		TargetOldWalletTxns,
		TargetOldAuditLogs,
		TargetOldEvents,
		TargetOldSnapshots,
	}
}

// CleanupRequest defines what to clean and how.
type CleanupRequest struct {
	Targets   []CleanupTarget
	OlderThan time.Duration // e.g., 90 days
	DryRun    bool
	BatchSize int // default 1000
}

// CleanupResult holds the outcome for a single target.
type CleanupResult struct {
	Target       CleanupTarget `json:"target"`
	RowsAffected int64         `json:"rows_affected"`
	Duration     time.Duration `json:"duration"`
	Error        string        `json:"error,omitempty"`
}

// CleanupPreview holds the preview data for a single target (dry-run).
type CleanupPreview struct {
	Target    CleanupTarget `json:"target"`
	RowCount  int64         `json:"row_count"`
	OldestRow time.Time     `json:"oldest_row"`
}

// CleanupEvent is emitted during cleanup for real-time progress reporting.
type CleanupEvent struct {
	Type     string        `json:"type"` // "progress", "complete", "error"
	Target   CleanupTarget `json:"target"`
	Progress float64       `json:"progress"` // 0.0 - 1.0
	Deleted  int64         `json:"deleted"`
	Message  string        `json:"message"`
}

// AutoCleanupConfig defines the schedule for automatic cleanup.
type AutoCleanupConfig struct {
	Enabled       bool                     `json:"enabled"`
	RunAt         string                   `json:"run_at"` // HH:MM format (e.g. "03:00")
	RetentionDays map[CleanupTarget]int    `json:"retention_days"`
}

// Service provides batch data cleanup operations.
type Service struct {
	db        *sql.DB
	batchSize int
	notify    func(event CleanupEvent)
}

// New creates a new CleanupService.
func New(db *sql.DB) *Service {
	return &Service{
		db:        db,
		batchSize: 1000,
	}
}

// SetNotify sets the callback for progress events.
func (s *Service) SetNotify(fn func(CleanupEvent)) {
	s.notify = fn
}

// Preview returns row counts and oldest records for the given targets
// without deleting anything.
func (s *Service) Preview(ctx context.Context, req CleanupRequest) ([]CleanupPreview, error) {
	targets := req.Targets
	if len(targets) == 0 {
		targets = AllTargets()
	}

	cutoff := time.Now().Add(-req.OlderThan)
	previews := make([]CleanupPreview, 0, len(targets))

	for _, target := range targets {
		countQuery, _, dateCol := buildQueries(target)
		if countQuery == "" {
			continue
		}

		var count int64
		err := s.db.QueryRowContext(ctx, countQuery, cutoff).Scan(&count)
		if err != nil {
			log.Printf("[cleanup] preview count error for %s: %v", target, err)
			count = 0
		}

		var oldest sql.NullTime
		oldestQuery := fmt.Sprintf("SELECT MIN(%s) FROM (%s) AS sub",
			dateCol, buildSubQuery(target, cutoff))
		_ = s.db.QueryRowContext(ctx, oldestQuery, cutoff).Scan(&oldest)

		preview := CleanupPreview{
			Target:   target,
			RowCount: count,
		}
		if oldest.Valid {
			preview.OldestRow = oldest.Time
		}
		previews = append(previews, preview)
	}

	return previews, nil
}

// Execute performs the actual cleanup in batches.
func (s *Service) Execute(ctx context.Context, req CleanupRequest) ([]CleanupResult, error) {
	if req.OlderThan < 7*24*time.Hour {
		return nil, fmt.Errorf("minimum retention period is 7 days")
	}

	targets := req.Targets
	if len(targets) == 0 {
		targets = AllTargets()
	}

	batchSize := req.BatchSize
	if batchSize <= 0 {
		batchSize = s.batchSize
	}

	cutoff := time.Now().Add(-req.OlderThan)
	results := make([]CleanupResult, 0, len(targets))

	for _, target := range targets {
		start := time.Now()
		deleted, err := s.deleteInBatches(ctx, target, cutoff, batchSize)

		result := CleanupResult{
			Target:       target,
			RowsAffected: deleted,
			Duration:     time.Since(start),
		}
		if err != nil {
			result.Error = err.Error()
		}
		results = append(results, result)

		if s.notify != nil {
			s.notify(CleanupEvent{
				Type:    "complete",
				Target:  target,
				Deleted: deleted,
				Message: fmt.Sprintf("deleted %d rows", deleted),
			})
		}
	}

	return results, nil
}

// deleteInBatches deletes rows for a target in batches of batchSize.
func (s *Service) deleteInBatches(ctx context.Context, target CleanupTarget, cutoff time.Time, batchSize int) (int64, error) {
	_, deleteQuery, _ := buildQueries(target)
	if deleteQuery == "" {
		return 0, fmt.Errorf("unsupported target: %s", target)
	}

	var totalDeleted int64

	for {
		select {
		case <-ctx.Done():
			return totalDeleted, ctx.Err()
		default:
		}

		result, err := s.db.ExecContext(ctx, deleteQuery, cutoff, batchSize)
		if err != nil {
			return totalDeleted, fmt.Errorf("delete batch for %s: %w", target, err)
		}

		affected, _ := result.RowsAffected()
		totalDeleted += affected

		if s.notify != nil && totalDeleted > 0 {
			s.notify(CleanupEvent{
				Type:    "progress",
				Target:  target,
				Deleted: totalDeleted,
			})
		}

		if affected < int64(batchSize) {
			break // No more rows to delete
		}

		// Pause between batches to avoid long-running locks
		time.Sleep(100 * time.Millisecond)
	}

	return totalDeleted, nil
}

// buildQueries returns (countQuery, deleteQuery, dateColumn) for a target.
func buildQueries(target CleanupTarget) (string, string, string) {
	switch target {
	case TargetStaleSessions:
		return "SELECT COUNT(*) FROM radacct WHERE acctstoptime IS NOT NULL AND acctstoptime < ?",
			"DELETE FROM radacct WHERE acctstoptime IS NOT NULL AND acctstoptime < ? LIMIT ?",
			"acctstoptime"
	case TargetExpiredSubs:
		return "SELECT COUNT(*) FROM customers WHERE status='expired' AND updated_at < ?",
			"DELETE FROM customers WHERE status='expired' AND updated_at < ? AND deleted_at IS NOT NULL LIMIT ?",
			"updated_at"
	case TargetOrphanedRecords:
		return "SELECT COUNT(*) FROM radacct WHERE username NOT IN (SELECT username FROM customers WHERE deleted_at IS NULL) AND acctstarttime < ?",
			"DELETE FROM radacct WHERE username NOT IN (SELECT username FROM customers WHERE deleted_at IS NULL) AND acctstarttime < ? LIMIT ?",
			"acctstarttime"
	case TargetOldRadAcct:
		return "SELECT COUNT(*) FROM radacct WHERE acctstoptime IS NOT NULL AND acctstoptime < ?",
			"DELETE FROM radacct WHERE acctstoptime IS NOT NULL AND acctstoptime < ? LIMIT ?",
			"acctstoptime"
	case TargetOldWalletTxns:
		return "SELECT COUNT(*) FROM wallet_transactions WHERE created_at < ?",
			"DELETE FROM wallet_transactions WHERE created_at < ? LIMIT ?",
			"created_at"
	case TargetOldAuditLogs:
		return "SELECT COUNT(*) FROM audit_logs WHERE created_at < ?",
			"DELETE FROM audit_logs WHERE created_at < ? LIMIT ?",
			"created_at"
	case TargetOldEvents:
		return "SELECT COUNT(*) FROM events WHERE created_at < ?",
			"DELETE FROM events WHERE created_at < ? LIMIT ?",
			"created_at"
	case TargetOldSnapshots:
		return "SELECT COUNT(*) FROM node_usage_snapshots WHERE created_at < ?",
			"DELETE FROM node_usage_snapshots WHERE created_at < ? LIMIT ?",
			"created_at"
	default:
		return "", "", ""
	}
}

// buildSubQuery returns a SELECT query for oldest date calculation.
func buildSubQuery(target CleanupTarget, cutoff time.Time) string {
	countQuery, _, dateCol := buildQueries(target)
	if countQuery == "" {
		return "SELECT NOW() AS dt"
	}
	// Convert COUNT query to a SELECT of the date column
	// This is approximate — we just need the oldest row
	switch target {
	case TargetStaleSessions, TargetOldRadAcct:
		return fmt.Sprintf("SELECT %s FROM radacct WHERE acctstoptime IS NOT NULL AND acctstoptime < ? ORDER BY %s ASC LIMIT 1", dateCol, dateCol)
	case TargetExpiredSubs:
		return fmt.Sprintf("SELECT %s FROM customers WHERE status='expired' AND updated_at < ? ORDER BY %s ASC LIMIT 1", dateCol, dateCol)
	case TargetOrphanedRecords:
		return fmt.Sprintf("SELECT %s FROM radacct WHERE acctstarttime < ? ORDER BY %s ASC LIMIT 1", dateCol, dateCol)
	case TargetOldWalletTxns:
		return fmt.Sprintf("SELECT %s FROM wallet_transactions WHERE created_at < ? ORDER BY %s ASC LIMIT 1", dateCol, dateCol)
	case TargetOldAuditLogs:
		return fmt.Sprintf("SELECT %s FROM audit_logs WHERE created_at < ? ORDER BY %s ASC LIMIT 1", dateCol, dateCol)
	case TargetOldEvents:
		return fmt.Sprintf("SELECT %s FROM events WHERE created_at < ? ORDER BY %s ASC LIMIT 1", dateCol, dateCol)
	case TargetOldSnapshots:
		return fmt.Sprintf("SELECT %s FROM node_usage_snapshots WHERE created_at < ? ORDER BY %s ASC LIMIT 1", dateCol, dateCol)
	default:
		return "SELECT NOW() AS dt"
	}
}
