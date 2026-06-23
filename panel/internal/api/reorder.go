package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// adminReorder handles POST /api/admin/reorder
// Body: {"entity": "plans|nodes", "order": [3, 1, 2, 5, ...]}
// Updates the sort_order column in the relevant table.
func (s *Server) adminReorder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONCode(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
		return
	}

	limitBody(w, r, maxJSONBody)
	var in struct {
		Entity string  `json:"entity"`
		Order  []int64 `json:"order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bad_json"})
		return
	}

	// Validate entity
	var table string
	switch in.Entity {
	case "plans":
		table = "plans"
	case "nodes":
		table = "nodes"
	default:
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_entity"})
		return
	}

	if len(in.Order) == 0 {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "empty_order"})
		return
	}

	// Build a CASE WHEN statement for batch update
	// UPDATE <table> SET sort_order = CASE id WHEN 3 THEN 0 WHEN 1 THEN 1 ... END WHERE id IN (...)
	var caseBuilder strings.Builder
	args := make([]any, 0, len(in.Order)*2)
	placeholders := make([]string, 0, len(in.Order))

	caseBuilder.WriteString(fmt.Sprintf("UPDATE `%s` SET sort_order = CASE id ", table))
	for i, id := range in.Order {
		caseBuilder.WriteString("WHEN ? THEN ? ")
		args = append(args, id, i)
		placeholders = append(placeholders, "?")
	}
	caseBuilder.WriteString("ELSE sort_order END WHERE id IN (")
	caseBuilder.WriteString(strings.Join(placeholders, ","))
	caseBuilder.WriteString(")")

	// Add IDs again for the WHERE IN clause
	for _, id := range in.Order {
		args = append(args, id)
	}

	_, err := s.DB.Exec(caseBuilder.String(), args...)
	if err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error"})
		return
	}

	// Invalidate cached list so next fetch returns the new order
	if s.Cache != nil {
		s.Cache.InvalidatePrefix(table + ":")
	}

	writeJSON(w, map[string]any{"ok": true})
}
