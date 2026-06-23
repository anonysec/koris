package tui

import (
	"io"
	"time"
)

// Option is a functional option for configuring the Logger.
type Option func(*Logger)

// WithLevel sets the minimum log level. Messages below this level are discarded.
func WithLevel(level Level) Option {
	return func(l *Logger) {
		l.level = level
	}
}

// WithOutput sets the io.Writer where log output is written.
// Defaults to os.Stdout.
func WithOutput(w io.Writer) Option {
	return func(l *Logger) {
		l.output = w
	}
}

// WithRefreshInterval sets the TUI dashboard refresh rate.
// Defaults to 1 second.
func WithRefreshInterval(d time.Duration) Option {
	return func(l *Logger) {
		l.refreshInterval = d
	}
}

// WithDashboard enables or disables the live TUI dashboard.
// When disabled, the logger outputs plain formatted log lines.
// Explicitly calling this option prevents the PANEL_TUI_ENABLED
// environment variable from overriding the setting.
func WithDashboard(enabled bool) Option {
	return func(l *Logger) {
		l.dashboardEnabled = enabled
		l.dashboardOverridden = true
	}
}

// WithBufferSize sets the ring buffer capacity for log entries.
// Defaults to 1000.
func WithBufferSize(size int) Option {
	return func(l *Logger) {
		if size > 0 {
			l.bufferSize = size
		}
	}
}

// WithColor explicitly enables or disables ANSI color output.
// When not called, color is auto-detected based on whether output is a TTY.
func WithColor(enabled bool) Option {
	return func(l *Logger) {
		l.colorEnabled = enabled
		l.colorOverridden = true
	}
}
