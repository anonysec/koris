package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestInternalWorkers_GET(t *testing.T) {
	s := &Server{}

	req := httptest.NewRequest(http.MethodGet, "/internal/workers", nil)
	rec := httptest.NewRecorder()

	s.internalWorkers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp struct {
		OK      bool `json:"ok"`
		Workers []struct {
			ID            int    `json:"id"`
			PID           int    `json:"pid"`
			Status        string `json:"status"`
			UptimeSeconds int64  `json:"uptime_seconds"`
			Restarts      int    `json:"restarts"`
		} `json:"workers"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.OK {
		t.Fatal("expected ok=true")
	}
	if len(resp.Workers) != 1 {
		t.Fatalf("expected 1 worker, got %d", len(resp.Workers))
	}

	w := resp.Workers[0]
	if w.ID != 1 {
		t.Errorf("expected worker ID=1, got %d", w.ID)
	}
	if w.PID != os.Getpid() {
		t.Errorf("expected PID=%d, got %d", os.Getpid(), w.PID)
	}
	if w.Status != "running" {
		t.Errorf("expected status=running, got %q", w.Status)
	}
	if w.UptimeSeconds < 0 {
		t.Errorf("expected non-negative uptime, got %d", w.UptimeSeconds)
	}
	if w.Restarts != 0 {
		t.Errorf("expected restarts=0, got %d", w.Restarts)
	}
}

func TestInternalWorkers_MethodNotAllowed(t *testing.T) {
	s := &Server{}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/internal/workers", nil)
			rec := httptest.NewRecorder()

			s.internalWorkers(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status 405 for %s, got %d", method, rec.Code)
			}
		})
	}
}

func TestInternalWorkers_JSONFormat(t *testing.T) {
	s := &Server{}

	req := httptest.NewRequest(http.MethodGet, "/internal/workers", nil)
	rec := httptest.NewRecorder()

	s.internalWorkers(rec, req)

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type=application/json, got %q", ct)
	}

	// Verify we can decode as generic map to check structure
	var raw map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&raw); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	// Verify top-level keys
	if _, exists := raw["ok"]; !exists {
		t.Error("missing 'ok' field in response")
	}
	if _, exists := raw["workers"]; !exists {
		t.Error("missing 'workers' field in response")
	}
}
