// Package tui provides a structured terminal UI logger with colored output,
// a ring buffer for recent entries, and an optional live dashboard.
//
// Usage:
//
//	logger := tui.New(
//	    tui.WithLevel(tui.LevelInfo),
//	    tui.WithRefreshInterval(time.Second),
//	)
//	logger.Info("api", "server started", map[string]any{"port": 8080})
package tui

import (
	"context"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	// defaultBufferSize is the default ring buffer capacity.
	defaultBufferSize = 1000
	// defaultRefreshInterval is the default dashboard refresh rate.
	defaultRefreshInterval = time.Second
)

// Logger provides structured, colored terminal output with an optional
// live dashboard. It maintains a ring buffer of recent log entries and
// supports level-based filtering.
type Logger struct {
	output              io.Writer
	level               Level
	mu                  sync.Mutex
	dashboard           *Dashboard
	ringBuf             *RingBuffer[LogEntry]
	refreshInterval     time.Duration
	dashboardEnabled    bool
	dashboardOverridden bool // true when WithDashboard was explicitly called
	bufferSize          int
	colorEnabled        bool
	colorOverridden     bool // true when WithColor was explicitly called
}

// Note: Dashboard struct is defined in dashboard.go with full implementation.

// New creates a new Logger with the given options applied.
// Defaults: output=os.Stdout, level=LevelInfo, bufferSize=1000,
// refreshInterval=1s, dashboard=true, colorEnabled=auto-detected via TTY.
func New(opts ...Option) *Logger {
	l := &Logger{
		output:           os.Stdout,
		level:            LevelInfo,
		refreshInterval:  defaultRefreshInterval,
		dashboardEnabled: true,
		bufferSize:       defaultBufferSize,
	}
	for _, opt := range opts {
		opt(l)
	}
	// Check PANEL_TUI_ENABLED env var unless WithDashboard was explicitly called.
	if !l.dashboardOverridden {
		if v := os.Getenv("PANEL_TUI_ENABLED"); v != "" {
			switch strings.ToLower(v) {
			case "false", "0", "no":
				l.dashboardEnabled = false
			}
		}
	}
	// Auto-detect TTY for color support unless explicitly overridden by WithColor.
	if !l.colorOverridden {
		l.colorEnabled = isTTY(l.output)
	}
	l.ringBuf = newRingBuffer[LogEntry](l.bufferSize)
	return l
}

// Info logs a message at INFO level with the given component tag.
func (l *Logger) Info(component, msg string, fields ...map[string]any) {
	l.log(LevelInfo, component, msg, fields...)
}

// Warn logs a message at WARN level with the given component tag.
func (l *Logger) Warn(component, msg string, fields ...map[string]any) {
	l.log(LevelWarn, component, msg, fields...)
}

// Error logs a message at ERROR level with the given component tag.
func (l *Logger) Error(component, msg string, fields ...map[string]any) {
	l.log(LevelError, component, msg, fields...)
}

// Debug logs a message at DEBUG level with the given component tag.
func (l *Logger) Debug(component, msg string, fields ...map[string]any) {
	l.log(LevelDebug, component, msg, fields...)
}

// StartDashboard launches the live TUI dashboard rendering loop.
// It blocks until the context is cancelled or an error occurs.
func (l *Logger) StartDashboard(ctx context.Context) error {
	if !l.dashboardEnabled {
		<-ctx.Done()
		return ctx.Err()
	}

	l.mu.Lock()
	l.dashboard = NewDashboard(l.output, l.refreshInterval, l.ringBuf)
	l.mu.Unlock()

	return l.dashboard.Start(ctx)
}

// StopDashboard terminates the live TUI dashboard and restores the terminal.
func (l *Logger) StopDashboard() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.dashboard != nil {
		l.dashboard.Stop()
		l.dashboard = nil
	}
}

// SetLevel dynamically changes the minimum log level.
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// Entries returns a copy of all log entries currently in the ring buffer.
func (l *Logger) Entries() []LogEntry {
	if l.ringBuf == nil {
		return nil
	}
	return l.ringBuf.All()
}

// LastEntries returns up to the last n entries (most recent), ordered oldest to newest.
func (l *Logger) LastEntries(n int) []LogEntry {
	if l.ringBuf == nil {
		return nil
	}
	return l.ringBuf.Last(n)
}

// log is the internal method that filters by level, builds a LogEntry,
// stores it in the ring buffer, and writes formatted output to l.output.
// When colorEnabled is true (TTY detected), output is ANSI-colored.
// When colorEnabled is false (non-TTY), output is JSON-structured lines.
func (l *Logger) log(level Level, component, msg string, fields ...map[string]any) {
	l.mu.Lock()
	if level < l.level {
		l.mu.Unlock()
		return
	}
	l.mu.Unlock()

	entry := LogEntry{
		Time:      time.Now().UTC(),
		Level:     level,
		Component: component,
		Message:   msg,
	}
	if len(fields) > 0 && fields[0] != nil {
		entry.Fields = fields[0]
	}

	l.ringBuf.Push(entry)

	// Format and write to output.
	var line string
	if l.colorEnabled {
		line = formatColored(entry)
	} else {
		line = formatJSON(entry)
	}

	l.mu.Lock()
	_, _ = io.WriteString(l.output, line)
	l.mu.Unlock()
}
