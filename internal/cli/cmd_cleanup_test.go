package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCleanupCommand_DryRun_TableOutput(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/internal/cleanup" {
			http.NotFound(w, r)
			return
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		if body["dry_run"] != true {
			t.Errorf("expected dry_run=true, got %v", body["dry_run"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"dry_run": true,
			"results": []map[string]any{
				{"target": "stale_sessions", "row_count": 150, "oldest": "2024-01-01T00:00:00Z"},
				{"target": "old_events", "row_count": 2300, "oldest": "2023-06-15T00:00:00Z"},
			},
		})
	}))
	defer ts.Close()

	var buf bytes.Buffer
	c := New(WithOutput(&buf), WithSocketPath("/nonexistent/path.sock"))

	flags := map[string]string{"dry-run": ""}
	err := runCleanupWithURL(c, ts.URL, flags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	checks := []string{
		"Cleanup Preview (dry-run)",
		"Target",
		"Row Count",
		"Oldest Record",
		"stale_sessions",
		"150",
		"2024-01-01T00:00:00Z",
		"old_events",
		"2300",
		"2023-06-15T00:00:00Z",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("output missing %q\nGot:\n%s", check, output)
		}
	}
}

func TestCleanupCommand_Confirm_TableOutput(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		if body["confirm"] != true {
			t.Errorf("expected confirm=true, got %v", body["confirm"])
		}
		if body["older_than"] != "90d" {
			t.Errorf("expected older_than=90d, got %v", body["older_than"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"dry_run": false,
			"results": []map[string]any{
				{"target": "stale_sessions", "row_count": 150, "oldest": "2024-01-01T00:00:00Z"},
			},
		})
	}))
	defer ts.Close()

	var buf bytes.Buffer
	c := New(WithOutput(&buf), WithSocketPath("/nonexistent/path.sock"))

	flags := map[string]string{"confirm": "", "older-than": "90d"}
	err := runCleanupWithURL(c, ts.URL, flags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Cleanup Results") {
		t.Errorf("expected 'Cleanup Results' header, got:\n%s", output)
	}
	if !strings.Contains(output, "Rows Deleted") {
		t.Errorf("expected 'Rows Deleted' column, got:\n%s", output)
	}
}

func TestCleanupCommand_WithTargets(t *testing.T) {
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		json.Unmarshal(bodyBytes, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"dry_run": false,
			"results": []map[string]any{
				{"target": "stale_sessions", "row_count": 50, "oldest": "2024-03-01T00:00:00Z"},
			},
		})
	}))
	defer ts.Close()

	var buf bytes.Buffer
	c := New(WithOutput(&buf), WithSocketPath("/nonexistent/path.sock"))

	flags := map[string]string{
		"confirm":    "",
		"older-than": "30d",
		"targets":    "stale_sessions,old_events",
	}
	err := runCleanupWithURL(c, ts.URL, flags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	targets, ok := receivedBody["targets"].([]any)
	if !ok {
		t.Fatalf("expected targets array in request body, got %v", receivedBody["targets"])
	}
	if len(targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(targets))
	}
	if targets[0] != "stale_sessions" || targets[1] != "old_events" {
		t.Errorf("unexpected targets: %v", targets)
	}
}

func TestCleanupCommand_JSONOutput(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"dry_run": true,
			"results": []map[string]any{
				{"target": "stale_sessions", "row_count": 150, "oldest": "2024-01-01T00:00:00Z"},
			},
		})
	}))
	defer ts.Close()

	var buf bytes.Buffer
	c := New(WithOutput(&buf), WithJSONOutput(true), WithSocketPath("/nonexistent/path.sock"))

	flags := map[string]string{"dry-run": ""}
	err := runCleanupWithURL(c, ts.URL, flags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	var parsed map[string]any
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nGot: %s", err, output)
	}

	if parsed["ok"] != true {
		t.Errorf("expected ok=true, got %v", parsed["ok"])
	}
	if parsed["dry_run"] != true {
		t.Errorf("expected dry_run=true, got %v", parsed["dry_run"])
	}

	results, ok := parsed["results"].([]any)
	if !ok || len(results) != 1 {
		t.Fatalf("expected 1 result in JSON output, got %v", parsed["results"])
	}
}

