package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWorkersCommand_TableOutput(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/workers" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"workers": []map[string]any{
				{
					"id":             1,
					"pid":            12345,
					"status":         "running",
					"uptime_seconds": 3661,
					"restarts":       0,
				},
				{
					"id":             2,
					"pid":            12346,
					"status":         "running",
					"uptime_seconds": 86520,
					"restarts":       2,
				},
			},
		})
	}))
	defer ts.Close()

	var buf bytes.Buffer
	c := New(
		WithOutput(&buf),
		WithSocketPath("/nonexistent/path.sock"),
	)

	cmd := Command{
		Name: "workers",
		Run: func(args []string) error {
			return runWorkersWithURL(c, ts.URL)
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"workers"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	checks := []string{
		"PID",
		"Status",
		"Uptime",
		"Restart Count",
		"12345",
		"running",
		"1h 1m",
		"0",
		"12346",
		"1d 0h 2m",
		"2",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("output missing %q\nGot:\n%s", check, output)
		}
	}
}

func TestWorkersCommand_JSONOutput(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/workers" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"workers": []map[string]any{
				{
					"id":             1,
					"pid":            9999,
					"status":         "running",
					"uptime_seconds": 120,
					"restarts":       1,
				},
			},
		})
	}))
	defer ts.Close()

	var buf bytes.Buffer
	c := New(
		WithOutput(&buf),
		WithJSONOutput(true),
		WithSocketPath("/nonexistent/path.sock"),
	)

	cmd := Command{
		Name: "workers",
		Run: func(args []string) error {
			return runWorkersWithURL(c, ts.URL)
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"workers"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	var parsed map[string]any
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nGot: %s", err, output)
	}

	workers, ok := parsed["workers"].([]any)
	if !ok || len(workers) != 1 {
		t.Fatalf("expected 1 worker in JSON output, got %v", parsed["workers"])
	}

	worker := workers[0].(map[string]any)
	if int(worker["pid"].(float64)) != 9999 {
		t.Errorf("expected worker PID 9999, got %v", worker["pid"])
	}
	if worker["status"] != "running" {
		t.Errorf("expected worker status 'running', got %v", worker["status"])
	}
}

func TestWorkersCommand_EmptyList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/workers" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"workers": []map[string]any{},
		})
	}))
	defer ts.Close()

	var buf bytes.Buffer
	c := New(
		WithOutput(&buf),
		WithSocketPath("/nonexistent/path.sock"),
	)

	cmd := Command{
		Name: "workers",
		Run: func(args []string) error {
			return runWorkersWithURL(c, ts.URL)
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"workers"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No workers found.") {
		t.Errorf("expected 'No workers found.' message, got:\n%s", output)
	}
}

func TestWorkersCommand_ConnectionError(t *testing.T) {
	var buf bytes.Buffer
	c := New(
		WithOutput(&buf),
		WithSocketPath("/nonexistent/path.sock"),
	)

	cmd := Command{
		Name: "workers",
		Run: func(args []string) error {
			return runWorkersWithURL(c, "http://127.0.0.1:1")
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"workers"})
	if err == nil {
		t.Fatal("expected an error for unreachable panel")
	}
}

func TestWorkersCommand_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"ok":false,"error":"internal_error"}`))
	}))
	defer ts.Close()

	var buf bytes.Buffer
	c := New(
		WithOutput(&buf),
		WithSocketPath("/nonexistent/path.sock"),
	)

	cmd := Command{
		Name: "workers",
		Run: func(args []string) error {
			return runWorkersWithURL(c, ts.URL)
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"workers"})
	if err == nil {
		t.Fatal("expected an error for server error response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to contain status code 500, got: %v", err)
	}
}

func TestFormatWorkerUptime(t *testing.T) {
	tests := []struct {
		name     string
		seconds  int64
		expected string
	}{
		{"zero", 0, "0m"},
		{"minutes only", 300, "5m"},
		{"hours and minutes", 3661, "1h 1m"},
		{"days hours minutes", 86520, "1d 0h 2m"},
		{"multiple days", 172800, "2d 0h 0m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatWorkerUptime(tt.seconds)
			if result != tt.expected {
				t.Errorf("formatWorkerUptime(%d) = %q, want %q", tt.seconds, result, tt.expected)
			}
		})
	}
}
