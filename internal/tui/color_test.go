package tui

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestColorize(t *testing.T) {
	tests := []struct {
		name     string
		level    Level
		text     string
		wantPre  string
		wantPost string
	}{
		{"debug gray", LevelDebug, "hello", ansiGray, ansiReset},
		{"info green", LevelInfo, "hello", ansiGreen, ansiReset},
		{"warn yellow", LevelWarn, "hello", ansiYellow, ansiReset},
		{"error red", LevelError, "hello", ansiRed, ansiReset},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := colorize(tt.level, tt.text)
			if !strings.HasPrefix(got, tt.wantPre) {
				t.Errorf("colorize(%v, %q) prefix = %q, want %q", tt.level, tt.text, got[:len(tt.wantPre)], tt.wantPre)
			}
			if !strings.HasSuffix(got, tt.wantPost) {
				t.Errorf("colorize(%v, %q) suffix = %q, want %q", tt.level, tt.text, got[len(got)-len(tt.wantPost):], tt.wantPost)
			}
			if !strings.Contains(got, tt.text) {
				t.Errorf("colorize(%v, %q) does not contain text", tt.level, tt.text)
			}
		})
	}
}

func TestColorForLevel(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{LevelDebug, ansiGray},
		{LevelInfo, ansiGreen},
		{LevelWarn, ansiYellow},
		{LevelError, ansiRed},
		{Level(99), ansiReset}, // unknown level
	}
	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			got := colorForLevel(tt.level)
			if got != tt.want {
				t.Errorf("colorForLevel(%v) = %q, want %q", tt.level, got, tt.want)
			}
		})
	}
}

func TestFormatColored(t *testing.T) {
	entry := LogEntry{
		Time:      time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Level:     LevelInfo,
		Component: "api",
		Message:   "server started",
		Fields:    map[string]any{"port": 8080},
	}
	got := formatColored(entry)

	// Should contain the colored level tag
	if !strings.Contains(got, ansiGreen) {
		t.Error("expected green ANSI code for INFO level")
	}
	if !strings.Contains(got, ansiReset) {
		t.Error("expected reset ANSI code")
	}
	// Should contain component in brackets
	if !strings.Contains(got, "[api]") {
		t.Error("expected [api] component tag")
	}
	// Should contain the message
	if !strings.Contains(got, "server started") {
		t.Error("expected message in output")
	}
	// Should contain the field
	if !strings.Contains(got, "port=8080") {
		t.Error("expected port=8080 field in output")
	}
	// Should end with newline
	if !strings.HasSuffix(got, "\n") {
		t.Error("expected trailing newline")
	}
}

func TestFormatColoredNoFields(t *testing.T) {
	entry := LogEntry{
		Time:      time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Level:     LevelWarn,
		Component: "backup",
		Message:   "disk low",
	}
	got := formatColored(entry)

	if !strings.Contains(got, ansiYellow) {
		t.Error("expected yellow ANSI code for WARN level")
	}
	if !strings.Contains(got, "[backup]") {
		t.Error("expected [backup] component tag")
	}
	if !strings.Contains(got, "disk low") {
		t.Error("expected message in output")
	}
}

func TestFormatJSON(t *testing.T) {
	entry := LogEntry{
		Time:      time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Level:     LevelError,
		Component: "db",
		Message:   "connection lost",
		Fields:    map[string]any{"attempts": 3},
	}
	got := formatJSON(entry)

	// Should be valid JSON
	var obj map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(got)), &obj); err != nil {
		t.Fatalf("formatJSON output is not valid JSON: %v\ngot: %s", err, got)
	}

	// Verify required fields
	if obj["time"] != "2024-01-15T10:30:00Z" {
		t.Errorf("time = %v, want 2024-01-15T10:30:00Z", obj["time"])
	}
	if obj["level"] != "ERROR" {
		t.Errorf("level = %v, want ERROR", obj["level"])
	}
	if obj["component"] != "db" {
		t.Errorf("component = %v, want db", obj["component"])
	}
	if obj["message"] != "connection lost" {
		t.Errorf("message = %v, want connection lost", obj["message"])
	}
	if obj["attempts"] != float64(3) {
		t.Errorf("attempts = %v, want 3", obj["attempts"])
	}

	// Should end with newline
	if !strings.HasSuffix(got, "\n") {
		t.Error("expected trailing newline")
	}
}

func TestFormatJSONNoFields(t *testing.T) {
	entry := LogEntry{
		Time:      time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Level:     LevelInfo,
		Component: "api",
		Message:   "request handled",
	}
	got := formatJSON(entry)

	var obj map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(got)), &obj); err != nil {
		t.Fatalf("formatJSON output is not valid JSON: %v", err)
	}
	if obj["level"] != "INFO" {
		t.Errorf("level = %v, want INFO", obj["level"])
	}
}

func TestIsTTY_NonFile(t *testing.T) {
	// A bytes.Buffer is not a file, so isTTY should return false.
	var buf bytes.Buffer
	if isTTY(&buf) {
		t.Error("expected isTTY(bytes.Buffer) = false")
	}
}

func TestLoggerColoredOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := New(WithOutput(&buf), WithColor(true), WithLevel(LevelDebug))

	logger.Info("api", "started", map[string]any{"port": 8080})

	got := buf.String()
	if !strings.Contains(got, ansiGreen) {
		t.Error("expected colored output with green for INFO")
	}
	if !strings.Contains(got, "[api]") {
		t.Error("expected [api] component")
	}
	if !strings.Contains(got, "started") {
		t.Error("expected message in output")
	}
}

func TestLoggerJSONOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := New(WithOutput(&buf), WithColor(false), WithLevel(LevelDebug))

	logger.Error("db", "connection failed", map[string]any{"host": "localhost"})

	got := buf.String()
	var obj map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(got)), &obj); err != nil {
		t.Fatalf("expected valid JSON output, got: %s", got)
	}
	if obj["level"] != "ERROR" {
		t.Errorf("level = %v, want ERROR", obj["level"])
	}
	if obj["component"] != "db" {
		t.Errorf("component = %v, want db", obj["component"])
	}
	if obj["message"] != "connection failed" {
		t.Errorf("message = %v, want connection failed", obj["message"])
	}
	if obj["host"] != "localhost" {
		t.Errorf("host = %v, want localhost", obj["host"])
	}
}

func TestLoggerAutoDetectNonTTY(t *testing.T) {
	// When output is a buffer (non-TTY), auto-detection should disable color.
	var buf bytes.Buffer
	logger := New(WithOutput(&buf), WithLevel(LevelDebug))

	logger.Info("test", "hello")

	got := buf.String()
	// Should be JSON (no ANSI codes) since bytes.Buffer is not a TTY.
	if strings.Contains(got, "\033[") {
		t.Error("expected no ANSI codes for non-TTY output")
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(got)), &obj); err != nil {
		t.Fatalf("expected valid JSON for non-TTY output, got: %s", got)
	}
}

func TestLoggerWithColorOverride(t *testing.T) {
	// WithColor(true) should force colored output even for non-TTY writer.
	var buf bytes.Buffer
	logger := New(WithOutput(&buf), WithColor(true), WithLevel(LevelDebug))

	logger.Warn("test", "warning message")

	got := buf.String()
	if !strings.Contains(got, ansiYellow) {
		t.Error("expected yellow ANSI code when WithColor(true) is set")
	}
}

func TestLoggerLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := New(WithOutput(&buf), WithColor(false), WithLevel(LevelWarn))

	logger.Debug("test", "debug msg")
	logger.Info("test", "info msg")
	logger.Warn("test", "warn msg")
	logger.Error("test", "error msg")

	got := buf.String()
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 log lines (WARN+ERROR), got %d: %s", len(lines), got)
	}
}

func TestANSISequences(t *testing.T) {
	// Verify the exact ANSI escape sequences embedded in colored output.
	tests := []struct {
		name     string
		level    Level
		wantSeq  string
		wantText string
	}{
		{"debug escape", LevelDebug, "\033[37;2m", "DBG"},
		{"info escape", LevelInfo, "\033[32m", "INF"},
		{"warn escape", LevelWarn, "\033[33m", "WRN"},
		{"error escape", LevelError, "\033[31m", "ERR"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := New(WithOutput(&buf), WithColor(true), WithLevel(LevelDebug))

			switch tt.level {
			case LevelDebug:
				logger.Debug("test", "msg")
			case LevelInfo:
				logger.Info("test", "msg")
			case LevelWarn:
				logger.Warn("test", "msg")
			case LevelError:
				logger.Error("test", "msg")
			}

			got := buf.String()
			if !strings.Contains(got, tt.wantSeq) {
				t.Errorf("output does not contain ANSI sequence %q for %v level", tt.wantSeq, tt.level)
			}
			if !strings.Contains(got, "\033[0m") {
				t.Errorf("output missing ANSI reset sequence \\033[0m")
			}
		})
	}
}

func TestNoANSIWhenColorDisabled(t *testing.T) {
	var buf bytes.Buffer
	logger := New(WithOutput(&buf), WithColor(false), WithLevel(LevelDebug))

	logger.Debug("test", "debug")
	logger.Info("test", "info")
	logger.Warn("test", "warn")
	logger.Error("test", "error")

	got := buf.String()
	if strings.Contains(got, "\033[") {
		t.Errorf("expected no ANSI escape codes when color is disabled, got: %s", got)
	}

	// Verify all output lines are valid JSON
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 log lines, got %d", len(lines))
	}
	for i, line := range lines {
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("line %d is not valid JSON: %v", i, err)
		}
	}
}

func TestAllLevelColorsInOutput(t *testing.T) {
	// Verify each log level produces distinct ANSI color codes in formatted output.
	tests := []struct {
		level     Level
		wantColor string
		wantLabel string
	}{
		{LevelDebug, "\033[37;2m", "DEBUG"},
		{LevelInfo, "\033[32m", "INFO"},
		{LevelWarn, "\033[33m", "WARN"},
		{LevelError, "\033[31m", "ERROR"},
	}
	for _, tt := range tests {
		t.Run(tt.wantLabel, func(t *testing.T) {
			entry := LogEntry{
				Time:      time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC),
				Level:     tt.level,
				Component: "test",
				Message:   "test message",
			}
			output := formatColored(entry)

			if !strings.Contains(output, tt.wantColor) {
				t.Errorf("formatColored for %s: missing color code %q in output: %s",
					tt.wantLabel, tt.wantColor, output)
			}
			if !strings.Contains(output, "\033[0m") {
				t.Errorf("formatColored for %s: missing reset code", tt.wantLabel)
			}
			if !strings.Contains(output, "[test]") {
				t.Errorf("formatColored for %s: missing component tag", tt.wantLabel)
			}
			if !strings.Contains(output, "test message") {
				t.Errorf("formatColored for %s: missing message", tt.wantLabel)
			}
		})
	}
}
