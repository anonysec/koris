package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNodeBulk_MethodNotAllowed(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/admin/nodes/bulk", nil)
	rec := httptest.NewRecorder()
	s.nodeBulk(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestNodeBulk_InvalidJSON(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/nodes/bulk", strings.NewReader("{bad json"))
	rec := httptest.NewRecorder()
	s.nodeBulk(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["error"] != "bad_json" {
		t.Errorf("error = %v, want bad_json", resp["error"])
	}
}

func TestNodeBulk_InvalidAction(t *testing.T) {
	s := &Server{}
	body := `{"action":"destroy_all","node_ids":[1,2]}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/nodes/bulk", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.nodeBulk(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["error"] != "invalid_action" {
		t.Errorf("error = %v, want invalid_action", resp["error"])
	}
}

func TestNodeBulk_EmptyNodeIDs(t *testing.T) {
	s := &Server{}
	body := `{"action":"restart_openvpn","node_ids":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/nodes/bulk", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.nodeBulk(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["error"] != "node_ids_required" {
		t.Errorf("error = %v, want node_ids_required", resp["error"])
	}
}

func TestNodeBulk_TooManyNodes(t *testing.T) {
	s := &Server{}
	ids := make([]int64, 51)
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	idsJSON, _ := json.Marshal(ids)
	body := `{"action":"restart_openvpn","node_ids":` + string(idsJSON) + `}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/nodes/bulk", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.nodeBulk(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["error"] != "too_many_nodes" {
		t.Errorf("error = %v, want too_many_nodes", resp["error"])
	}
}

func TestNodeBulk_EnableProtocolMissingParam(t *testing.T) {
	s := &Server{}
	body := `{"action":"enable_protocol","node_ids":[1],"params":{}}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/nodes/bulk", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.nodeBulk(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["error"] != "protocol_required" {
		t.Errorf("error = %v, want protocol_required", resp["error"])
	}
}

func TestNodeBulk_RunCommandMissingParam(t *testing.T) {
	s := &Server{}
	body := `{"action":"run_command","node_ids":[1],"params":{}}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/nodes/bulk", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.nodeBulk(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["error"] != "command_required" {
		t.Errorf("error = %v, want command_required", resp["error"])
	}
}

func TestNodeBulk_RestartSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &Server{DB: db}

	// Node 1 exists
	mock.ExpectQuery("SELECT 1 FROM nodes WHERE id").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
	// Insert task for node 1
	mock.ExpectExec("INSERT INTO node_tasks").
		WillReturnResult(sqlmock.NewResult(10, 1))

	// Node 2 exists
	mock.ExpectQuery("SELECT 1 FROM nodes WHERE id").
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
	// Insert task for node 2
	mock.ExpectExec("INSERT INTO node_tasks").
		WillReturnResult(sqlmock.NewResult(11, 1))

	// Audit log
	mock.ExpectExec("INSERT INTO audit_logs").
		WillReturnResult(sqlmock.NewResult(1, 1))

	body := `{"action":"restart_openvpn","node_ids":[1,2]}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/nodes/bulk", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.nodeBulk(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["ok"] != true {
		t.Errorf("ok = %v, want true", resp["ok"])
	}

	results := resp["results"].([]any)
	if len(results) != 2 {
		t.Fatalf("results length = %d, want 2", len(results))
	}

	r1 := results[0].(map[string]any)
	if r1["success"] != true {
		t.Errorf("result[0].success = %v, want true", r1["success"])
	}
	if r1["task_id"].(float64) != 10 {
		t.Errorf("result[0].task_id = %v, want 10", r1["task_id"])
	}

	r2 := results[1].(map[string]any)
	if r2["success"] != true {
		t.Errorf("result[1].success = %v, want true", r2["success"])
	}
	if r2["task_id"].(float64) != 11 {
		t.Errorf("result[1].task_id = %v, want 11", r2["task_id"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestNodeBulk_PartialFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &Server{DB: db}

	// Node 1 exists
	mock.ExpectQuery("SELECT 1 FROM nodes WHERE id").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
	// Insert task for node 1 succeeds
	mock.ExpectExec("INSERT INTO node_tasks").
		WillReturnResult(sqlmock.NewResult(10, 1))

	// Node 999 does not exist
	mock.ExpectQuery("SELECT 1 FROM nodes WHERE id").
		WithArgs(int64(999)).
		WillReturnRows(sqlmock.NewRows([]string{"1"})) // no rows

	// Audit log
	mock.ExpectExec("INSERT INTO audit_logs").
		WillReturnResult(sqlmock.NewResult(1, 1))

	body := `{"action":"restart_all","node_ids":[1,999]}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/nodes/bulk", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.nodeBulk(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["ok"] != true {
		t.Errorf("ok = %v, want true", resp["ok"])
	}

	results := resp["results"].([]any)
	if len(results) != 2 {
		t.Fatalf("results length = %d, want 2", len(results))
	}

	// First node succeeds
	r1 := results[0].(map[string]any)
	if r1["success"] != true {
		t.Errorf("result[0].success = %v, want true", r1["success"])
	}

	// Second node fails (not found)
	r2 := results[1].(map[string]any)
	if r2["success"] != false {
		t.Errorf("result[1].success = %v, want false", r2["success"])
	}
	if r2["error"] != "node not found" {
		t.Errorf("result[1].error = %v, want 'node not found'", r2["error"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestNodeBulk_MaintenanceOn(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &Server{DB: db}

	// Node exists
	mock.ExpectQuery("SELECT 1 FROM nodes WHERE id").
		WithArgs(int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
	// Update maintenance mode
	mock.ExpectExec("UPDATE nodes SET maintenance_mode").
		WithArgs(true, int64(3)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Audit log
	mock.ExpectExec("INSERT INTO audit_logs").
		WillReturnResult(sqlmock.NewResult(1, 1))

	body := `{"action":"maintenance_on","node_ids":[3]}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/nodes/bulk", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.nodeBulk(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["ok"] != true {
		t.Errorf("ok = %v, want true", resp["ok"])
	}

	results := resp["results"].([]any)
	r1 := results[0].(map[string]any)
	if r1["success"] != true {
		t.Errorf("result[0].success = %v, want true", r1["success"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestNodeBulk_ValidActions(t *testing.T) {
	tests := []struct {
		name   string
		action string
		params string
	}{
		{"restart_openvpn", "restart_openvpn", "{}"},
		{"restart_all", "restart_all", "{}"},
		{"push_config", "push_config", "{}"},
		{"enable_protocol", "enable_protocol", `{"protocol":"wireguard"}`},
		{"disable_protocol", "disable_protocol", `{"protocol":"openvpn"}`},
		{"run_command", "run_command", `{"command":"uptime"}`},
		{"maintenance_on", "maintenance_on", "{}"},
		{"maintenance_off", "maintenance_off", "{}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create sqlmock: %v", err)
			}
			defer db.Close()

			s := &Server{DB: db}

			// Node exists
			mock.ExpectQuery("SELECT 1 FROM nodes WHERE id").
				WithArgs(int64(1)).
				WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

			if tt.action == "maintenance_on" || tt.action == "maintenance_off" {
				mock.ExpectExec("UPDATE nodes SET maintenance_mode").
					WillReturnResult(sqlmock.NewResult(0, 1))
			} else {
				mock.ExpectExec("INSERT INTO node_tasks").
					WillReturnResult(sqlmock.NewResult(1, 1))
			}

			// Audit log
			mock.ExpectExec("INSERT INTO audit_logs").
				WillReturnResult(sqlmock.NewResult(1, 1))

			body := `{"action":"` + tt.action + `","node_ids":[1],"params":` + tt.params + `}`
			req := httptest.NewRequest(http.MethodPost, "/api/admin/nodes/bulk", strings.NewReader(body))
			rec := httptest.NewRecorder()
			s.nodeBulk(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
			}

			var resp map[string]any
			json.NewDecoder(rec.Body).Decode(&resp)
			if resp["ok"] != true {
				t.Errorf("ok = %v, want true", resp["ok"])
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}
