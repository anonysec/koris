package tui

import "time"

// LogEntry represents a single structured log record stored in the ring buffer
// and rendered by the dashboard.
type LogEntry struct {
	// Time is the UTC timestamp when the entry was created.
	Time time.Time
	// Level is the severity of the log entry.
	Level Level
	// Component identifies the subsystem that produced the log (e.g. "backup", "api", "bot").
	Component string
	// Message is the human-readable log message.
	Message string
	// Fields contains optional structured key-value data attached to the entry.
	Fields map[string]any
}
