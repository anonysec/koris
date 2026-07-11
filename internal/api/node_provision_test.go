package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/anonysec/koris/internal/testutil"
)

func TestNodeProvision_MethodNotAllowed(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodPut, "/api/admin/nodes/provision", nil)
	rr := httptest.NewRecorder()
	s.nodeProvision(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestNodeProvision_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &Server{DB: db}

	// Expect INSERT INTO nodes
	mock.ExpectExec("INSERT INTO nodes").
		WillReturnResult(sqlmock.NewResult(5, 1))

	// Expect panel_settings query for getPanelURL (may return no rows)
	mock.ExpectQuery("SELECT setting_value FROM panel_settings").
		WillReturnRows(sqlmock.NewRows([]string{"setting_value"}))

	// Expect audit log INSERT
	mock.ExpectExec("INSERT INTO audit_logs").
		WillReturnResult(sqlmock.NewResult(1, 1))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/nodes/provision?name=test-node", nil)
	req.Host = "panel.example.com"
	rr := httptest.NewRecorder()
	s.nodeProvision(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["ok"] != true {
		t.Errorf("expected ok=true, got %v", resp["ok"])
	}
	if resp["node_id"] != float64(5) {
		t.Errorf("expected node_id=5, got %v", resp["node_id"])
	}
	token, ok := resp["token"].(string)
	if !ok || token == "" {
		t.Errorf("expected non-empty token, got %v", resp["token"])
	}
	if token[:3] != "kn_" {
		t.Errorf("expected token to start with 'kn_', got %s", token)
	}
	installCmd, ok := resp["install_command"].(string)
	if !ok || installCmd == "" {
		t.Errorf("expected non-empty install_command, got %v", resp["install_command"])
	}
	installURL, ok := resp["install_url"].(string)
	if !ok || installURL == "" {
		t.Errorf("expected non-empty install_url, got %v", resp["install_url"])
	}
}

func TestNodeProvision_WithProtocols(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &Server{DB: db}

	// Expect INSERT INTO nodes
	mock.ExpectExec("INSERT INTO nodes").
		WillReturnResult(sqlmock.NewResult(7, 1))

	// Expect tag inserts for each protocol
	mock.ExpectExec("INSERT INTO node_tags").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO node_tags").WillReturnResult(sqlmock.NewResult(2, 1))

	// Expect panel_settings query
	mock.ExpectQuery("SELECT setting_value FROM panel_settings").
		WillReturnRows(sqlmock.NewRows([]string{"setting_value"}).AddRow("panel.mysite.com"))

	// Expect audit log
	mock.ExpectExec("INSERT INTO audit_logs").
		WillReturnResult(sqlmock.NewResult(1, 1))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/nodes/provision?name=vpn-node&protocols=openvpn,wireguard", nil)
	rr := httptest.NewRecorder()
	s.nodeProvision(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["ok"] != true {
		t.Errorf("expected ok=true, got %v", resp["ok"])
	}
	if resp["node_id"] != float64(7) {
		t.Errorf("expected node_id=7, got %v", resp["node_id"])
	}

	// Verify install URL contains panel domain from settings
	installURL, _ := resp["install_url"].(string)
	if installURL == "" {
		t.Error("expected non-empty install_url")
	}
}

func TestNodeProvision_NameTooLong(t *testing.T) {
	s := &Server{}
	longName := "a"
	for i := 0; i < 65; i++ {
		longName += "x"
	}
	req := httptest.NewRequest(http.MethodGet, "/api/admin/nodes/provision?name="+longName, nil)
	rr := httptest.NewRecorder()
	s.nodeProvision(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestNodeInstallScript_GET(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/node/install.sh", nil)
	rr := httptest.NewRecorder()
	s.nodeInstallScript(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "text/x-shellscript; charset=utf-8" {
		t.Errorf("expected shell content type, got %s", contentType)
	}

	body := rr.Body.String()
	if body == "" {
		t.Error("expected non-empty script body")
	}
	if !testutil.Contains(body, "#!/usr/bin/env bash") {
		t.Error("expected script to start with bash shebang")
	}
	if !testutil.Contains(body, "PANEL_URL") {
		t.Error("expected script to reference PANEL_URL")
	}
	if !testutil.Contains(body, "NODE_TOKEN") {
		t.Error("expected script to reference NODE_TOKEN")
	}
}

func TestNodeInstallScript_WithQueryParams(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/node/install.sh?token=abc123&panel_url=https://my.panel.com", nil)
	rr := httptest.NewRecorder()
	s.nodeInstallScript(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	body := rr.Body.String()
	if !testutil.Contains(body, "abc123") {
		t.Error("expected script to contain embedded token")
	}
	if !testutil.Contains(body, "https://my.panel.com") {
		t.Error("expected script to contain embedded panel URL")
	}
}

func TestNodeInstallScript_MethodNotAllowed(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/node/install.sh", nil)
	rr := httptest.NewRecorder()
	s.nodeInstallScript(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestGenerateInstallScript_NoParams(t *testing.T) {
	script := generateInstallScript("", "")
	if !testutil.Contains(script, "#!/usr/bin/env bash") {
		t.Error("expected bash shebang")
	}
	if !testutil.Contains(script, `NODE_TOKEN="${NODE_TOKEN:-}"`) {
		t.Error("expected empty NODE_TOKEN default when no token provided")
	}
	if !testutil.Contains(script, `PANEL_URL="${PANEL_URL:-}"`) {
		t.Error("expected empty PANEL_URL default when no URL provided")
	}
}

func TestGenerateInstallScript_WithParams(t *testing.T) {
	script := generateInstallScript("kn_test123", "https://panel.test.com")
	if !testutil.Contains(script, "kn_test123") {
		t.Error("expected token in script")
	}
	if !testutil.Contains(script, "https://panel.test.com") {
		t.Error("expected panel URL in script")
	}
}

func TestGetPanelURL_FromSettings(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &Server{DB: db}

	mock.ExpectQuery("SELECT setting_value FROM panel_settings").
		WillReturnRows(sqlmock.NewRows([]string{"setting_value"}).AddRow("my.panel.com"))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Host = "fallback.host.com"

	url := s.getPanelURL(req)
	if url != "https://my.panel.com" {
		t.Errorf("expected https://my.panel.com, got %s", url)
	}
}

func TestGetPanelURL_FallbackToHost(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &Server{DB: db}

	// Return empty rows (no domain configured)
	mock.ExpectQuery("SELECT setting_value FROM panel_settings").
		WillReturnRows(sqlmock.NewRows([]string{"setting_value"}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Host = "fallback.host.com"
	req.Header.Set("X-Forwarded-Proto", "https")

	url := s.getPanelURL(req)
	if url != "https://fallback.host.com" {
		t.Errorf("expected https://fallback.host.com, got %s", url)
	}
}
