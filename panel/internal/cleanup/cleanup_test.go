package cleanup

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAllTargets(t *testing.T) {
	targets := AllTargets()

	if len(targets) != 8 {
		t.Fatalf("expected 8 targets, got %d", len(targets))
	}

	expected := map[CleanupTarget]bool{
		TargetStaleSessions:   true,
		TargetExpiredSubs:     true,
		TargetOrphanedRecords: true,
		TargetOldRadAcct:      true,
		TargetOldWalletTxns:   true,
		TargetOldAuditLogs:    true,
		TargetOldEvents:       true,
		TargetOldSnapshots:    true,
	}

	for _, tgt := range targets {
		if !expected[tgt] {
			t.Errorf("unexpected target in AllTargets: %s", tgt)
		}
	}
}

func TestBuildQueries(t *testing.T) {
	tests := []struct {
		name   string
		target CleanupTarget
	}{
		{"stale_sessions", TargetStaleSessions},
		{"expired_subscriptions", TargetExpiredSubs},
		{"orphaned_records", TargetOrphanedRecords},
		{"old_radacct", TargetOldRadAcct},
		{"old_wallet_transactions", TargetOldWalletTxns},
		{"old_audit_logs", TargetOldAuditLogs},
		{"old_events", TargetOldEvents},
		{"old_snapshots", TargetOldSnapshots},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			countQ, deleteQ, dateCol := buildQueries(tc.target)
			if countQ == "" {
				t.Error("countQuery is empty")
			}
			if deleteQ == "" {
				t.Error("deleteQuery is empty")
			}
			if dateCol == "" {
				t.Error("dateColumn is empty")
			}
		})
	}

	t.Run("unsupported_target", func(t *testing.T) {
		countQ, deleteQ, dateCol := buildQueries("nonexistent")
		if countQ != "" || deleteQ != "" || dateCol != "" {
			t.Error("expected empty strings for unsupported target")
		}
	})
}

func TestPreview(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	svc := New(db)
	ctx := context.Background()

	tests := []struct {
		name          string
		targets       []CleanupTarget
		mockCounts    []int64
		expectedCount int
	}{
		{
			name:          "single target",
			targets:       []CleanupTarget{TargetOldAuditLogs},
			mockCounts:    []int64{42},
			expectedCount: 1,
		},
		{
			name:          "multiple targets",
			targets:       []CleanupTarget{TargetOldAuditLogs, TargetOldEvents},
			mockCounts:    []int64{100, 250},
			expectedCount: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for i, count := range tc.mockCounts {
				// Mock the COUNT query
				mock.ExpectQuery("SELECT COUNT").
					WithArgs(sqlmock.AnyArg()).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(count))

				// Mock the oldest row query (MIN date)
				_ = i
				mock.ExpectQuery("SELECT MIN").
					WithArgs(sqlmock.AnyArg()).
					WillReturnRows(sqlmock.NewRows([]string{"oldest"}).AddRow(time.Now().Add(-90 * 24 * time.Hour)))
			}

			req := CleanupRequest{
				Targets:   tc.targets,
				OlderThan: 30 * 24 * time.Hour,
			}

			previews, err := svc.Preview(ctx, req)
			if err != nil {
				t.Fatalf("Preview returned error: %v", err)
			}

			if len(previews) != tc.expectedCount {
				t.Fatalf("expected %d previews, got %d", tc.expectedCount, len(previews))
			}

			for i, p := range previews {
				if p.Target != tc.targets[i] {
					t.Errorf("preview[%d] target = %s, want %s", i, p.Target, tc.targets[i])
				}
				if p.RowCount != tc.mockCounts[i] {
					t.Errorf("preview[%d] row_count = %d, want %d", i, p.RowCount, tc.mockCounts[i])
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet mock expectations: %v", err)
			}
		})
	}
}

func TestPreviewDefaultsToAllTargets(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	svc := New(db)
	ctx := context.Background()

	// When targets is empty, Preview should use all 8 targets
	for i := 0; i < 8; i++ {
		mock.ExpectQuery("SELECT COUNT").
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(i * 10)))
		mock.ExpectQuery("SELECT MIN").
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"oldest"}).AddRow(nil))
	}

	req := CleanupRequest{
		Targets:   nil, // empty → defaults to AllTargets
		OlderThan: 30 * 24 * time.Hour,
	}

	previews, err := svc.Preview(ctx, req)
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}

	if len(previews) != 8 {
		t.Fatalf("expected 8 previews for all targets, got %d", len(previews))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet mock expectations: %v", err)
	}
}

