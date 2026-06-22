package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"KorisLite/internal/auth"
)

func (s *Server) customers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listCustomers(w, r)
	case http.MethodPost:
		s.createCustomer(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, M{"ok": false, "error": "method"})
	}
}

func (s *Server) listCustomers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	limit := 50
	offset := (page - 1) * limit
	status := q.Get("status")
	search := q.Get("search")

	query := `SELECT id, username, status, COALESCE(data_limit_gb, 0), COALESCE(data_used_gb, 0), created_at FROM customers WHERE deleted_at IS NULL`
	countQuery := `SELECT COUNT(*) FROM customers WHERE deleted_at IS NULL`
	var args []any
	var countArgs []any

	if status != "" {
		query += ` AND status=?`
		countQuery += ` AND status=?`
		args = append(args, status)
		countArgs = append(countArgs, status)
	}
	if search != "" {
		query += ` AND username LIKE ?`
		countQuery += ` AND username LIKE ?`
		args = append(args, "%"+search+"%")
		countArgs = append(countArgs, "%"+search+"%")
	}

	var total int
	s.DB.QueryRow(countQuery, countArgs...).Scan(&total)

	query += ` ORDER BY id DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"ok": false, "error": "db_error"})
		return
	}
	defer rows.Close()

	type Customer struct {
		ID        int64   `json:"id"`
		Username  string  `json:"username"`
		Status    string  `json:"status"`
		DataLimit float64 `json:"data_limit_gb"`
		DataUsed  float64 `json:"data_used_gb"`
		CreatedAt string  `json:"created_at"`
	}

	customers := []Customer{}
	for rows.Next() {
		var c Customer
		var t time.Time
		if rows.Scan(&c.ID, &c.Username, &c.Status, &c.DataLimit, &c.DataUsed, &t) == nil {
			c.CreatedAt = t.Format(time.RFC3339)
			customers = append(customers, c)
		}
	}

	ok(w, M{"ok": true, "customers": customers, "total": total, "page": page})
}

func (s *Server) createCustomer(w http.ResponseWriter, r *http.Request) {
	limitBody(r)
	var in struct {
		Username    string  `json:"username"`
		Password    string  `json:"password"`
		DataLimitGB float64 `json:"data_limit_gb"`
		MaxSessions int     `json:"max_sessions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, M{"ok": false, "error": "bad_json"})
		return
	}
	if in.Username == "" || in.Password == "" {
		writeJSON(w, http.StatusBadRequest, M{"ok": false, "error": "username_and_password_required"})
		return
	}
	if in.DataLimitGB <= 0 {
		in.DataLimitGB = 50
	}
	if in.MaxSessions <= 0 {
		in.MaxSessions = 1
	}

	// Check uniqueness
	var exists int
	s.DB.QueryRow(`SELECT COUNT(*) FROM customers WHERE username=?`, in.Username).Scan(&exists)
	if exists > 0 {
		writeJSON(w, http.StatusConflict, M{"ok": false, "error": "username_exists"})
		return
	}

	// Hash password for RADIUS (cleartext stored for RADIUS CHAP compatibility — standard practice)
	// Also store bcrypt hash for portal login
	portalHash, _ := auth.HashPassword(in.Password)

	res, err := s.DB.Exec(`INSERT INTO customers (username, password, portal_password_hash, status, data_limit_gb, max_sessions, created_at)
		VALUES (?, ?, ?, 'active', ?, ?, NOW())`,
		in.Username, in.Password, portalHash, in.DataLimitGB, in.MaxSessions)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"ok": false, "error": "db_error"})
		return
	}

	// Also insert into radcheck for RADIUS auth
	s.DB.Exec(`INSERT INTO radcheck (username, attribute, op, value) VALUES (?, 'Cleartext-Password', ':=', ?)`,
		in.Username, in.Password)
	// Simultaneous-Use limit
	s.DB.Exec(`INSERT INTO radcheck (username, attribute, op, value) VALUES (?, 'Simultaneous-Use', ':=', ?)`,
		in.Username, strconv.Itoa(in.MaxSessions))

	id, _ := res.LastInsertId()
	ok(w, M{"ok": true, "id": id, "username": in.Username})
}