func TestCleanupCommand_NoFlagsError(t *testing.T) {
	var buf bytes.Buffer
	c := New(WithOutput(&buf), WithSocketPath("/nonexistent/path.sock"))

	flags := map[string]string{}
	err := runCleanupWithURL(c, "http://localhost:1", flags)
	if err == nil {
		t.Fatal("expected an error when no flags provided")
	}
	if !strings.Contains(err.Error(), "--dry-run") || !strings.Contains(err.Error(), "--confirm") {
		t.Errorf("expected error mentioning --dry-run and --confirm, got: %v", err)
	}
}

func TestCleanupCommand_InvalidOlderThan(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"no suffix", "90"},
		{"wrong suffix", "90h"},
		{"zero days", "0d"},
		{"negative", "-5d"},
		{"not a number", "abcd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			c := New(WithOutput(&buf), WithSocketPath("/nonexistent/path.sock"))

			flags := map[string]string{"confirm": "", "older-than": tt.value}
			err := runCleanupWithURL(c, "http://localhost:1", flags)
			if err == nil {
				t.Fatal("expected an error for invalid --older-than")
			}
			if !strings.Contains(err.Error(), "invalid --older-than") {
				t.Errorf("expected invalid format error, got: %v", err)
			}
		})
	}
}

func TestCleanupCommand_ValidOlderThanFormats(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"dry_run": true,
			"results": []map[string]any{},
		})
	}))
	defer ts.Close()

	tests := []string{"1d", "30d", "90d", "365d"}
	for _, val := range tests {
		t.Run(val, func(t *testing.T) {
			var buf bytes.Buffer
			c := New(WithOutput(&buf), WithSocketPath("/nonexistent/path.sock"))

			flags := map[string]string{"dry-run": "", "older-than": val}
			err := runCleanupWithURL(c, ts.URL, flags)
			if err != nil {
				t.Fatalf("unexpected error for older-than=%s: %v", val, err)
			}
		})
	}
}

func TestCleanupCommand_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"ok":false,"error":"db_error"}`))
	}))
	defer ts.Close()

	var buf bytes.Buffer
	c := New(WithOutput(&buf), WithSocketPath("/nonexistent/path.sock"))

	flags := map[string]string{"dry-run": ""}
	err := runCleanupWithURL(c, ts.URL, flags)
	if err == nil {
		t.Fatal("expected an error for server error response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to contain status code 500, got: %v", err)
	}
}

func TestCleanupCommand_ConnectionError(t *testing.T) {
	var buf bytes.Buffer
	c := New(WithOutput(&buf), WithSocketPath("/nonexistent/path.sock"))

	flags := map[string]string{"dry-run": ""}
	err := runCleanupWithURL(c, "http://127.0.0.1:1", flags)
	if err == nil {
		t.Fatal("expected an error for unreachable panel")
	}
}

func TestCleanupCommand_EmptyResults(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"dry_run": true,
			"results": []map[string]any{},
		})
	}))
	defer ts.Close()

	var buf bytes.Buffer
	c := New(WithOutput(&buf), WithSocketPath("/nonexistent/path.sock"))

	flags := map[string]string{"dry-run": ""}
	err := runCleanupWithURL(c, ts.URL, flags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No cleanup targets found.") {
		t.Errorf("expected 'No cleanup targets found.' message, got:\n%s", output)
	}
}

func TestCleanupCommand_ViaExecute(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"dry_run": true,
			"results": []map[string]any{
				{"target": "stale_sessions", "row_count": 100, "oldest": "2024-01-01T00:00:00Z"},
			},
		})
	}))
	defer ts.Close()

	var buf bytes.Buffer
	c := New(WithOutput(&buf), WithSocketPath("/nonexistent/path.sock"))

	// Register the cleanup command with a test URL override.
	cmd := Command{
		Name:        "cleanup",
		Description: "Preview or execute data cleanup operations",
		Run: func(args []string) error {
			flags := parseFlags(args)
			return runCleanupWithURL(c, ts.URL, flags)
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"cleanup", "--dry-run"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "stale_sessions") {
		t.Errorf("expected output to contain 'stale_sessions', got:\n%s", output)
	}
}

func TestCleanupCommand_ViaExecute_NoFlags(t *testing.T) {
	var buf bytes.Buffer
	c := New(WithOutput(&buf), WithSocketPath("/nonexistent/path.sock"))

	cmd := Command{
		Name:        "cleanup",
		Description: "Preview or execute data cleanup operations",
		Run: func(args []string) error {
			flags := parseFlags(args)
			return runCleanupWithURL(c, "http://localhost:1", flags)
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"cleanup"})
	if err == nil {
		t.Fatal("expected an error when no flags provided")
	}
}
