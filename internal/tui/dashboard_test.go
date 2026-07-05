package tui

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestNewDashboard(t *testing.T) {
	rb := newRingBuffer[LogEntry](100)
	var buf bytes.Buffer

	d := NewDashboard(&buf, 500*time.Millisecond, rb)
	if d == nil {
		t.Fatal("NewDashboard returned nil")
	}
	if !d.enabled {
		t.Error("expected dashboard to be enabled")
	}
	if d.refresh != 500*time.Millisecond {
		t.Errorf("expected refresh=500ms, got %v", d.refresh)
	}
	if len(d.sections) != 2 {
		t.Errorf("expected 2 sections, got %d", len(d.sections))
	}
	if d.width != 80 || d.height != 24 {
		t.Errorf("expected default 80x24, got %dx%d", d.width, d.height)
	}
}

func TestNewDashboard_Defaults(t *testing.T) {
	rb := newRingBuffer[LogEntry](100)

	// nil output should default to os.Stdout
	d := NewDashboard(nil, 0, rb)
	if d == nil {
		t.Fatal("NewDashboard returned nil")
	}
	if d.refresh != defaultRefreshInterval {
		t.Errorf("expected default refresh interval, got %v", d.refresh)
	}
}

func TestDashboard_Render(t *testing.T) {
	rb := newRingBuffer[LogEntry](100)
	var buf bytes.Buffer

	// Add some entries to the ring buffer
	rb.Push(LogEntry{
		Time:      time.Now().UTC(),
		Level:     LevelInfo,
		Component: "api",
		Message:   "server started",
	})
	rb.Push(LogEntry{
		Time:      time.Now().UTC(),
		Level:     LevelWarn,
		Component: "db",
		Message:   "slow query detected",
	})

	d := NewDashboard(&buf, time.Second, rb)
	d.render()

	output := buf.String()
	// Should contain the clear screen sequence
	if !strings.Contains(output, "\033[2J") {
		t.Error("expected clear screen escape sequence in output")
	}
	// Should contain cursor positioning
	if !strings.Contains(output, "\033[") {
		t.Error("expected cursor positioning escape sequences")
	}
	// Should contain metrics panel border
	if !strings.Contains(output, "KorisPanel Dashboard") {
		t.Error("expected dashboard title in output")
	}
}

func TestDashboard_Stop(t *testing.T) {
	rb := newRingBuffer[LogEntry](100)
	var buf bytes.Buffer

	d := NewDashboard(&buf, time.Second, rb)

	// Calling Stop without Start should not panic
	d.Stop()

	output := buf.String()
	// Should show cursor and exit alt screen
	if !strings.Contains(output, "\033[?25h") {
		t.Error("expected show cursor escape sequence")
	}
	if !strings.Contains(output, "\033[?1049l") {
		t.Error("expected exit alt screen escape sequence")
	}
}

func TestTerminalControl_EnterExitAltScreen(t *testing.T) {
	var buf bytes.Buffer

	enterAltScreen(&buf)
	if buf.String() != "\033[?1049h" {
		t.Errorf("enterAltScreen: got %q", buf.String())
	}

	buf.Reset()
	exitAltScreen(&buf)
	if buf.String() != "\033[?1049l" {
		t.Errorf("exitAltScreen: got %q", buf.String())
	}
}

func TestTerminalControl_CursorAndScreen(t *testing.T) {
	var buf bytes.Buffer

	moveCursor(&buf, 5, 10)
	if buf.String() != "\033[5;10H" {
		t.Errorf("moveCursor: got %q", buf.String())
	}

	buf.Reset()
	clearScreen(&buf)
	if buf.String() != "\033[2J" {
		t.Errorf("clearScreen: got %q", buf.String())
	}

	buf.Reset()
	hideCursor(&buf)
	if buf.String() != "\033[?25l" {
		t.Errorf("hideCursor: got %q", buf.String())
	}

	buf.Reset()
	showCursor(&buf)
	if buf.String() != "\033[?25h" {
		t.Errorf("showCursor: got %q", buf.String())
	}
}

func TestMetricsPanel_Render(t *testing.T) {
	mp := &MetricsPanel{}
	mp.Update(MetricsData{
		ActiveConnections: 42,
		RequestRate:       15.5,
		WorkerStatus:      "4/4 running",
		Uptime:            2*time.Hour + 30*time.Minute,
	})

	var buf bytes.Buffer
	mp.Render(&buf, 120, 5)

	output := buf.String()
	if !strings.Contains(output, "42") {
		t.Errorf("expected active connections in output, got:\n%s", output)
	}
	if !strings.Contains(output, "15.5") {
		t.Errorf("expected request rate in output, got:\n%s", output)
	}
	if !strings.Contains(output, "4/4 running") {
		t.Errorf("expected worker status in output, got:\n%s", output)
	}
	if !strings.Contains(output, "2h 30m") {
		t.Errorf("expected uptime in output, got:\n%s", output)
	}
}

func TestMetricsPanel_Update_WrongType(t *testing.T) {
	mp := &MetricsPanel{}
	// Passing wrong type should not panic
	mp.Update("invalid")
	mp.Update(123)
	mp.Update(nil)
}

func TestLogPanel_Render(t *testing.T) {
	rb := newRingBuffer[LogEntry](100)
	rb.Push(LogEntry{
		Time:      time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Level:     LevelInfo,
		Component: "api",
		Message:   "request handled",
		Fields:    map[string]any{"status": 200},
	})

	lp := &LogPanel{ringBuf: rb}
	var buf bytes.Buffer
	lp.Render(&buf, 80, 10)

	output := buf.String()
	if !strings.Contains(output, "10:30:00") {
		t.Error("expected timestamp in log panel output")
	}
	if !strings.Contains(output, "api") {
		t.Error("expected component in log panel output")
	}
	if !strings.Contains(output, "request handled") {
		t.Error("expected message in log panel output")
	}
}

func TestLogPanel_RenderEmpty(t *testing.T) {
	rb := newRingBuffer[LogEntry](100)
	lp := &LogPanel{ringBuf: rb}

	var buf bytes.Buffer
	lp.Render(&buf, 80, 5)

	// Should produce output (empty lines) without panicking
	if buf.Len() == 0 {
		t.Error("expected some output even with empty ring buffer")
	}
}

func TestLogPanel_NilRingBuffer(t *testing.T) {
	lp := &LogPanel{ringBuf: nil}
	var buf bytes.Buffer
	// Should not panic
	lp.Render(&buf, 80, 5)
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"seconds", 45 * time.Second, "45s"},
		{"minutes", 5*time.Minute + 30*time.Second, "5m 30s"},
		{"hours", 3*time.Hour + 15*time.Minute, "3h 15m"},
		{"days", 26*time.Hour + 30*time.Minute, "1d 2h 30m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.d)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s      string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello"},
		{"", 5, ""},
		{"test", 0, ""},
	}
	for _, tt := range tests {
		got := truncate(tt.s, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
		}
	}
}

func TestGetTerminalSize_Fallback(t *testing.T) {
	// When not attached to a real terminal (e.g., in CI/test), should get fallback
	w, h := getTerminalSize()
	if w <= 0 || h <= 0 {
		t.Errorf("getTerminalSize returned invalid dimensions: %dx%d", w, h)
	}
}
