package api

import (
	"testing"
	"time"
)

func TestCheckMetric_FirstBreach(t *testing.T) {
	// Reset state
	alertStateMu.Lock()
	alertState = make(map[string]time.Time)
	alertStateMu.Unlock()

	var notified bool
	notify := func(msg string) { notified = true }

	now := time.Now()
	checkMetric(1, "test-node", "cpu", 95.0, 80, now, notify)

	if !notified {
		t.Error("expected notification on first breach")
	}

	// Verify state was recorded
	alertStateMu.Lock()
	_, exists := alertState[alertKey(1, "cpu")]
	alertStateMu.Unlock()
	if !exists {
		t.Error("expected alert state to be recorded")
	}
}

func TestCheckMetric_CooldownSuppresses(t *testing.T) {
	alertStateMu.Lock()
	alertState = make(map[string]time.Time)
	alertStateMu.Unlock()

	callCount := 0
	notify := func(msg string) { callCount++ }

	now := time.Now()
	// First breach
	checkMetric(1, "test-node", "cpu", 95.0, 80, now, notify)
	// Second breach within cooldown — should be suppressed
	checkMetric(1, "test-node", "cpu", 92.0, 80, now.Add(5*time.Minute), notify)

	if callCount != 1 {
		t.Errorf("expected 1 notification, got %d", callCount)
	}
}

func TestCheckMetric_CooldownExpired(t *testing.T) {
	alertStateMu.Lock()
	alertState = make(map[string]time.Time)
	alertStateMu.Unlock()

	callCount := 0
	notify := func(msg string) { callCount++ }

	now := time.Now()
	// First breach
	checkMetric(1, "test-node", "cpu", 95.0, 80, now, notify)
	// Second breach after cooldown expired — should notify again
	checkMetric(1, "test-node", "cpu", 92.0, 80, now.Add(16*time.Minute), notify)

	if callCount != 2 {
		t.Errorf("expected 2 notifications after cooldown, got %d", callCount)
	}
}

func TestCheckMetric_Recovery(t *testing.T) {
	alertStateMu.Lock()
	alertState = make(map[string]time.Time)
	alertStateMu.Unlock()

	callCount := 0
	notify := func(msg string) { callCount++ }

	now := time.Now()
	// Breach
	checkMetric(1, "test-node", "ram", 95.0, 90, now, notify)
	// Recovery
	checkMetric(1, "test-node", "ram", 70.0, 90, now.Add(1*time.Minute), notify)

	// Verify state cleared
	alertStateMu.Lock()
	_, exists := alertState[alertKey(1, "ram")]
	alertStateMu.Unlock()
	if exists {
		t.Error("expected alert state to be cleared after recovery")
	}

	// New breach should trigger immediately
	checkMetric(1, "test-node", "ram", 92.0, 90, now.Add(2*time.Minute), notify)
	if callCount != 2 {
		t.Errorf("expected 2 notifications (breach, recovery re-breach), got %d", callCount)
	}
}

func TestCheckMetric_ZeroThresholdSkips(t *testing.T) {
	alertStateMu.Lock()
	alertState = make(map[string]time.Time)
	alertStateMu.Unlock()

	var notified bool
	notify := func(msg string) { notified = true }

	checkMetric(1, "test-node", "disk", 99.0, 0, time.Now(), notify)
	if notified {
		t.Error("should not notify when threshold is 0 (disabled)")
	}
}

func TestCheckMetricCount_ConnectionBreach(t *testing.T) {
	alertStateMu.Lock()
	alertState = make(map[string]time.Time)
	alertStateMu.Unlock()

	var notified bool
	var lastMsg string
	notify := func(msg string) { notified = true; lastMsg = msg }

	now := time.Now()
	checkMetricCount(5, "prod-node", "conn", 600, 500, now, notify)

	if !notified {
		t.Error("expected notification for connection count breach")
	}
	if lastMsg == "" {
		t.Error("expected message to be set")
	}
}

func TestCheckMetricCount_CooldownSuppresses(t *testing.T) {
	alertStateMu.Lock()
	alertState = make(map[string]time.Time)
	alertStateMu.Unlock()

	callCount := 0
	notify := func(msg string) { callCount++ }

	now := time.Now()
	checkMetricCount(5, "prod-node", "conn", 600, 500, now, notify)
	checkMetricCount(5, "prod-node", "conn", 650, 500, now.Add(10*time.Minute), notify)

	if callCount != 1 {
		t.Errorf("expected 1 notification within cooldown, got %d", callCount)
	}
}

func TestAlertKey(t *testing.T) {
	tests := []struct {
		nodeID   int64
		metric   string
		expected string
	}{
		{1, "cpu", "1:cpu"},
		{42, "ram", "42:ram"},
		{100, "conn", "100:conn"},
	}
	for _, tt := range tests {
		got := alertKey(tt.nodeID, tt.metric)
		if got != tt.expected {
			t.Errorf("alertKey(%d, %q) = %q, want %q", tt.nodeID, tt.metric, got, tt.expected)
		}
	}
}
