package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// cleanupTarget defines the SQL logic for a cleanup target.
type cleanupTarget struct {
	Name      string
	Table     string
	Condition string // SQL WHERE fragment (e.g. "created_at < NOW() - INTERVAL '...'")
	TimestampCol string
}

// knownTargets maps cleanup target names to their SQL configuration.
// All targets use the *timestamp* column with the parsed older_than interval.
var knownTargets = map[string]cleanupTarget{
	"stale_sessions":   {Name: "stale_sessions", Table: "admin_sessions", TimestampCol: "last_active_at", Condition: "last_active_at < NOW() - INTERVAL '%s'"},
	"expired_sessions": {Name: "expired_sessions", Table: "admin_sessions", TimestampCol: "expires_at", Condition: "expires_at < NOW() - INTERVAL '%s'"},
	"old_events":       {Name: "old_events", Table: "events", TimestampCol: "created_at", Condition: "created_at < NOW() - INTERVAL '%s'"},
	"old_api_logs":     {Name: "old_api_logs", Table: "api_logs", TimestampCol: "created_at", Condition: "created_at < NOW() - INTERVAL '%s'"},
	"old_audit_logs":   {Name: "old_audit_logs", Table: "audit_logs", TimestampCol: "created_at", Condition: "created_at < NOW() - INTERVAL '%s'"},
	"old_webhook_logs": {Name: "old_webhook_logs", Table: "webhook_logs", TimestampCol: "created_at", Condition: "created_at < NOW() - INTERVAL '%s'"},
	"old_radacct_archive": {Name: "old_radacct_archive", Table: "radacct_archive", TimestampCol: "archived_at", Condition: "archived_at < NOW() - INTERVAL '%s'"},
	"old_deleted_archive": {Name: "old_deleted_archive", Table: "deleted_archive", TimestampCol: "created_at", Condition: "created_at < NOW() - INTERVAL '%s' AND restored_at IS NULL"},
	"old_login_attempts": {Name: "old_login_attempts", Table: "login_attempts", TimestampCol: "created_at", Condition: "created_at < NOW() - INTERVAL '%s'"},
}

// parseOlderThan converts a duration string like "90d", "7d", "24h", "30m" into
// a PostgreSQL interval string. Defaults to "90d" if empty or invalid.
func parseOlderThan(input string) string {
	if input == "" {
		return "90 days"
	}
	input = strings.TrimSpace(strings.ToLower(input))

	// Extract numeric part and unit
	numStr := input
	unit := "d"
	for i, ch := range input {
		if ch < '0' || ch > '9' {
			numStr = input[:i]
			unit = input[i:]
			break
		}
	}
	num, err := strconv.Atoi(numStr)
	if err != nil || num <= 0 {
		return "90 days"
	}

	switch unit {
	case "d", "day", "days":
		return fmt.Sprintf("%d days", num)
	case "h", "hour", "hours":
		return fmt.Sprintf("%d hours", num)
	case "m", "min", "mins", "minute", "minutes":
		return fmt.Sprintf("%d minutes", num)
	default:
		return fmt.Sprintf("%d days", num)
	}
}

// internalCleanup handles cleanup requests from the CLI or admin API.
// Requires admin authentication because it is exposed on the public HTTP listener.
//
// POST /internal/cleanup
//
// Request body:
//	{"dry_run": true, "confirm": false, "older_than": "90d", "targets": ["stale_sessions", "old_events"]}
//
// When dry_run is true, returns counts but does not delete.
// When dry_run is false and confirm is true, performs actual deletion.
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

	interval := parseOlderThan(in.OlderThan)

	type result struct {
		Target   string `json:"target"`
		RowCount int64  `json:"row_count"`
		Oldest   string `json:"oldest"`
		Deleted  bool   `json:"deleted"`
	}

	results := []result{}

	// Determine which targets to process
	targetNames := in.Targets
	if len(targetNames) == 0 {
		// Default to all known targets
		for name := range knownTargets {
			targetNames = append(targetNames, name)
		}
	}

	for _, name := range targetNames {
		target, ok := knownTargets[name]
		if !ok {
			results = append(results, result{
				Target:   name,
				RowCount: 0,
				Oldest:   "",
				Deleted:  false,
			})
			continue
		}

		// Count matching rows
		var rowCount int64
		var oldest sqlNullTime
		countQuery := fmt.Sprintf(
			"SELECT COUNT(*), MIN(%s) FROM %s WHERE "+target.Condition,
			target.TimestampCol, target.Table, interval,
		)
		err := s.DB.QueryRow(countQuery).Scan(&rowCount, &oldest)
		if err != nil {
			log.Printf("[cleanup] failed to count %s: %v", target.Name, err)
			results = append(results, result{
				Target:   target.Name,
				RowCount: 0,
				Oldest:   "",
				Deleted:  false,
			})
			continue
		}

		deleted := false
		if !in.DryRun && in.Confirm && rowCount > 0 {
			deleteQuery := fmt.Sprintf(
				"DELETE FROM %s WHERE "+target.Condition,
				target.Table, interval,
			)
			_, err := s.DB.Exec(deleteQuery)
			if err != nil {
				log.Printf("[cleanup] failed to delete %s: %v", target.Name, err)
			} else {
				deleted = true
				log.Printf("[cleanup] deleted %d rows from %s (older than %s)", rowCount, target.Table, interval)
			}
		}

		results = append(results, result{
			Target:   target.Name,
			RowCount: rowCount,
			Oldest:   formatNullTime(oldest),
			Deleted:  deleted,
		})
	}

	writeJSON(w, map[string]any{
		"ok":        true,
		"dry_run":   in.DryRun,
		"older_than": interval,
		"results":   results,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// sqlNullTime wraps sql.NullTime for JSON-friendly output.
type sqlNullTime struct {
	Time  time.Time
	Valid bool
}

func (s *sqlNullTime) Scan(value interface{}) error {
	if value == nil {
		s.Time, s.Valid = time.Time{}, false
		return nil
	}
	s.Valid = true
	switch v := value.(type) {
	case time.Time:
		s.Time = v
		return nil
	default:
		return fmt.Errorf("cannot scan %T into sqlNullTime", value)
	}
}

func formatNullTime(t sqlNullTime) string {
	if !t.Valid || t.Time.IsZero() {
		return ""
	}
	return t.Time.UTC().Format(time.RFC3339)
}

