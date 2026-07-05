package tui

import "strings"

// Level represents log severity for the TUI logger.
type Level int

const (
	// LevelDebug is the most verbose log level.
	LevelDebug Level = iota
	// LevelInfo is the default log level for normal operational messages.
	LevelInfo
	// LevelWarn indicates a potential issue that does not interrupt operation.
	LevelWarn
	// LevelError indicates a failure that requires attention.
	LevelError
)

// String returns the human-readable name of the log level.
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel converts a string log level to a Level constant.
// Returns LevelInfo for unrecognized values.
func ParseLevel(s string) Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}
