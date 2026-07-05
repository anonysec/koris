package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUpdateCommand_NoUpdateAvailable(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/update/check" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"update": map[string]any{
				"current_version": "0.93.0",
				"latest_version":  "0.93.0",
				"changelog":       "",
				"available":       false,
			},
		})
	}))
	defer ts.Close()

	var buf bytes.Buffer
	c := New(WithOutput(&buf), WithSocketPath("/nonexistent/path.sock"))

	cmd := Command{
		Name: "update",
		Run: func(args []string) error {
			return runUpdateWithURL(c, ts.URL, args)
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"update"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "up to date") {
		t.Errorf("expected 'up to date' message, got: %s", output)
	}
	if !strings.Contains(output, "0.93.0") {
		t.Errorf("expected version in output, got: %s", output)
	}
}

func TestUpdateCommand_UpdateAvailable_NoConfirm(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/update/check" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"update": map[string]any{
				"current_version": "0.92.0",
				"latest_version":  "0.93.0",
				"changelog":       "- Bug fixes\n- New features",
				"available":       true,
			},
		})
	}))
	defer ts.Close()

	var buf bytes.Buffer
	c := New(WithOutput(&buf), WithSocketPath("/nonexistent/path.sock"))

	cmd := Command{
		Name: "update",
		Run: func(args []string) error {
			return runUpdateWithURL(c, ts.URL, args)
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"update"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "0.92.0") {
		t.Errorf("expected current version in output, got: %s", output)
	}
	if !strings.Contains(output, "0.93.0") {
		t.Errorf("expected latest version in output, got: %s", output)
	}
	if !strings.Contains(output, "Bug fixes") {
		t.Errorf("expected changelog in output, got: %s", output)
	}
	if !strings.Contains(output, "koris update --yes") {
		t.Errorf("expected confirmation hint, got: %s", output)
	}
}

func TestUpdateCommand_UpdateAvailable_WithYes(t *testing.T) {
	checkCalled := false
	applyCalled := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/internal/update/check":
			checkCalled = true
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"update": map[string]any{
					"current_version": "0.92.0",
					"latest_version":  "0.93.0",
					"changelog":       "- Improvements",
					"available":       true,
				},
			})
		case "/internal/update/apply":
			applyCalled = true
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"ok":      true,
				"message": "update applied, restarting...",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	var buf bytes.Buffer
	c := New(WithOutput(&buf), WithSocketPath("/nonexistent/path.sock"))

	cmd := Command{
		Name: "update",
		Run: func(args []string) error {
			return runUpdateWithURL(c, ts.URL, args)
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"update", "--yes"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !checkCalled {
		t.Error("expected check endpoint to be called")
	}
	if !applyCalled {
		t.Error("expected apply endpoint to be called")
	}

	output := buf.String()
	if !strings.Contains(output, "Applying update") {
		t.Errorf("expected 'Applying update' message, got: %s", output)
	}
	if !strings.Contains(output, "update applied") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestUpdateCommand_CheckOnly(t *testing.T) {
	applyCalled := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/internal/update/check":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"update": map[string]any{
					"current_version": "0.92.0",
					"latest_version":  "0.93.0",
					"changelog":       "- Fix",
					"available":       true,
				},
			})
		case "/internal/update/apply":
			applyCalled = true
			http.Error(w, "should not be called", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	var buf bytes.Buffer
	c := New(WithOutput(&buf), WithSocketPath("/nonexistent/path.sock"))

	cmd := Command{
		Name: "update",
		Run: func(args []string) error {
			return runUpdateWithURL(c, ts.URL, args)
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"update", "--check"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if applyCalled {
		t.Error("apply endpoint should not be called with --check flag")
	}

	output := buf.String()
	if !strings.Contains(output, "0.93.0") {
		t.Errorf("expected version info, got: %s", output)
	}
}

func TestUpdateCommand_JSONOutput(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/update/check" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"update": map[string]any{
				"current_version": "0.92.0",
				"latest_version":  "0.93.0",
				"changelog":       "",
				"available":       true,
			},
		})
	}))
	defer ts.Close()

	var buf bytes.Buffer
	c := New(WithOutput(&buf), WithJSONOutput(true), WithSocketPath("/nonexistent/path.sock"))

	cmd := Command{
		Name: "update",
		Run: func(args []string) error {
			return runUpdateWithURL(c, ts.URL, args)
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"update"})
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
}

func TestUpdateCommand_ApplyFails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/internal/update/check":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"update": map[string]any{
					"current_version": "0.92.0",
					"latest_version":  "0.93.0",
					"changelog":       "",
					"available":       true,
				},
			})
		case "/internal/update/apply":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"ok":    false,
				"error": "apply_failed",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	var buf bytes.Buffer
	c := New(WithOutput(&buf), WithSocketPath("/nonexistent/path.sock"))

	cmd := Command{
		Name: "update",
		Run: func(args []string) error {
			return runUpdateWithURL(c, ts.URL, args)
		},
	}
	c.RegisterCommand(cmd)

	err := c.Execute([]string{"update", "--yes"})
	if err == nil {
		t.Fatal("expected an error when apply fails")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected status 500 in error, got: %v", err)
	}
}
