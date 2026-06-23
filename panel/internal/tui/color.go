package tui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// ANSI escape sequences for log level colors.
const (
	ansiReset  = "\033[0m"
	ansiGray   = "\033[37;2m" // dim white (gray) for DEBUG
	ansiGreen  = "\033[32m"   // green for INFO
	ansiYellow = "\033[33m"   // yellow for WARN
	ansiRed    = "\033[31m"   // red for ERROR
)

// colorForLevel returns the ANSI escape sequence for the given log level.
func colorForLevel(level Level) string {
	switch level {
	case LevelDebug:
		return ansiGray
	case LevelInfo:
		return ansiGreen
	case LevelWarn:
		return ansiYellow
	case LevelError:
		return ansiRed
	default:
		return ansiReset
	}
}

// colorize wraps text with the ANSI color code for the given log level.
func colorize(level Level, text string) string {
	return colorForLevel(level) + text + ansiReset
}

// isTTY checks whether the given writer is a terminal (TTY) by inspecting
// os.Stdout's file mode for the os.ModeCharDevice bit.
func isTTY(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		info, err := f.Stat()
		if err != nil {
			return false
		}
		return info.Mode()&os.ModeCharDevice != 0
	}
	return false
}

// formatColored formats a log entry with ANSI color codes for TTY output.
// Format: [LEVEL] [component] message key=value key=value...
func formatColored(entry LogEntry) string {
	var sb strings.Builder
	sb.WriteString(colorize(entry.Level, fmt.Sprintf("[%-5s]", entry.Level.String())))
	sb.WriteString(" ")
	sb.WriteString(fmt.Sprintf("[%s]", entry.Component))
	sb.WriteString(" ")
	sb.WriteString(entry.Message)
	if len(entry.Fields) > 0 {
		for k, v := range entry.Fields {
			sb.WriteString(fmt.Sprintf(" %s=%v", k, v))
		}
	}
	sb.WriteString("\n")
	return sb.String()
}

// formatJSON formats a log entry as a JSON-structured line for non-TTY output.
// Output: {"time":"...","level":"INFO","component":"api","message":"...","field":"value"}
func formatJSON(entry LogEntry) string {
	obj := map[string]any{
		"time":      entry.Time.Format(time.RFC3339),
		"level":     entry.Level.String(),
		"component": entry.Component,
		"message":   entry.Message,
	}
	for k, v := range entry.Fields {
		obj[k] = v
	}
	data, err := json.Marshal(obj)
	if err != nil {
		// Fallback to a minimal JSON line on marshal error.
		return fmt.Sprintf(`{"time":"%s","level":"%s","component":"%s","message":"%s"}`+"\n",
			entry.Time.Format(time.RFC3339), entry.Level.String(), entry.Component, entry.Message)
	}
	return string(data) + "\n"
}
