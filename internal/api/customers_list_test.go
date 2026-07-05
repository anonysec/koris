package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anonysec/koris/internal/config"

	"github.com/DATA-DOG/go-sqlmock"
)

// Note: listCustomers calls currentAdmin internally. Without a valid session cookie,
// currentAdmin returns ("", "", false) — no reseller scoping applied.

func TestListCustomers_DefaultPagination(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	s := &Server{DB: db, Config: config.Config{SessionSecret: "test"}}

	// Count query
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM customers c`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// Data query
	rows := sqlmock.NewRows([]string{"id", "username", "display_name", "status", "plan_id", "plan_name", "credit", "created_by", "avatar", "created_at"}).
		AddRow(1, "user1", "User One", "active", 1, "Basic", 10.0, "admin", "", time.Now()).
		AddRow(2, "user2", "User Two", "expired", 2, "Pro", 0.0, "admin", "", time.Now())
	mock.ExpectQuery(`SELECT c\.id,c\.username`).WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/customers", nil)
	rec := httptest.NewRecorder()
	s.listCustomers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["ok"] != true {
		t.Errorf("ok = %v, want true", resp["ok"])
	}
	if int(resp["total"].(float64)) != 2 {
		t.Errorf("total = %v, want 2", resp["total"])
	}
	if int(resp["page"].(float64)) != 1 {
		t.Errorf("page = %v, want 1", resp["page"])
	}
	if int(resp["limit"].(float64)) != 20 {
		t.Errorf("limit = %v, want 20", resp["limit"])
	}
	customers := resp["customers"].([]any)
	if len(customers) != 2 {
		t.Errorf("customers count = %d, want 2", len(customers))
	}
}

func TestListCustomers_PaginationLimitCap(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	s := &Server{DB: db, Config: config.Config{SessionSecret: "test"}}

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM customers c`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT c\.id,c\.username`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "display_name", "status", "plan_id", "plan_name", "credit", "created_by", "avatar", "created_at"}))

	// Request with limit > 100 should be capped
	req := httptest.NewRequest(http.MethodGet, "/api/customers?limit=500&page=3", nil)
	rec := httptest.NewRecorder()
	s.listCustomers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if int(resp["limit"].(float64)) != 100 {
		t.Errorf("limit = %v, want 100 (capped)", resp["limit"])
	}
	if int(resp["page"].(float64)) != 3 {
		t.Errorf("page = %v, want 3", resp["page"])
	}
}

func TestListCustomers_SearchParam(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	s := &Server{DB: db, Config: config.Config{SessionSecret: "test"}}

	// Count query — expect LIKE params for search
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM customers c`).
		WithArgs("%john%", "%john%", "%john%", "%john%", "%john%", "%john%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{"id", "username", "display_name", "status", "plan_id", "plan_name", "credit", "created_by", "avatar", "created_at"}).
		AddRow(1, "john_doe", "John Doe", "active", 1, "Basic", 5.0, "admin", "", time.Now())
	mock.ExpectQuery(`SELECT c\.id,c\.username`).
		WithArgs("%john%", "%john%", "%john%", "%john%", "%john%", "%john%", 20, 0).
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/customers?search=john", nil)
	rec := httptest.NewRecorder()
	s.listCustomers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["ok"] != true {
		t.Errorf("ok = %v, want true", resp["ok"])
	}
	if int(resp["total"].(float64)) != 1 {
		t.Errorf("total = %v, want 1", resp["total"])
	}
}

func TestListCustomers_StatusFilter(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	s := &Server{DB: db, Config: config.Config{SessionSecret: "test"}}

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM customers c`).
		WithArgs("active").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	mock.ExpectQuery(`SELECT c\.id,c\.username`).
		WithArgs("active", 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "display_name", "status", "plan_id", "plan_name", "credit", "created_by", "avatar", "created_at"}))

	req := httptest.NewRequest(http.MethodGet, "/api/customers?status=active", nil)
	rec := httptest.NewRecorder()
	s.listCustomers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if int(resp["total"].(float64)) != 5 {
		t.Errorf("total = %v, want 5", resp["total"])
	}
}

func TestBuildCustomerSortClause_Default(t *testing.T) {
	s := &Server{}
	result := s.buildCustomerSortClause("")
	if result != "c.created_at DESC" {
		t.Errorf("sort = %q, want %q", result, "c.created_at DESC")
	}
}

func TestBuildCustomerSortClause_Single(t *testing.T) {
	s := &Server{}
	result := s.buildCustomerSortClause("username:asc")
	if result != "c.username ASC" {
		t.Errorf("sort = %q, want %q", result, "c.username ASC")
	}
}

func TestBuildCustomerSortClause_Multi(t *testing.T) {
	s := &Server{}
	result := s.buildCustomerSortClause("username:asc,created_at:desc")
	expected := "c.username ASC, c.created_at DESC"
	if result != expected {
		t.Errorf("sort = %q, want %q", result, expected)
	}
}

func TestBuildCustomerSortClause_InvalidField(t *testing.T) {
	s := &Server{}
	// Invalid fields should be ignored, falling back to default
	result := s.buildCustomerSortClause("hacker:asc")
	if result != "c.created_at DESC" {
		t.Errorf("sort = %q, want %q (default fallback)", result, "c.created_at DESC")
	}
}

func TestBuildCustomerSortClause_MixedValidInvalid(t *testing.T) {
	s := &Server{}
	// Mix of valid and invalid: only valid fields should appear
	result := s.buildCustomerSortClause("username:desc,hacker:asc,status:asc")
	expected := "c.username DESC, c.status ASC"
	if result != expected {
		t.Errorf("sort = %q, want %q", result, expected)
	}
}

func TestBuildCustomerSortClause_DefaultDirection(t *testing.T) {
	s := &Server{}
	// No direction specified => defaults to ASC
	result := s.buildCustomerSortClause("username")
	if result != "c.username ASC" {
		t.Errorf("sort = %q, want %q", result, "c.username ASC")
	}
}

func TestBuildCustomerSortClause_SQLInjectionPrevented(t *testing.T) {
	s := &Server{}
	// Attempting SQL injection via sort param
	result := s.buildCustomerSortClause("username; DROP TABLE customers--:asc")
	// Should fall back to default since the field name isn't in allowlist
	if result != "c.created_at DESC" {
		t.Errorf("sort = %q, want default (SQL injection attempt blocked)", result)
	}
}
