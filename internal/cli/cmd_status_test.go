package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStatusCommand_TableOutput(t *testing.T) {
	// Set up a test server returning a known status response.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/status" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":             true,
			"version":        "0.92.1",
			"uptime_seconds": 86400.0,
			"workers":        1,
			"db_pool": map[string]any{
				"max_open": 25,
				"open":     10,
				"in_use":   3,
				"idle":     7,
			},
			"nodes": map[string]any{
				"online":  5,
				"stale":   1,
				"offline": 2,
			},
		})
	}))
	defer ts.Close()

	var buf bytes.Buffer
	c := New(
		WithOutput(&buf),
		WithSocketPath("/nonexistent/path.sock"), // force HTTP fallback
	)

	// Override the makeRequest to hit our test server instead.
	cmd := Command{
		Name: "status",
		Run: func(args []string) error {
			return runStatusWithURL(c, ts.URL)
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"status"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Verify key content is present.
	checks := []string{
		"Panel Status",
		"0.92.1",
		"1d 0h 0m",
		"Max Open",
		"25",
		"In Use",
		"3",
		"Online",
		"5",
		"Stale",
		"1",
		"Offline",
		"2",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("output missing %q\nGot:\n%s", check, output)
		}
	}
}

func TestStatusCommand_JSONOutput(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/status" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":             true,
			"version":        "0.92.1",
			"uptime_seconds": 3661.0,
			"workers":        2,
			"db_pool": map[string]any{
				"max_open": 25,
				"open":     5,
				"in_use":   2,
				"idle":     3,
			},
			"nodes": map[string]any{
				"online":  3,
				"stale":   0,
				"offline": 1,
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
		Name: "status",
		Run: func(args []string) error {
			return runStatusWithURL(c, ts.URL)
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"status"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify output is valid JSON.
	output := strings.TrimSpace(buf.String())
	var parsed map[string]any
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nGot: %s", err, output)
	}

	if parsed["version"] != "0.92.1" {
		t.Errorf("expected version 0.92.1, got %v", parsed["version"])
	}
}

func TestStatusCommand_ConnectionError(t *testing.T) {
	var buf bytes.Buffer
	c := New(
		WithOutput(&buf),
		WithSocketPath("/nonexistent/path.sock"),
	)

	cmd := Command{
		Name: "status",
		Run: func(args []string) error {
			// Use a URL that won't be reachable.
			return runStatusWithURL(c, "http://127.0.0.1:1")
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"status"})
	if err == nil {
		t.Fatal("expected an error for unreachable panel")
	}
}

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		name     string
		seconds  float64
		expected string
	}{
		{"zero", 0, "0m"},
		{"minutes only", 300, "5m"},
		{"hours and minutes", 3661, "1h 1m"},
		{"days hours minutes", 90061, "1d 1h 1m"},
		{"multiple days", 259200, "3d 0h 0m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatUptime(tt.seconds)
			if got != tt.expected {
				t.Errorf("formatUptime(%v) = %q, want %q", tt.seconds, got, tt.expected)
			}
		})
	}
}
