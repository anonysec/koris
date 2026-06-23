package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUsersListCommand_TableOutput(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/users" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"users": []map[string]any{
				{
					"id":         1,
					"username":   "alice",
					"status":     "active",
					"plan":       "Basic",
					"created_at": "2024-01-15T10:30:00Z",
				},
				{
					"id":         2,
					"username":   "bob",
					"status":     "expired",
					"plan":       "Pro",
					"created_at": "2024-02-20T14:00:00Z",
				},
			},
			"total": 2,
			"page":  1,
			"limit": 50,
		})
	}))
	defer ts.Close()

	var buf bytes.Buffer
	c := New(
		WithOutput(&buf),
		WithSocketPath("/nonexistent/path.sock"),
	)

	cmd := Command{
		Name: "users",
		SubCommands: []Command{
			{
				Name: "list",
				Run: func(args []string) error {
					return runUsersListWithURL(c, ts.URL, args)
				},
			},
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"users", "list"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	checks := []string{
		"ID",
		"Username",
		"Status",
		"Plan",
		"Created",
		"alice",
		"active",
		"Basic",
		"2024-01-15T10:30:00Z",
		"bob",
		"expired",
		"Pro",
		"2024-02-20T14:00:00Z",
		"2 total users",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("output missing %q\nGot:\n%s", check, output)
		}
	}
}

func TestUsersListCommand_StatusFilter(t *testing.T) {
	var receivedQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":    true,
			"users": []map[string]any{},
			"total": 0,
			"page":  1,
			"limit": 50,
		})
	}))
	defer ts.Close()

	var buf bytes.Buffer
	c := New(
		WithOutput(&buf),
		WithSocketPath("/nonexistent/path.sock"),
	)

	cmd := Command{
		Name: "users",
		SubCommands: []Command{
			{
				Name: "list",
				Run: func(args []string) error {
					return runUsersListWithURL(c, ts.URL, args)
				},
			},
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"users", "list", "--status=active"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(receivedQuery, "status=active") {
		t.Errorf("expected query to contain status=active, got: %s", receivedQuery)
	}
}

func TestUsersListCommand_Pagination(t *testing.T) {
	var receivedQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":    true,
			"users": []map[string]any{},
			"total": 0,
			"page":  2,
			"limit": 25,
		})
	}))
	defer ts.Close()

	var buf bytes.Buffer
	c := New(
		WithOutput(&buf),
		WithSocketPath("/nonexistent/path.sock"),
	)

	cmd := Command{
		Name: "users",
		SubCommands: []Command{
			{
				Name: "list",
				Run: func(args []string) error {
					return runUsersListWithURL(c, ts.URL, args)
				},
			},
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"users", "list", "--page=2", "--limit=25"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(receivedQuery, "page=2") {
		t.Errorf("expected query to contain page=2, got: %s", receivedQuery)
	}
	if !strings.Contains(receivedQuery, "limit=25") {
		t.Errorf("expected query to contain limit=25, got: %s", receivedQuery)
	}
}

func TestUsersListCommand_JSONOutput(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"users": []map[string]any{
				{
					"id":         1,
					"username":   "alice",
					"status":     "active",
					"plan":       "Basic",
					"created_at": "2024-01-15T10:30:00Z",
				},
			},
			"total": 1,
			"page":  1,
			"limit": 50,
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
		Name: "users",
		SubCommands: []Command{
			{
				Name: "list",
				Run: func(args []string) error {
					return runUsersListWithURL(c, ts.URL, args)
				},
			},
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"users", "list"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	var parsed map[string]any
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nGot: %s", err, output)
	}

	users, ok := parsed["users"].([]any)
	if !ok || len(users) != 1 {
		t.Fatalf("expected 1 user in JSON output, got %v", parsed["users"])
	}

	user := users[0].(map[string]any)
	if user["username"] != "alice" {
		t.Errorf("expected username alice, got %v", user["username"])
	}

	if parsed["total"].(float64) != 1 {
		t.Errorf("expected total 1, got %v", parsed["total"])
	}
}

func TestUsersListCommand_EmptyList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":    true,
			"users": []map[string]any{},
			"total": 0,
			"page":  1,
			"limit": 50,
		})
	}))
	defer ts.Close()

	var buf bytes.Buffer
	c := New(
		WithOutput(&buf),
		WithSocketPath("/nonexistent/path.sock"),
	)

	cmd := Command{
		Name: "users",
		SubCommands: []Command{
			{
				Name: "list",
				Run: func(args []string) error {
					return runUsersListWithURL(c, ts.URL, args)
				},
			},
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"users", "list"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No users found.") {
		t.Errorf("expected 'No users found.' message, got:\n%s", output)
	}
}

func TestUsersListCommand_ConnectionError(t *testing.T) {
	var buf bytes.Buffer
	c := New(
		WithOutput(&buf),
		WithSocketPath("/nonexistent/path.sock"),
	)

	cmd := Command{
		Name: "users",
		SubCommands: []Command{
			{
				Name: "list",
				Run: func(args []string) error {
					return runUsersListWithURL(c, "http://127.0.0.1:1", args)
				},
			},
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"users", "list"})
	if err == nil {
		t.Fatal("expected an error for unreachable panel")
	}
}

func TestUsersListCommand_ServerError(t *testing.T) {
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
		Name: "users",
		SubCommands: []Command{
			{
				Name: "list",
				Run: func(args []string) error {
					return runUsersListWithURL(c, ts.URL, args)
				},
			},
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"users", "list"})
	if err == nil {
		t.Fatal("expected an error for server error response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to contain status code 500, got: %v", err)
	}
}

func TestUsersCommand_NoSubcommand_ShowsHelp(t *testing.T) {
	var buf bytes.Buffer
	c := New(
		WithOutput(&buf),
		WithSocketPath("/nonexistent/path.sock"),
	)

	c.RegisterCommand(UsersCommand(c))

	err := c.Execute([]string{"users"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "list") {
		t.Errorf("expected help output to contain 'list' subcommand, got:\n%s", output)
	}
}
