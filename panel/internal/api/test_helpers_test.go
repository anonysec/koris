package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// boolPtr returns a pointer to a bool value (test helper).
func boolPtr(v bool) *bool {
	return &v
}

// assertStatus checks the HTTP status code matches expected.
func assertStatus(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("status = %d, want %d", got, want)
	}
}

// assertOK checks the "ok" field in the JSON response.
func assertOK(t *testing.T, rr *httptest.ResponseRecorder, want bool) {
	t.Helper()
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v, body: %s", err, rr.Body.String())
	}
	if resp["ok"] != want {
		t.Errorf("ok = %v, want %v", resp["ok"], want)
	}
}

// assertErrorCode checks the "error" field in the JSON response.
func assertErrorCode(t *testing.T, rr *httptest.ResponseRecorder, want string) {
	t.Helper()
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v, body: %s", err, rr.Body.String())
	}
	if resp["error"] != want {
		t.Errorf("error = %v, want %q", resp["error"], want)
	}
}
