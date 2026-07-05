package api

import (
	"net/http"
	"strconv"
	"time"
)

// internalLogs returns recent log entries from the TUI logger's ring buffer.
// It does not require authentication since it is only exposed on the
// Unix socket or localhost internal listener.
// GET /internal/logs?tail=100
func (s *Server) internalLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	if s.LogEntries == nil {
		writeJSONCode(w, http.StatusServiceUnavailable, map[string]any{
			"ok":    false,
			"error": "log_entries_unavailable",
		})
		return
	}

	// Parse tail parameter (default 100, max 1000).
	tail := 100
	if t := r.URL.Query().Get("tail"); t != "" {
		parsed, err := strconv.Atoi(t)
		if err != nil || parsed < 1 {
			writeJSONCode(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": "invalid 'tail' parameter",
			})
			return
		}
		tail = parsed
		if tail > 1000 {
			tail = 1000
		}
	}

	type logEntryResponse struct {
		Time      string         `json:"time"`
		Level     string         `json:"level"`
		Component string         `json:"component"`
		Message   string         `json:"message"`
		Fields    map[string]any `json:"fields,omitempty"`
	}

	rawEntries := s.LogEntries(tail)
	entries := make([]logEntryResponse, 0, len(rawEntries))
	for _, e := range rawEntries {
		entries = append(entries, logEntryResponse{
			Time:      e.Time.Format(time.RFC3339),
			Level:     e.Level.String(),
			Component: e.Component,
			Message:   e.Message,
			Fields:    e.Fields,
		})
	}

	writeJSON(w, map[string]any{
		"ok":      true,
		"entries": entries,
	})
}