func TestExecuteRejectsShortRetention(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	svc := New(db)
	ctx := context.Background()

	tests := []struct {
		name      string
		olderThan time.Duration
		wantErr   bool
	}{
		{"1 day rejected", 24 * time.Hour, true},
		{"6 days rejected", 6 * 24 * time.Hour, true},
		{"7 days accepted", 7 * 24 * time.Hour, false},
		{"30 days accepted", 30 * 24 * time.Hour, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db2, mock2, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to open sqlmock: %v", err)
			}
			defer db2.Close()

			svc2 := New(db2)

			if !tc.wantErr {
				// For valid requests, mock a single DELETE that returns 0 rows (nothing to delete)
				mock2.ExpectExec("DELETE FROM").
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(0, 0))
			}

			req := CleanupRequest{
				Targets:   []CleanupTarget{TargetOldAuditLogs},
				OlderThan: tc.olderThan,
				BatchSize: 100,
			}

			_, execErr := svc2.Execute(ctx, req)
			if tc.wantErr && execErr == nil {
				t.Error("expected error for short retention, got nil")
			}
			if !tc.wantErr && execErr != nil {
				t.Errorf("unexpected error: %v", execErr)
			}

			_ = svc
		})
	}
}

func TestExecuteMockedBatchDeletes(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	svc := New(db)
	ctx := context.Background()

	// Simulate two batches: first returns batchSize rows, second returns fewer (end)
	batchSize := 100

	// First batch: deletes 100 rows (== batchSize, so loop continues)
	mock.ExpectExec("DELETE FROM").
		WithArgs(sqlmock.AnyArg(), batchSize).
		WillReturnResult(sqlmock.NewResult(0, int64(batchSize)))

	// Second batch: deletes 50 rows (< batchSize, so loop ends)
	mock.ExpectExec("DELETE FROM").
		WithArgs(sqlmock.AnyArg(), batchSize).
		WillReturnResult(sqlmock.NewResult(0, 50))

	req := CleanupRequest{
		Targets:   []CleanupTarget{TargetOldAuditLogs},
		OlderThan: 30 * 24 * time.Hour,
		BatchSize: batchSize,
	}

	results, err := svc.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].RowsAffected != 150 {
		t.Errorf("expected 150 total rows affected, got %d", results[0].RowsAffected)
	}

	if results[0].Target != TargetOldAuditLogs {
		t.Errorf("expected target %s, got %s", TargetOldAuditLogs, results[0].Target)
	}

	if results[0].Error != "" {
		t.Errorf("expected no error, got %s", results[0].Error)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet mock expectations: %v", err)
	}
}

func TestExecuteMultipleTargets(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	svc := New(db)
	ctx := context.Background()

	targets := []CleanupTarget{TargetOldEvents, TargetOldSnapshots}

	// Target 1: one batch of 30 rows
	mock.ExpectExec("DELETE FROM").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 30))

	// Target 2: one batch of 75 rows
	mock.ExpectExec("DELETE FROM").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 75))

	req := CleanupRequest{
		Targets:   targets,
		OlderThan: 14 * 24 * time.Hour,
		BatchSize: 1000,
	}

	results, err := svc.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].RowsAffected != 30 {
		t.Errorf("target[0] rows = %d, want 30", results[0].RowsAffected)
	}
	if results[1].RowsAffected != 75 {
		t.Errorf("target[1] rows = %d, want 75", results[1].RowsAffected)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet mock expectations: %v", err)
	}
}

func TestExecuteNotifyCallback(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	svc := New(db)
	ctx := context.Background()

	var events []CleanupEvent
	svc.SetNotify(func(e CleanupEvent) {
		events = append(events, e)
	})

	// Single target, single batch of 25 rows
	mock.ExpectExec("DELETE FROM").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 25))

	req := CleanupRequest{
		Targets:   []CleanupTarget{TargetOldWalletTxns},
		OlderThan: 90 * 24 * time.Hour,
		BatchSize: 1000,
	}

	_, err = svc.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	// Should receive at least a progress event and a complete event
	if len(events) == 0 {
		t.Fatal("expected notify events, got none")
	}

	// The last event for the target should be "complete"
	var hasComplete bool
	var hasProgress bool
	for _, e := range events {
		if e.Type == "complete" && e.Target == TargetOldWalletTxns {
			hasComplete = true
			if e.Deleted != 25 {
				t.Errorf("complete event deleted = %d, want 25", e.Deleted)
			}
		}
		if e.Type == "progress" && e.Target == TargetOldWalletTxns {
			hasProgress = true
		}
	}

	if !hasProgress {
		t.Error("expected at least one 'progress' event")
	}
	if !hasComplete {
		t.Error("expected a 'complete' event")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet mock expectations: %v", err)
	}
}

func TestExecuteUsesDefaultBatchSize(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	svc := New(db)
	ctx := context.Background()

	// When BatchSize is 0, it should default to service's batchSize (1000)
	mock.ExpectExec("DELETE FROM").
		WithArgs(sqlmock.AnyArg(), 1000).
		WillReturnResult(sqlmock.NewResult(0, 10))

	req := CleanupRequest{
		Targets:   []CleanupTarget{TargetOldEvents},
		OlderThan: 30 * 24 * time.Hour,
		BatchSize: 0, // should use default
	}

	results, err := svc.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if results[0].RowsAffected != 10 {
		t.Errorf("expected 10 rows, got %d", results[0].RowsAffected)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet mock expectations: %v", err)
	}
}
