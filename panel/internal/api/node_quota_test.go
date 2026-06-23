package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestUpdateBandwidthQuota_ZeroBytes(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// Zero bytes should be a no-op (no DB call)
	UpdateBandwidthQuota(db, 1, 0, 0)
}

func TestUpdateBandwidthQuota_NegativeBytes(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// Negative total should be a no-op
	UpdateBandwidthQuota(db, 1, -100, 0)
}

func TestUpdateBandwidthQuota_AddsUsage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// 1 GB = 1073741824 bytes
	rxBytes := int64(536870912) // 0.5 GB
	txBytes := int64(536870912) // 0.5 GB
	expectedDelta := float64(rxBytes+txBytes) / (1024 * 1024 * 1024)

	mock.ExpectExec("UPDATE node_bandwidth_quotas SET current_usage_gb").
		WithArgs(expectedDelta, int64(5)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	UpdateBandwidthQuota(db, 5, rxBytes, txBytes)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestCheckBandwidthQuotas_SoftLimitAlert(t *testing.T) {
	// Reset global alert state
	bwAlertLevelMu.Lock()
	bwAlertLevel = make(map[int64]int)
	bwAlertLevelMu.Unlock()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// Node at 85% usage with 80% threshold
	rows := sqlmock.NewRows([]string{"node_id", "name", "monthly_limit_gb", "current_usage_gb", "alert_threshold_pct"}).
		AddRow(1, "US-East", 1000, 850.0, 80)
	mock.ExpectQuery("SELECT q.node_id, n.name, q.monthly_limit_gb, q.current_usage_gb, q.alert_threshold_pct").
		WillReturnRows(rows)

	var messages []string
	notify := func(msg string) { messages = append(messages, msg) }

	CheckBandwidthQuotas(db, notify)

	if len(messages) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(messages))
	}
	// Should be a warning (soft limit)
	if messages[0] == "" {
		t.Error("expected non-empty notification message")
	}

	// Verify alert level set to 1
	bwAlertLevelMu.Lock()
	level := bwAlertLevel[1]
	bwAlertLevelMu.Unlock()
	if level != 1 {
		t.Errorf("alert level = %d, want 1", level)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestCheckBandwidthQuotas_HardLimitAlert(t *testing.T) {
	bwAlertLevelMu.Lock()
	bwAlertLevel = make(map[int64]int)
	bwAlertLevelMu.Unlock()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// Node at 105% usage
	rows := sqlmock.NewRows([]string{"node_id", "name", "monthly_limit_gb", "current_usage_gb", "alert_threshold_pct"}).
		AddRow(2, "EU-West", 500, 525.0, 80)
	mock.ExpectQuery("SELECT q.node_id, n.name, q.monthly_limit_gb, q.current_usage_gb, q.alert_threshold_pct").
		WillReturnRows(rows)

	var messages []string
	notify := func(msg string) { messages = append(messages, msg) }

	CheckBandwidthQuotas(db, notify)

	if len(messages) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(messages))
	}

	// Verify alert level set to 2 (hard limit)
	bwAlertLevelMu.Lock()
	level := bwAlertLevel[2]
	bwAlertLevelMu.Unlock()
	if level != 2 {
		t.Errorf("alert level = %d, want 2", level)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestCheckBandwidthQuotas_NoDoubleAlert(t *testing.T) {
	bwAlertLevelMu.Lock()
	bwAlertLevel = make(map[int64]int)
	bwAlertLevelMu.Unlock()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// First check — triggers soft limit
	rows1 := sqlmock.NewRows([]string{"node_id", "name", "monthly_limit_gb", "current_usage_gb", "alert_threshold_pct"}).
		AddRow(1, "US-East", 1000, 850.0, 80)
	mock.ExpectQuery("SELECT q.node_id").WillReturnRows(rows1)

	// Second check — same usage, should NOT alert again
	rows2 := sqlmock.NewRows([]string{"node_id", "name", "monthly_limit_gb", "current_usage_gb", "alert_threshold_pct"}).
		AddRow(1, "US-East", 1000, 870.0, 80)
	mock.ExpectQuery("SELECT q.node_id").WillReturnRows(rows2)

	callCount := 0
	notify := func(msg string) { callCount++ }

	CheckBandwidthQuotas(db, notify)
	CheckBandwidthQuotas(db, notify)

	if callCount != 1 {
		t.Errorf("expected 1 notification (no double alert), got %d", callCount)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestCheckBandwidthQuotas_SoftThenHardEscalation(t *testing.T) {
	bwAlertLevelMu.Lock()
	bwAlertLevel = make(map[int64]int)
	bwAlertLevelMu.Unlock()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// First check — at 85%, triggers soft limit
	rows1 := sqlmock.NewRows([]string{"node_id", "name", "monthly_limit_gb", "current_usage_gb", "alert_threshold_pct"}).
		AddRow(1, "US-East", 1000, 850.0, 80)
	mock.ExpectQuery("SELECT q.node_id").WillReturnRows(rows1)

	// Second check — at 100%, triggers hard limit
	rows2 := sqlmock.NewRows([]string{"node_id", "name", "monthly_limit_gb", "current_usage_gb", "alert_threshold_pct"}).
		AddRow(1, "US-East", 1000, 1050.0, 80)
	mock.ExpectQuery("SELECT q.node_id").WillReturnRows(rows2)

	callCount := 0
	notify := func(msg string) { callCount++ }

	CheckBandwidthQuotas(db, notify)
	CheckBandwidthQuotas(db, notify)

	if callCount != 2 {
		t.Errorf("expected 2 notifications (soft + hard), got %d", callCount)
	}

	bwAlertLevelMu.Lock()
	level := bwAlertLevel[1]
	bwAlertLevelMu.Unlock()
	if level != 2 {
		t.Errorf("alert level = %d, want 2 (hard)", level)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestCheckBandwidthQuotas_RecoveryClears(t *testing.T) {
	bwAlertLevelMu.Lock()
	bwAlertLevel = make(map[int64]int)
	bwAlertLevelMu.Unlock()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// First check — triggers soft limit at 85%
	rows1 := sqlmock.NewRows([]string{"node_id", "name", "monthly_limit_gb", "current_usage_gb", "alert_threshold_pct"}).
		AddRow(1, "US-East", 1000, 850.0, 80)
	mock.ExpectQuery("SELECT q.node_id").WillReturnRows(rows1)

	// Second check — recovered below threshold (50%)
	rows2 := sqlmock.NewRows([]string{"node_id", "name", "monthly_limit_gb", "current_usage_gb", "alert_threshold_pct"}).
		AddRow(1, "US-East", 1000, 500.0, 80)
	mock.ExpectQuery("SELECT q.node_id").WillReturnRows(rows2)

	// Third check — breaches threshold again, should alert
	rows3 := sqlmock.NewRows([]string{"node_id", "name", "monthly_limit_gb", "current_usage_gb", "alert_threshold_pct"}).
		AddRow(1, "US-East", 1000, 900.0, 80)
	mock.ExpectQuery("SELECT q.node_id").WillReturnRows(rows3)

	callCount := 0
	notify := func(msg string) { callCount++ }

	CheckBandwidthQuotas(db, notify)
	CheckBandwidthQuotas(db, notify)
	CheckBandwidthQuotas(db, notify)

	if callCount != 2 {
		t.Errorf("expected 2 notifications (breach, recovery, re-breach), got %d", callCount)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestCheckBandwidthQuotas_DefaultThreshold(t *testing.T) {
	bwAlertLevelMu.Lock()
	bwAlertLevel = make(map[int64]int)
	bwAlertLevelMu.Unlock()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// threshold_pct = 0 → should default to 80%
	// Usage at 85% should trigger
	rows := sqlmock.NewRows([]string{"node_id", "name", "monthly_limit_gb", "current_usage_gb", "alert_threshold_pct"}).
		AddRow(1, "US-East", 100, 85.0, 0)
	mock.ExpectQuery("SELECT q.node_id").WillReturnRows(rows)

	var notified bool
	notify := func(msg string) { notified = true }

	CheckBandwidthQuotas(db, notify)

	if !notified {
		t.Error("expected notification with default 80% threshold")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestCheckBandwidthQuotas_BelowThreshold_NoAlert(t *testing.T) {
	bwAlertLevelMu.Lock()
	bwAlertLevel = make(map[int64]int)
	bwAlertLevelMu.Unlock()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// Usage at 50% with 80% threshold — no alert
	rows := sqlmock.NewRows([]string{"node_id", "name", "monthly_limit_gb", "current_usage_gb", "alert_threshold_pct"}).
		AddRow(1, "US-East", 1000, 500.0, 80)
	mock.ExpectQuery("SELECT q.node_id").WillReturnRows(rows)

	var notified bool
	notify := func(msg string) { notified = true }

	CheckBandwidthQuotas(db, notify)

	if notified {
		t.Error("should not notify when below threshold")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestCheckBandwidthQuotas_MultipleNodes(t *testing.T) {
	bwAlertLevelMu.Lock()
	bwAlertLevel = make(map[int64]int)
	bwAlertLevelMu.Unlock()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"node_id", "name", "monthly_limit_gb", "current_usage_gb", "alert_threshold_pct"}).
		AddRow(1, "US-East", 1000, 900.0, 80). // soft limit
		AddRow(2, "EU-West", 500, 510.0, 80).  // hard limit
		AddRow(3, "Asia-1", 2000, 500.0, 80)   // below threshold
	mock.ExpectQuery("SELECT q.node_id").WillReturnRows(rows)

	callCount := 0
	notify := func(msg string) { callCount++ }

	CheckBandwidthQuotas(db, notify)

	// Node 1: soft, Node 2: hard, Node 3: no alert
	if callCount != 2 {
		t.Errorf("expected 2 notifications, got %d", callCount)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestResetBandwidthQuotas_ResetsMatchingDay(t *testing.T) {
	bwAlertLevelMu.Lock()
	bwAlertLevel = make(map[int64]int)
	// Pre-set alert levels that should be cleared
	bwAlertLevel[10] = 2
	bwAlertLevel[20] = 1
	bwAlertLevelMu.Unlock()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	today := time.Now().Day()

	// Reset query affects 2 rows
	mock.ExpectExec("UPDATE node_bandwidth_quotas SET current_usage_gb = 0").
		WithArgs(today).
		WillReturnResult(sqlmock.NewResult(0, 2))

	// Query for node IDs to clear alert state
	resetRows := sqlmock.NewRows([]string{"node_id"}).
		AddRow(10).
		AddRow(20)
	mock.ExpectQuery("SELECT node_id FROM node_bandwidth_quotas WHERE reset_day").
		WithArgs(today).
		WillReturnRows(resetRows)

	ResetBandwidthQuotas(db)

	// Verify alert state cleared
	bwAlertLevelMu.Lock()
	level10 := bwAlertLevel[10]
	level20 := bwAlertLevel[20]
	bwAlertLevelMu.Unlock()

	if level10 != 0 {
		t.Errorf("alert level for node 10 = %d, want 0 (cleared)", level10)
	}
	if level20 != 0 {
		t.Errorf("alert level for node 20 = %d, want 0 (cleared)", level20)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestResetBandwidthQuotas_NoMatchingDay(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	today := time.Now().Day()

	// No rows affected — no reset needed
	mock.ExpectExec("UPDATE node_bandwidth_quotas SET current_usage_gb = 0").
		WithArgs(today).
		WillReturnResult(sqlmock.NewResult(0, 0))

	ResetBandwidthQuotas(db)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestNodeQuota_GetNotConfigured(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &Server{DB: db}

	// No quota row exists
	mock.ExpectQuery("SELECT monthly_limit_gb, current_usage_gb, alert_threshold_pct, reset_day").
		WithArgs(int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"monthly_limit_gb", "current_usage_gb", "alert_threshold_pct", "reset_day"}))

	w := newTestRecorder()
	s.getNodeQuota(w, 5)

	resp := decodeJSON(t, w)
	if resp["ok"] != true {
		t.Errorf("ok = %v, want true", resp["ok"])
	}
	if resp["configured"] != false {
		t.Errorf("configured = %v, want false", resp["configured"])
	}
	if resp["monthly_limit_gb"].(float64) != 0 {
		t.Errorf("monthly_limit_gb = %v, want 0", resp["monthly_limit_gb"])
	}
}

func TestNodeQuota_GetConfigured(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &Server{DB: db}

	mock.ExpectQuery("SELECT monthly_limit_gb, current_usage_gb, alert_threshold_pct, reset_day").
		WithArgs(int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"monthly_limit_gb", "current_usage_gb", "alert_threshold_pct", "reset_day"}).
			AddRow(1000, 250.5, 80, 1))

	w := newTestRecorder()
	s.getNodeQuota(w, 5)

	resp := decodeJSON(t, w)
	if resp["ok"] != true {
		t.Errorf("ok = %v, want true", resp["ok"])
	}
	if resp["configured"] != true {
		t.Errorf("configured = %v, want true", resp["configured"])
	}
	if resp["monthly_limit_gb"].(float64) != 1000 {
		t.Errorf("monthly_limit_gb = %v, want 1000", resp["monthly_limit_gb"])
	}
	if resp["current_usage_gb"].(float64) != 250.5 {
		t.Errorf("current_usage_gb = %v, want 250.5", resp["current_usage_gb"])
	}
	// usage_percent = 250.5 / 1000 * 100 = 25.05
	usagePct := resp["usage_percent"].(float64)
	if usagePct < 25.04 || usagePct > 25.06 {
		t.Errorf("usage_percent = %v, want ~25.05", usagePct)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestNodeQuota_SetQuota(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantCode  int
		wantError string
	}{
		{
			name:     "valid_quota",
			body:     `{"monthly_limit_gb": 500, "alert_threshold_pct": 90, "reset_day": 15}`,
			wantCode: 200,
		},
		{
			name:      "negative_limit",
			body:      `{"monthly_limit_gb": -1}`,
			wantCode:  400,
			wantError: "invalid_monthly_limit",
		},
		{
			name:      "threshold_over_100",
			body:      `{"alert_threshold_pct": 101}`,
			wantCode:  400,
			wantError: "invalid_threshold",
		},
		{
			name:      "threshold_negative",
			body:      `{"alert_threshold_pct": -5}`,
			wantCode:  400,
			wantError: "invalid_threshold",
		},
		{
			name:      "reset_day_zero",
			body:      `{"reset_day": 0}`,
			wantCode:  400,
			wantError: "invalid_reset_day",
		},
		{
			name:      "reset_day_over_28",
			body:      `{"reset_day": 29}`,
			wantCode:  400,
			wantError: "invalid_reset_day",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create sqlmock: %v", err)
			}
			defer db.Close()

			s := &Server{DB: db}

			if tt.wantCode == 200 {
				mock.ExpectExec("INSERT INTO node_bandwidth_quotas").
					WillReturnResult(sqlmock.NewResult(1, 1))
			}

			req := newPostRequest("/api/admin/nodes/5/quota", tt.body)
			rec := newTestRecorder()
			s.setNodeQuota(rec, req, 5)

			if rec.Code != tt.wantCode {
				t.Errorf("status = %d, want %d, body: %s", rec.Code, tt.wantCode, rec.Body.String())
			}
			if tt.wantError != "" {
				resp := decodeJSON(t, rec)
				if resp["error"] != tt.wantError {
					t.Errorf("error = %v, want %v", resp["error"], tt.wantError)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// --- test helpers ---

func newTestRecorder() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}

func newPostRequest(url, body string) *http.Request {
	return httptest.NewRequest(http.MethodPost, url, strings.NewReader(body))
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode JSON: %v (body: %s)", err, rec.Body.String())
	}
	return resp
}
