package api

import (
	"encoding/json"
	"net/http"
)

// internalCleanup handles cleanup requests from the CLI.
// It does not require authentication since it is only exposed on the
// Unix socket or localhost internal listener.
//
// POST /internal/cleanup
//
// Request body:
//
//	{"dry_run": true, "confirm": false, "older_than": "90d", "targets": ["stale_sessions", "old_events"]}
//
// This is a placeholder implementation that returns mock data.
// The actual cleanup logic will be implemented in Task Group 9 via CleanupService.
func (s *Server) internalCleanup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	limitBody(w, r, maxJSONBody)

	var in struct {
		DryRun    bool     `json:"dry_run"`
		Confirm   bool     `json:"confirm"`
		OlderThan string   `json:"older_than"`
		Targets   []string `json:"targets"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bad_json"})
		return
	}

	// Placeholder response — real implementation will come from CleanupService (Task Group 9).
	type result struct {
		Target   string `json:"target"`
		RowCount int64  `json:"row_count"`
		Oldest   string `json:"oldest"`
	}

	results := []result{
		{Target: "stale_sessions", RowCount: 150, Oldest: "2024-01-01T00:00:00Z"},
		{Target: "old_events", RowCount: 2300, Oldest: "2023-06-15T00:00:00Z"},
	}

	// Filter by requested targets if specified.
	if len(in.Targets) > 0 {
		targetSet := make(map[string]bool, len(in.Targets))
		for _, t := range in.Targets {
			targetSet[t] = true
		}
		filtered := []result{}
		for _, r := range results {
			if targetSet[r.Target] {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	writeJSON(w, map[string]any{
		"ok":      true,
		"dry_run": in.DryRun,
		"results": results,
	})
}
