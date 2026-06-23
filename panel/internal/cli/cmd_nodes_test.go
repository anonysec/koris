package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNodesCommand_TableOutput(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/nodes" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"nodes": []map[string]any{
				{
					"id":           1,
					"name":         "DE-01",
					"ip":           "1.2.3.4",
					"health_score": 0.95,
					"status":       "online",
					"last_seen":    "2024-01-15T10:30:00Z",
				},
				{
					"id":           2,
					"name":         "US-01",
					"ip":           "5.6.7.8",
					"health_score": 0.72,
					"status":       "stale",
					"last_seen":    "2024-01-15T10:25:00Z",
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
		Name: "nodes",
		Run: func(args []string) error {
			return runNodesWithURL(c, ts.URL)
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"nodes"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	checks := []string{
		"ID",
		"Name",
		"IP",
		"Health Score",
		"Status",
		"Last Seen",
		"DE-01",
		"1.2.3.4",
		"0.95",
		"online",
		"2024-01-15T10:30:00Z",
		"US-01",
		"5.6.7.8",
		"0.72",
		"stale",
		"2024-01-15T10:25:00Z",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("output missing %q\nGot:\n%s", check, output)
		}
	}
}

func TestNodesCommand_JSONOutput(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/nodes" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"nodes": []map[string]any{
				{
					"id":           1,
					"name":         "DE-01",
					"ip":           "1.2.3.4",
					"health_score": 0.95,
					"status":       "online",
					"last_seen":    "2024-01-15T10:30:00Z",
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
		Name: "nodes",
		Run: func(args []string) error {
			return runNodesWithURL(c, ts.URL)
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"nodes"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	var parsed map[string]any
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nGot: %s", err, output)
	}

	nodes, ok := parsed["nodes"].([]any)
	if !ok || len(nodes) != 1 {
		t.Fatalf("expected 1 node in JSON output, got %v", parsed["nodes"])
	}

	node := nodes[0].(map[string]any)
	if node["name"] != "DE-01" {
		t.Errorf("expected node name DE-01, got %v", node["name"])
	}
}

func TestNodesCommand_EmptyList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/nodes" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":    true,
			"nodes": []map[string]any{},
		})
	}))
	defer ts.Close()

	var buf bytes.Buffer
	c := New(
		WithOutput(&buf),
		WithSocketPath("/nonexistent/path.sock"),
	)

	cmd := Command{
		Name: "nodes",
		Run: func(args []string) error {
			return runNodesWithURL(c, ts.URL)
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"nodes"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No nodes found.") {
		t.Errorf("expected 'No nodes found.' message, got:\n%s", output)
	}
}

func TestNodesCommand_ConnectionError(t *testing.T) {
	var buf bytes.Buffer
	c := New(
		WithOutput(&buf),
		WithSocketPath("/nonexistent/path.sock"),
	)

	cmd := Command{
		Name: "nodes",
		Run: func(args []string) error {
			return runNodesWithURL(c, "http://127.0.0.1:1")
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"nodes"})
	if err == nil {
		t.Fatal("expected an error for unreachable panel")
	}
}

func TestNodesCommand_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"ok":false,"error":"db_error"}`))
	}))
	defer ts.Close()

	var buf bytes.Buffer
	c := New(
		WithOutput(&buf),
		WithSocketPath("/nonexistent/path.sock"),
	)

	cmd := Command{
		Name: "nodes",
		Run: func(args []string) error {
			return runNodesWithURL(c, ts.URL)
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"nodes"})
	if err == nil {
		t.Fatal("expected an error for server error response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to contain status code 500, got: %v", err)
	}
}