func (s *Server) customerByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/admin/customers/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, M{"ok": false, "error": "invalid_id"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getCustomer(w, id)
	case http.MethodPut:
		s.updateCustomer(w, r, id)
	case http.MethodDelete:
		s.deleteCustomer(w, id)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, M{"ok": false, "error": "method"})
	}
}

func (s *Server) getCustomer(w http.ResponseWriter, id int64) {
	var username, status string
	var dataLimit, dataUsed float64
	var maxSessions int
	var createdAt time.Time
	var expiresAt sql.NullTime

	err := s.DB.QueryRow(`SELECT username, status, COALESCE(data_limit_gb,0), COALESCE(data_used_gb,0), COALESCE(max_sessions,1), created_at, expires_at
		FROM customers WHERE id=? AND deleted_at IS NULL`, id).
		Scan(&username, &status, &dataLimit, &dataUsed, &maxSessions, &createdAt, &expiresAt)
	if err != nil {
		writeJSON(w, http.StatusNotFound, M{"ok": false, "error": "not_found"})
		return
	}

	customer := M{
		"id":            id,
		"username":      username,
		"status":        status,
		"data_limit_gb": dataLimit,
		"data_used_gb":  dataUsed,
		"max_sessions":  maxSessions,
		"created_at":    createdAt.Format(time.RFC3339),
	}
	if expiresAt.Valid {
		customer["expires_at"] = expiresAt.Time.Format(time.RFC3339)
	}

	ok(w, M{"ok": true, "customer": customer})
}

func (s *Server) updateCustomer(w http.ResponseWriter, r *http.Request, id int64) {
	limitBody(r)
	var in struct {
		Status      *string  `json:"status"`
		DataLimitGB *float64 `json:"data_limit_gb"`
		MaxSessions *int     `json:"max_sessions"`
		Password    *string  `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, M{"ok": false, "error": "bad_json"})
		return
	}

	// Get current username
	var username string
	if err := s.DB.QueryRow(`SELECT username FROM customers WHERE id=? AND deleted_at IS NULL`, id).Scan(&username); err != nil {
		writeJSON(w, http.StatusNotFound, M{"ok": false, "error": "not_found"})
		return
	}

	if in.Status != nil {
		s.DB.Exec(`UPDATE customers SET status=? WHERE id=?`, *in.Status, id)
	}
	if in.DataLimitGB != nil {
		s.DB.Exec(`UPDATE customers SET data_limit_gb=? WHERE id=?`, *in.DataLimitGB, id)
	}
	if in.MaxSessions != nil {
		s.DB.Exec(`UPDATE customers SET max_sessions=? WHERE id=?`, *in.MaxSessions, id)
		s.DB.Exec(`UPDATE radcheck SET value=? WHERE username=? AND attribute='Simultaneous-Use'`,
			fmt.Sprintf("%d", *in.MaxSessions), username)
	}
	if in.Password != nil && *in.Password != "" {
		portalHash, _ := auth.HashPassword(*in.Password)
		s.DB.Exec(`UPDATE customers SET password=?, portal_password_hash=? WHERE id=?`, *in.Password, portalHash, id)
		s.DB.Exec(`UPDATE radcheck SET value=? WHERE username=? AND attribute='Cleartext-Password'`, *in.Password, username)
	}

	ok(w, M{"ok": true})
}

func (s *Server) deleteCustomer(w http.ResponseWriter, id int64) {
	var username string
	if err := s.DB.QueryRow(`SELECT username FROM customers WHERE id=? AND deleted_at IS NULL`, id).Scan(&username); err != nil {
		writeJSON(w, http.StatusNotFound, M{"ok": false, "error": "not_found"})
		return
	}

	s.DB.Exec(`UPDATE customers SET deleted_at=NOW(), status='deleted' WHERE id=?`, id)
	s.DB.Exec(`DELETE FROM radcheck WHERE username=?`, username)
	s.DB.Exec(`DELETE FROM radreply WHERE username=?`, username)

	ok(w, M{"ok": true})
}
