package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anonysec/koris/internal/config"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAdminImpersonateCustomer_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &Server{
		DB: db,
		Config: config.Config{
			SessionSecret: "test-secret-key-1234",
		},
	}

	// Customer lookup
	mock.ExpectQuery("SELECT username, deleted_at FROM customers WHERE").
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"username", "deleted_at"}).AddRow("testuser", nil))
	// Audit log
	mock.ExpectExec("INSERT INTO audit_logs").
		WillReturnResult(sqlmock.NewResult(1, 1))

	req := httptest.NewRequest(http.MethodPost, "/api/admin/customers/42/impersonate", nil)
	rec := httptest.NewRecorder()
	s.adminImpersonateCustomer(rec, req, 42)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["ok"] != true {
		t.Errorf("ok = %v, want true", resp["ok"])
	}
	if resp["portal_url"] == nil || resp["portal_url"] == "" {
		t.Error("portal_url should not be empty")
	}
	portalURL, _ := resp["portal_url"].(string)
	if len(portalURL) < 15 {
		t.Errorf("portal_url seems too short: %q", portalURL)
	}

	expiresIn, ok := resp["expires_in"].(float64)
	if !ok || expiresIn != float64(30*60) {
		t.Errorf("expires_in = %v, want %v", resp["expires_in"], 30*60)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestAdminImpersonateCustomer_DeletedCustomer(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &Server{
		DB: db,
		Config: config.Config{
			SessionSecret: "test-secret-key-1234",
		},
	}

	deletedTime := sql.NullTime{Time: time.Now().Add(-24 * time.Hour), Valid: true}
	mock.ExpectQuery("SELECT username, deleted_at FROM customers WHERE").
		WithArgs(int64(99)).
		WillReturnRows(sqlmock.NewRows([]string{"username", "deleted_at"}).AddRow("deleted_user", deletedTime))

	req := httptest.NewRequest(http.MethodPost, "/api/admin/customers/99/impersonate", nil)
	rec := httptest.NewRecorder()
	s.adminImpersonateCustomer(rec, req, 99)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["error"] != "customer_deleted" {
		t.Errorf("error = %v, want customer_deleted", resp["error"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestAdminImpersonateCustomer_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &Server{
		DB: db,
		Config: config.Config{
			SessionSecret: "test-secret-key-1234",
		},
	}

	mock.ExpectQuery("SELECT username, deleted_at FROM customers WHERE").
		WithArgs(int64(999)).
		WillReturnRows(sqlmock.NewRows([]string{"username", "deleted_at"})) // no rows

	req := httptest.NewRequest(http.MethodPost, "/api/admin/customers/999/impersonate", nil)
	rec := httptest.NewRecorder()
	s.adminImpersonateCustomer(rec, req, 999)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["error"] != "not_found" {
		t.Errorf("error = %v, want not_found", resp["error"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestAdminImpersonateCustomer_MethodNotAllowed(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/admin/customers/1/impersonate", nil)
	rec := httptest.NewRecorder()
	s.adminImpersonateCustomer(rec, req, 1)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestAdminImpersonateCustomer_TokenGeneration(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &Server{
		DB: db,
		Config: config.Config{
			SessionSecret: "my-secret-for-hmac",
		},
	}

	mock.ExpectQuery("SELECT username, deleted_at FROM customers WHERE").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"username", "deleted_at"}).AddRow("alice", nil))
	mock.ExpectExec("INSERT INTO audit_logs").
		WillReturnResult(sqlmock.NewResult(1, 1))

	req := httptest.NewRequest(http.MethodPost, "/api/admin/customers/7/impersonate", nil)
	rec := httptest.NewRecorder()
	s.adminImpersonateCustomer(rec, req, 7)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)

	portalURL, _ := resp["portal_url"].(string)
	if portalURL == "" {
		t.Fatal("portal_url is empty")
	}
	// portal_url should start with /portal/?token=
	if len(portalURL) < len("/portal/?token=") {
		t.Fatalf("portal_url too short: %q", portalURL)
	}
	prefix := "/portal/?token="
	if portalURL[:len(prefix)] != prefix {
		t.Errorf("portal_url = %q, should start with %q", portalURL, prefix)
	}
	// Token part should be non-empty
	token := portalURL[len(prefix):]
	if token == "" {
		t.Error("token part of portal_url is empty")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}
