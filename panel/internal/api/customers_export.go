package api

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const maxExportRows = 10000

// adminCustomersExport handles GET /api/admin/customers/export.
// Supports ?format=csv (default) or ?format=json.
// Applies the same filters as the list endpoint: ?status=, ?plan_id=, ?search=.
// Streams results directly to the client without buffering the full result set.
func (s *Server) adminCustomersExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	params := r.URL.Query()
	format := strings.ToLower(strings.TrimSpace(params.Get("format")))
	if format == "" {
		format = "csv"
	}
	if format != "csv" && format != "json" {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_format"})
		return
	}

	// --- Build WHERE clause (same filters as list endpoint) ---
	where := "c.deleted_at IS NULL"
	args := []any{}

	// Filter: search
	if search := strings.TrimSpace(params.Get("search")); search != "" {
		where += " AND (c.username LIKE ? OR COALESCE(c.display_name,'') LIKE ? OR COALESCE(c.email,'') LIKE ?)"
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}

	// Filter: status
	if status := strings.TrimSpace(params.Get("status")); status != "" {
		where += " AND c.status = ?"
		args = append(args, status)
	}

	// Filter: plan_id
	if planIDStr := params.Get("plan_id"); planIDStr != "" {
		if pid, err := strconv.ParseInt(planIDStr, 10, 64); err == nil && pid > 0 {
			where += " AND c.plan_id = ?"
			args = append(args, pid)
		}
	}

	query := fmt.Sprintf(`SELECT c.id, c.username, COALESCE(c.display_name,''), COALESCE(c.email,''), c.status, COALESCE(p.name,''), c.created_at
		FROM customers c
		LEFT JOIN plans p ON p.id = c.plan_id
		WHERE %s
		ORDER BY c.id DESC
		LIMIT %d`, where, maxExportRows)

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		log.Printf("[customers] export query error: %v", err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error"})
		return
	}
	defer rows.Close()

	switch format {
	case "csv":
		s.exportCustomersAsCSV(w, rows)
	case "json":
		s.exportCustomersAsJSON(w, rows)
	}
}

func (s *Server) exportCustomersAsCSV(w http.ResponseWriter, rows *sql.Rows) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="customers_export.csv"`)

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"id", "username", "display_name", "email", "status", "plan_name", "created_at"})

	for rows.Next() {
		var id int64
		var username, displayName, email, status, planName string
		var createdAt sql.NullTime
		if err := rows.Scan(&id, &username, &displayName, &email, &status, &planName, &createdAt); err != nil {
			continue
		}
		createdStr := ""
		if createdAt.Valid {
			createdStr = createdAt.Time.UTC().Format(time.RFC3339)
		}
		_ = cw.Write([]string{
			strconv.FormatInt(id, 10),
			username,
			displayName,
			email,
			status,
			planName,
			createdStr,
		})
	}
	cw.Flush()
}

type exportCustomerRow struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Status      string `json:"status"`
	PlanName    string `json:"plan_name"`
	CreatedAt   string `json:"created_at"`
}

func (s *Server) exportCustomersAsJSON(w http.ResponseWriter, rows *sql.Rows) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// Stream JSON array: write opening bracket, then each row comma-separated
	_, _ = w.Write([]byte("["))
	first := true

	enc := json.NewEncoder(w)
	for rows.Next() {
		var id int64
		var username, displayName, email, status, planName string
		var createdAt sql.NullTime
		if err := rows.Scan(&id, &username, &displayName, &email, &status, &planName, &createdAt); err != nil {
			continue
		}
		createdStr := ""
		if createdAt.Valid {
			createdStr = createdAt.Time.UTC().Format(time.RFC3339)
		}

		if !first {
			_, _ = w.Write([]byte(","))
		}
		first = false

		_ = enc.Encode(exportCustomerRow{
			ID:          id,
			Username:    username,
			DisplayName: displayName,
			Email:       email,
			Status:      status,
			PlanName:    planName,
			CreatedAt:   createdStr,
		})
	}
	_, _ = w.Write([]byte("]"))
}
