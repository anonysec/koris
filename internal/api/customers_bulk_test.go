package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAdminCustomersBulk_InvalidAction(t *testing.T) {
	s := &Server{}
	body := `{"action":"nuke","customer_ids":[1,2]}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/customers/bulk", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.adminCustomersBulk(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["error"] != "invalid_action" {
		t.Errorf("error = %v, want invalid_action", resp["error"])
	}
}

func TestAdminCustomersBulk_EmptyIDs(t *testing.T) {
	s := &Server{}
	body := `{"action":"disable","customer_ids":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/customers/bulk", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.adminCustomersBulk(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["error"] != "customer_ids_required" {
		t.Errorf("error = %v, want customer_ids_required", resp["error"])
	}
}

func TestAdminCustomersBulk_TooManyIDs(t *testing.T) {
	s := &Server{}
	ids := make([]int64, 101)
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	idsJSON, _ := json.Marshal(ids)
	body := `{"action":"disable","customer_ids":` + string(idsJSON) + `}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/customers/bulk", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.adminCustomersBulk(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["error"] != "too_many_customers" {
		t.Errorf("error = %v, want too_many_customers", resp["error"])
	}
}

func TestAdminCustomersBulk_MethodNotAllowed(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/admin/customers/bulk", nil)
	rec := httptest.NewRecorder()
	s.adminCustomersBulk(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestAdminCustomersBulk_DisableSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &Server{DB: db}

	// customerUsername lookup for ID 1
	mock.ExpectQuery("SELECT username FROM customers WHERE").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"username"}).AddRow("user1"))
	// disable customer
	mock.ExpectExec("UPDATE customers SET status='disabled'").
		WillReturnResult(sqlmock.NewResult(0, 1))
	// disconnectCustomerSessions query (returns no active sessions)
	mock.ExpectQuery("SELECT radacctid, acctsessionid").
		WillReturnRows(sqlmock.NewRows([]string{"radacctid", "acctsessionid", "nasipaddress"}))
	// audit log for individual action
	mock.ExpectExec("INSERT INTO audit_logs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	// bulk audit log
	mock.ExpectExec("INSERT INTO audit_logs").
		WillReturnResult(sqlmock.NewResult(2, 1))

	body := `{"action":"disable","customer_ids":[1]}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/customers/bulk", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.adminCustomersBulk(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["ok"] != true {
		t.Errorf("ok = %v, want true", resp["ok"])
	}
	if resp["affected"].(float64) != 1 {
		t.Errorf("affected = %v, want 1", resp["affected"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestAdminCustomersBulk_EnableSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &Server{DB: db}

	// customerUsername lookup
	mock.ExpectQuery("SELECT username FROM customers WHERE").
		WithArgs(int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"username"}).AddRow("user5"))
	// enable customer
	mock.ExpectExec("UPDATE customers SET status='active'").
		WillReturnResult(sqlmock.NewResult(0, 1))
	// audit log for action
	mock.ExpectExec("INSERT INTO audit_logs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	// bulk audit log
	mock.ExpectExec("INSERT INTO audit_logs").
		WillReturnResult(sqlmock.NewResult(2, 1))

	body := `{"action":"enable","customer_ids":[5]}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/customers/bulk", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.adminCustomersBulk(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["ok"] != true {
		t.Errorf("ok = %v, want true", resp["ok"])
	}
	if resp["affected"].(float64) != 1 {
		t.Errorf("affected = %v, want 1", resp["affected"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestAdminCustomersBulk_ExtendMissingDays(t *testing.T) {
	s := &Server{}
	body := `{"action":"extend","customer_ids":[1],"params":{}}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/customers/bulk", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.adminCustomersBulk(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["error"] != "params_days_required" {
		t.Errorf("error = %v, want params_days_required", resp["error"])
	}
}

func TestAdminCustomersBulk_ExtendSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &Server{DB: db}

	// customerUsername lookup
	mock.ExpectQuery("SELECT username FROM customers WHERE").
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"username"}).AddRow("user10"))
	// extend subscription
	mock.ExpectExec("UPDATE subscriptions SET expires_at").
		WillReturnResult(sqlmock.NewResult(0, 1))
	// audit log for action
	mock.ExpectExec("INSERT INTO audit_logs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	// bulk audit log
	mock.ExpectExec("INSERT INTO audit_logs").
		WillReturnResult(sqlmock.NewResult(2, 1))

	body := `{"action":"extend","customer_ids":[10],"params":{"days":30}}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/customers/bulk", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.adminCustomersBulk(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["ok"] != true {
		t.Errorf("ok = %v, want true", resp["ok"])
	}
	if resp["affected"].(float64) != 1 {
		t.Errorf("affected = %v, want 1", resp["affected"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestAdminCustomersBulk_DeleteSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &Server{DB: db}

	// customerUsername lookup
	mock.ExpectQuery("SELECT username FROM customers WHERE").
		WithArgs(int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"username"}).AddRow("user3"))
	// soft-delete customer
	mock.ExpectExec("UPDATE customers SET deleted_at").
		WillReturnResult(sqlmock.NewResult(0, 1))
	// disconnectCustomerSessions
	mock.ExpectQuery("SELECT radacctid, acctsessionid").
		WillReturnRows(sqlmock.NewRows([]string{"radacctid", "acctsessionid", "nasipaddress"}))
	// audit log for action
	mock.ExpectExec("INSERT INTO audit_logs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	// bulk audit log
	mock.ExpectExec("INSERT INTO audit_logs").
		WillReturnResult(sqlmock.NewResult(2, 1))

	body := `{"action":"delete","customer_ids":[3]}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/customers/bulk", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.adminCustomersBulk(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["ok"] != true {
		t.Errorf("ok = %v, want true", resp["ok"])
	}
	if resp["affected"].(float64) != 1 {
		t.Errorf("affected = %v, want 1", resp["affected"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestToPositiveInt(t *testing.T) {
	tests := []struct {
		name      string
		input     any
		wantVal   int
		wantValid bool
	}{
		{"float64 positive", float64(5), 5, true},
		{"float64 zero", float64(0), 0, false},
		{"float64 negative", float64(-3), 0, false},
		{"float64 fractional", float64(2.5), 0, false},
		{"string positive", "10", 10, true},
		{"string zero", "0", 0, false},
		{"string negative", "-1", 0, false},
		{"string non-numeric", "abc", 0, false},
		{"json.Number valid", json.Number("7"), 7, true},
		{"json.Number zero", json.Number("0"), 0, false},
		{"json.Number negative", json.Number("-5"), 0, false},
		{"json.Number float", json.Number("3.14"), 0, false},
		{"nil input", nil, 0, false},
		{"bool input", true, 0, false},
		{"slice input", []int{1}, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, valid := toPositiveInt(tt.input)
			if val != tt.wantVal || valid != tt.wantValid {
				t.Errorf("toPositiveInt(%v) = (%d, %v), want (%d, %v)", tt.input, val, valid, tt.wantVal, tt.wantValid)
			}
		})
	}
}
