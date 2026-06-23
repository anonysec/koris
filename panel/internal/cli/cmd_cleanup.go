package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// cleanupRequest is the JSON body sent to POST /internal/cleanup.
type cleanupRequest struct {
	DryRun    bool     `json:"dry_run"`
	Confirm   bool     `json:"confirm"`
	OlderThan string   `json:"older_than,omitempty"`
	Targets   []string `json:"targets,omitempty"`
}

// cleanupResponse represents the JSON payload returned by POST /internal/cleanup.
type cleanupResponse struct {
	OK      bool            `json:"ok"`
	DryRun  bool            `json:"dry_run"`
	Results []cleanupResult `json:"results"`
}

type cleanupResult struct {
	Target   string `json:"target"`
	RowCount int64  `json:"row_count"`
	Oldest   string `json:"oldest"`
}

// CleanupCommand returns a Command for `koris cleanup`.
// It connects to the panel's internal cleanup endpoint to preview
// or execute data cleanup operations.
//
// Flags:
//
//	--dry-run          Preview cleanup without deleting (default behavior)
//	--confirm          Actually execute the cleanup
//	--older-than=Xd    Retention period in days (e.g. "90d", "30d")
//	--targets=x,y,z    Comma-separated list of cleanup targets
func CleanupCommand(c *CLI) Command {
	return Command{
		Name:        "cleanup",
		Description: "Preview or execute data cleanup operations",
		Flags: []Flag{
			{Name: "dry-run", Description: "Preview cleanup without deleting"},
			{Name: "confirm", Description: "Execute the cleanup (requires --older-than)"},
			{Name: "older-than", Description: "Retention period (e.g. 90d, 30d)", HasValue: true},
			{Name: "targets", Description: "Comma-separated cleanup targets", HasValue: true},
		},
		Run: func(args []string) error {
			return runCleanup(c, args)
		},
	}
}

func runCleanup(c *CLI, args []string) error {
	flags := parseFlags(args)
	return runCleanupWithURL(c, "", flags)
}

// runCleanupWithURL performs cleanup against a specific base URL (used in tests).
// If baseURL is empty, it uses the CLI's makeRequest mechanism.
func runCleanupWithURL(c *CLI, baseURL string, flags map[string]string) error {
	_, hasDryRun := flags["dry-run"]
	_, hasConfirm := flags["confirm"]

	// Require at least one mode flag.
	if !hasDryRun && !hasConfirm {
		return fmt.Errorf("cleanup requires either --dry-run or --confirm flag\n\nUsage:\n  koris cleanup --dry-run\n  koris cleanup --older-than=90d --confirm")
	}

	// Validate --older-than format if provided.
	olderThan := flags["older-than"]
	if olderThan != "" {
		if err := validateOlderThan(olderThan); err != nil {
			return err
		}
	}

	// Build request body.
	reqBody := cleanupRequest{
		DryRun:    hasDryRun,
		Confirm:   hasConfirm,
		OlderThan: olderThan,
	}

	if targets, ok := flags["targets"]; ok && targets != "" {
		reqBody.Targets = strings.Split(targets, ",")
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	// Make the request.
	var resp *http.Response
	if baseURL != "" {
		client := &http.Client{}
		req, err := http.NewRequest(http.MethodPost, baseURL+"/internal/cleanup", bytes.NewReader(bodyBytes))
		if err != nil {
			return fmt.Errorf("cannot connect to panel: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err = client.Do(req)
		if err != nil {
			return fmt.Errorf("cannot connect to panel: %w", err)
		}
	} else {
		resp, err = c.makeRequestWithBody(http.MethodPost, "/internal/cleanup", bodyBytes)
		if err != nil {
			return fmt.Errorf("cannot connect to panel: %w", err)
		}
	}
	defer resp.Body.Close()

	return renderCleanup(c, resp, hasDryRun)
}

func renderCleanup(c *CLI, resp *http.Response, dryRun bool) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("panel returned status %d: %s", resp.StatusCode, string(body))
	}

	var data cleanupResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// JSON output mode.
	if c.JSONOutput() {
		_, err := fmt.Fprintln(c.Output(), string(body))
		return err
	}

	// Formatted table output.
	w := c.Output()

	if dryRun || data.DryRun {
		fmt.Fprintln(w, "Cleanup Preview (dry-run)")
		fmt.Fprintln(w, "")
	} else {
		fmt.Fprintln(w, "Cleanup Results")
		fmt.Fprintln(w, "")
	}

	rowLabel := "Row Count"
	if !dryRun && !data.DryRun {
		rowLabel = "Rows Deleted"
	}

	table := NewTable("Target", rowLabel, "Oldest Record")
	for _, r := range data.Results {
		table.AddRow(
			r.Target,
			strconv.FormatInt(r.RowCount, 10),
			r.Oldest,
		)
	}

	if table.RowCount() == 0 {
		fmt.Fprintln(w, "No cleanup targets found.")
		return nil
	}

	table.Render(w)
	return nil
}

// validateOlderThan checks that the value follows the Nd format (e.g. "90d", "30d").
func validateOlderThan(val string) error {
	if !strings.HasSuffix(val, "d") {
		return fmt.Errorf("invalid --older-than format %q: must be in Nd format (e.g. 90d, 30d)", val)
	}
	numStr := strings.TrimSuffix(val, "d")
	days, err := strconv.Atoi(numStr)
	if err != nil || days < 1 {
		return fmt.Errorf("invalid --older-than format %q: must be a positive number of days (e.g. 90d, 30d)", val)
	}
	return nil
}
