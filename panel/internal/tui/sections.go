package tui

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// MetricsData holds the values displayed in the metrics panel.
type MetricsData struct {
	ActiveConnections int
	RequestRate       float64 // requests per second
	WorkerStatus      string  // e.g. "4/4 running"
	Uptime            time.Duration
}

// MetricsPanel displays key server metrics: active connections, request rate,
// worker status, and uptime. Values are populated via Update().
type MetricsPanel struct {
	mu   sync.Mutex
	data MetricsData
}

// Render writes the metrics panel content to w within the given bounds.
func (m *MetricsPanel) Render(w io.Writer, width, height int) {
	m.mu.Lock()
	data := m.data
	m.mu.Unlock()

	// Title line
	title := "╔═ KorisPanel Dashboard "
	padding := width - len(title) - 1
	if padding < 0 {
		padding = 0
	}
	fmt.Fprintf(w, "%s%s╗\n", title, strings.Repeat("═", padding))

	// Metrics row 1
	line1 := fmt.Sprintf("║ Connections: %-6d │ Req/s: %-8.1f │ Workers: %-12s │ Uptime: %s",
		data.ActiveConnections,
		data.RequestRate,
		data.WorkerStatus,
		formatDuration(data.Uptime),
	)
	// Pad to width
	if len(line1) < width-1 {
		line1 += strings.Repeat(" ", width-1-len(line1))
	}
	fmt.Fprintf(w, "%s║\n", truncate(line1, width-1))

	// Bottom border
	bottom := "╚" + strings.Repeat("═", width-2) + "╝"
	fmt.Fprintln(w, bottom)

	// Fill remaining height with empty lines if needed
	for i := 3; i < height; i++ {
		fmt.Fprintln(w)
	}
}

// Update receives new metrics data. Expects data to be of type MetricsData.
func (m *MetricsPanel) Update(data any) {
	if d, ok := data.(MetricsData); ok {
		m.mu.Lock()
		m.data = d
		m.mu.Unlock()
	}
}

// LogPanel displays recent log entries from the ring buffer, formatted with colors.
type LogPanel struct {
	mu      sync.Mutex
	ringBuf *RingBuffer[LogEntry]
}

// Render writes the most recent log entries that fit within the given height.
func (lp *LogPanel) Render(w io.Writer, width, height int) {
	if lp.ringBuf == nil {
		return
	}

	// Get the last N entries that fit in our height
	entries := lp.ringBuf.Last(height)

	for i, entry := range entries {
		if i >= height {
			break
		}
		line := formatLogLine(entry, width)
		fmt.Fprint(w, line)
	}

	// Fill remaining lines with empty space to avoid ghost content
	remaining := height - len(entries)
	for i := 0; i < remaining; i++ {
		fmt.Fprintf(w, "%s\n", strings.Repeat(" ", width))
	}
}

// Update is a no-op for LogPanel since it reads directly from the ring buffer.
// The data parameter is ignored.
func (lp *LogPanel) Update(data any) {
	// LogPanel reads directly from ringBuf, no update needed.
}

// formatLogLine formats a single log entry for the log panel display.
func formatLogLine(entry LogEntry, maxWidth int) string {
	timestamp := entry.Time.Format("15:04:05")
	level := colorize(entry.Level, fmt.Sprintf("%-5s", entry.Level.String()))
	component := fmt.Sprintf("[%s]", entry.Component)

	line := fmt.Sprintf("%s %s %s %s", timestamp, level, component, entry.Message)

	// Append fields if space allows
	if len(entry.Fields) > 0 {
		var fields []string
		for k, v := range entry.Fields {
			fields = append(fields, fmt.Sprintf("%s=%v", k, v))
		}
		fieldStr := strings.Join(fields, " ")
		line += " " + fieldStr
	}

	// Truncate to width and ensure newline
	if len(line) > maxWidth {
		line = line[:maxWidth-1] + "…"
	}
	return line + "\n"
}

// formatDuration formats a duration as a human-readable string like "2h 15m".
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if hours >= 24 {
		days := hours / 24
		hours = hours % 24
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

// truncate ensures a string does not exceed maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 0 {
		return ""
	}
	return s[:maxLen]
}
