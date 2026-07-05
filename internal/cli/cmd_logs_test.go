package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogsCommand_FormattedOutput(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/logs" {
			http.NotFound(w, r)
			return
		}
		// Verify tail parameter
		tail := r.URL.Query().Get("tail")
		if tail != "100" {
			t.Errorf("expected tail=100, got %s", tail)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"entries": []map[string]any{
				{
					"time":      "2024-01-15T10:30:00Z",
					"level":     "INFO",
					"component": "api",
					"message":   "request handled",
					"fields":    map[string]any{"path": "/api/nodes"},
				},
				{
					"time":      "2024-01-15T10:30:01Z",
					"level":     "WARN",
					"component": "db",
					"message":   "slow query detected",
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
		Name: "logs",
		Run: func(args []string) error {
			return runLogsWithURL(c, ts.URL, args)
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"logs"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	checks := []string{
		"2024-01-15T10:30:00Z",
		"[INFO]",
		"[api]",
		"request handled",
		"path=/api/nodes",
		"2024-01-15T10:30:01Z",
		"[WARN]",
		"[db]",
		"slow query detected",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("output missing %q\nGot:\n%s", check, output)
		}
	}
}

func TestLogsCommand_TailFlag(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/logs" {
			http.NotFound(w, r)
			return
		}
		tail := r.URL.Query().Get("tail")
		if tail != "50" {
			t.Errorf("expected tail=50, got %s", tail)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"entries": []map[string]any{
				{
					"time":      "2024-01-15T10:30:00Z",
					"level":     "DEBUG",
					"component": "worker",
					"message":   "tick",
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
		Name: "logs",
		Run: func(args []string) error {
			return runLogsWithURL(c, ts.URL, args)
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"logs", "--tail=50"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[DEBUG]") {
		t.Errorf("expected [DEBUG] in output, got:\n%s", output)
	}
	if !strings.Contains(output, "[worker]") {
		t.Errorf("expected [worker] in output, got:\n%s", output)
	}
}

func TestLogsCommand_JSONOutput(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/logs" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"entries": []map[string]any{
				{
					"time":      "2024-01-15T10:30:00Z",
					"level":     "ERROR",
					"component": "main",
					"message":   "something broke",
					"fields":    map[string]any{"code": float64(500)},
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
		Name: "logs",
		Run: func(args []string) error {
			return runLogsWithURL(c, ts.URL, args)
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"logs"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	var parsed map[string]any
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nGot: %s", err, output)
	}

	entries, ok := parsed["entries"].([]any)
	if !ok || len(entries) != 1 {
		t.Fatalf("expected 1 entry in JSON output, got %v", parsed["entries"])
	}

	entry := entries[0].(map[string]any)
	if entry["level"] != "ERROR" {
		t.Errorf("expected level ERROR, got %v", entry["level"])
	}
	if entry["component"] != "main" {
		t.Errorf("expected component 'main', got %v", entry["component"])
	}
}

func TestLogsCommand_EmptyEntries(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/logs" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"entries": []map[string]any{},
		})
	}))
	defer ts.Close()

	var buf bytes.Buffer
	c := New(
		WithOutput(&buf),
		WithSocketPath("/nonexistent/path.sock"),
	)

	cmd := Command{
		Name: "logs",
		Run: func(args []string) error {
			return runLogsWithURL(c, ts.URL, args)
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"logs"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No log entries found.") {
		t.Errorf("expected 'No log entries found.' message, got:\n%s", output)
	}
}

func TestLogsCommand_ConnectionError(t *testing.T) {
	var buf bytes.Buffer
	c := New(
		WithOutput(&buf),
		WithSocketPath("/nonexistent/path.sock"),
	)

	cmd := Command{
		Name: "logs",
		Run: func(args []string) error {
			return runLogsWithURL(c, "http://127.0.0.1:1", args)
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"logs"})
	if err == nil {
		t.Fatal("expected an error for unreachable panel")
	}
}

func TestLogsCommand_ServerError(t *testing.T) {
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
		Name: "logs",
		Run: func(args []string) error {
			return runLogsWithURL(c, ts.URL, args)
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"logs"})
	if err == nil {
		t.Fatal("expected an error for server error response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to contain status code 500, got: %v", err)
	}
}

func TestLogsCommand_InvalidTailFlag(t *testing.T) {
	var buf bytes.Buffer
	c := New(
		WithOutput(&buf),
		WithSocketPath("/nonexistent/path.sock"),
	)

	cmd := Command{
		Name: "logs",
		Run: func(args []string) error {
			return runLogsWithURL(c, "http://example.com", args)
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"logs", "--tail=abc"})
	if err == nil {
		t.Fatal("expected an error for invalid --tail value")
	}
	if !strings.Contains(err.Error(), "invalid --tail value") {
		t.Errorf("expected error about invalid --tail value, got: %v", err)
	}
}
