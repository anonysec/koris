package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// internalUsers returns the user list for CLI consumption.
// It does not require authentication since it is only exposed on the
// Unix socket or localhost internal listener.
// GET /internal/users?status=active&page=1&limit=50
func (s *Server) internalUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()

	// Parse pagination params with defaults.
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	status := q.Get("status")

	// Build query with optional status filter.
	countQuery := `SELECT COUNT(*) FROM customers WHERE deleted_at IS NULL`
	dataQuery := `SELECT c.id, c.username, c.status, COALESCE(p.name, ''), c.created_at
		FROM customers c
		LEFT JOIN plans p ON p.id = c.plan_id
		WHERE c.deleted_at IS NULL`

	var args []any
	argN := 1
	if status != "" {
		countQuery += fmt.Sprintf(` AND status = $%d`, argN)
		dataQuery += fmt.Sprintf(` AND c.status = $%d`, argN)
		args = append(args, status)
		argN++
	}

	dataQuery += fmt.Sprintf(` ORDER BY c.id DESC LIMIT $%d OFFSET $%d`, argN, argN+1)

	// Get total count.
	var total int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	if err := s.DB.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error"})
		return
	}

	// Fetch users.
	dataArgs := append(args, limit, offset)
	rows, err := s.DB.Query(dataQuery, dataArgs...)
	if err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error"})
		return
	}
	defer rows.Close()

	type userEntry struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		Status    string `json:"status"`
		Plan      string `json:"plan"`
		CreatedAt string `json:"created_at"`
	}

	users := []userEntry{}
	for rows.Next() {
		var u userEntry
		var createdAt time.Time
		if err := rows.Scan(&u.ID, &u.Username, &u.Status, &u.Plan, &createdAt); err != nil {
			writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "scan_error"})
			return
		}
		u.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		users = append(users, u)
	}

	writeJSON(w, map[string]any{
		"ok":    true,
		"users": users,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}
