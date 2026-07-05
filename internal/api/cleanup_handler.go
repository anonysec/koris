package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/anonysec/koris/internal/cleanup"
)

// cleanupPost handles the admin cleanup API.
// POST /api/admin/cleanup
//
// Request body:
//
//	{"dry_run": true, "confirm": false, "older_than": "90d", "targets": ["stale_sessions", "old_events"]}
func (s *Server) cleanupPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limitBody(w, r, maxJSONBody)

	var in struct {
		DryRun    bool     `json:"dry_run"`
		Confirm   bool     `json:"confirm"`
		OlderThan string   `json:"older_than"` // format: "90d"
		Targets   []string `json:"targets"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bad_json"})
		return
	}

	// Parse older_than duration
	olderThan := 90 * 24 * time.Hour // default 90 days
	if in.OlderThan != "" {
		days, err := parseDaysDuration(in.OlderThan)
		if err != nil {
			writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_older_than"})
			return
		}
		if days < 7 {
			writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "minimum_7_days"})
			return
		}
		olderThan = time.Duration(days) * 24 * time.Hour
	}

	// Convert target strings to CleanupTarget type
	targets := make([]cleanup.CleanupTarget, 0, len(in.Targets))
	for _, t := range in.Targets {
		targets = append(targets, cleanup.CleanupTarget(t))
	}

	svc := cleanup.New(s.DB)

	// Set up WebSocket notification for progress
	svc.SetNotify(func(event cleanup.CleanupEvent) {
		s.broadcastWSCleanup(map[string]any{
			"type":    "cleanup_progress",
			"target":  event.Target,
			"deleted": event.Deleted,
			"message": event.Message,
		})
	})

	req := cleanup.CleanupRequest{
		Targets:   targets,
		OlderThan: olderThan,
		DryRun:    in.DryRun,
		BatchSize: 1000,
	}

	if in.DryRun || (!in.Confirm) {
		// Preview mode
		previews, err := svc.Preview(r.Context(), req)
		if err != nil {
			writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "preview_failed"})
			return
		}

		type result struct {
			Target   string `json:"target"`
			RowCount int64  `json:"row_count"`
			Oldest   string `json:"oldest"`
		}
		results := make([]result, 0, len(previews))
		for _, p := range previews {
			oldest := ""
			if !p.OldestRow.IsZero() {
				oldest = p.OldestRow.Format(time.RFC3339)
			}
			results = append(results, result{
				Target:   string(p.Target),
				RowCount: p.RowCount,
				Oldest:   oldest,
			})
		}

		writeJSON(w, map[string]any{
			"ok":      true,
			"dry_run": true,
			"results": results,
		})
		return
	}

	// Execute mode
	results, err := svc.Execute(r.Context(), req)
	if err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	type resultEntry struct {
		Target      string `json:"target"`
		RowsDeleted int64  `json:"rows_deleted"`
		DurationMs  int64  `json:"duration_ms"`
		Error       string `json:"error,omitempty"`
	}
	output := make([]resultEntry, 0, len(results))
	for _, r := range results {
		entry := resultEntry{
			Target:      string(r.Target),
			RowsDeleted: r.RowsAffected,
			DurationMs:  r.Duration.Milliseconds(),
			Error:       r.Error,
		}
		output = append(output, entry)
	}

	// Audit log
	s.logAudit("admin", "cleanup_execute", "system", "", nil, map[string]any{"targets": len(targets), "older_than": in.OlderThan}, "")

	writeJSON(w, map[string]any{
		"ok":      true,
		"dry_run": false,
		"results": output,
	})
}

// cleanupSchedulePost configures auto-cleanup schedule.
// POST /api/admin/cleanup/schedule
func (s *Server) cleanupSchedulePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limitBody(w, r, maxJSONBody)

	var in struct {
		Enabled       bool           `json:"enabled"`
		RunAt         string         `json:"run_at"` // HH:MM format
		RetentionDays map[string]int `json:"retention_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bad_json"})
		return
	}

	// Validate run_at format
	if in.RunAt != "" {
		parts := strings.Split(in.RunAt, ":")
		if len(parts) != 2 {
			writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_run_at_format"})
			return
		}
	}

	// Store in panel_settings as JSON
	configJSON, _ := json.Marshal(in)
	_, err := s.DB.Exec(`INSERT INTO panel_settings (key_name, value) VALUES ('auto_cleanup_config', $1) ON CONFLICT (key_name) DO UPDATE SET value=EXCLUDED.value`,
		string(configJSON))
	if err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error"})
		return
	}

	writeJSON(w, map[string]any{"ok": true})
}

// parseDaysDuration parses a string like "90d" into an integer number of days.
func parseDaysDuration(s string) (int, error) {
	s = strings.TrimSpace(s)
	if !strings.HasSuffix(s, "d") {
		return 0, fmt.Errorf("invalid format: %s (expected Nd)", s)
	}
	numStr := strings.TrimSuffix(s, "d")
	days, err := strconv.Atoi(numStr)
	if err != nil || days < 1 {
		return 0, fmt.Errorf("invalid days: %s", s)
	}
	return days, nil
}

// broadcastWSCleanup sends a cleanup progress message to all connected WebSocket clients.
func (s *Server) broadcastWSCleanup(msg map[string]any) {
	s.wsNotifMu.RLock()
	defer s.wsNotifMu.RUnlock()
	for _, ch := range s.wsNotifChans {
		select {
		case ch <- msg:
		default:
			// Drop if channel is full
		}
	}
}
