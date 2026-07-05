package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// updateCheckResponse represents the JSON payload returned by GET /internal/update/check.
type updateCheckResponse struct {
	OK     bool       `json:"ok"`
	Error  string     `json:"error,omitempty"`
	Update updateInfo `json:"update"`
}

type updateInfo struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	Changelog      string `json:"changelog"`
	Available      bool   `json:"available"`
}

// updateApplyResponse represents the JSON payload returned by POST /internal/update/apply.
type updateApplyResponse struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

// UpdateCommand returns a Command for `koris update`.
// It checks for available panel updates and optionally applies them.
// Use --yes to skip the confirmation prompt and apply immediately.
func UpdateCommand(c *CLI) Command {
	return Command{
		Name:        "update",
		Description: "Check for panel updates and apply if available",
		Flags: []Flag{
			{Name: "yes", Short: "y", Description: "Skip confirmation and apply update immediately"},
			{Name: "check", Description: "Only check for updates without applying"},
		},
		Run: func(args []string) error {
			return runUpdate(c, args)
		},
	}
}

// runUpdateWithURL is used in tests to hit a specific base URL instead of the socket.
func runUpdateWithURL(c *CLI, baseURL string, args []string) error {
	autoConfirm := false
	checkOnly := false
	for _, arg := range args {
		if arg == "--yes" || arg == "-y" {
			autoConfirm = true
		}
		if arg == "--check" {
			checkOnly = true
		}
	}

	client := &http.Client{}

	// Step 1: Check for updates
	req, err := http.NewRequest(http.MethodGet, baseURL+"/internal/update/check", nil)
	if err != nil {
		return fmt.Errorf("cannot connect to panel: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot connect to panel: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("panel returned status %d: %s", resp.StatusCode, string(body))
	}

	var check updateCheckResponse
	if err := json.Unmarshal(body, &check); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !check.OK {
		return fmt.Errorf("update check failed: %s", check.Error)
	}

	// JSON output mode
	if c.JSONOutput() {
		_, err := fmt.Fprintln(c.Output(), string(body))
		return err
	}

	w := c.Output()

	if !check.Update.Available {
		fmt.Fprintf(w, "✓ Panel is up to date (v%s)\n", check.Update.CurrentVersion)
		return nil
	}

	fmt.Fprintf(w, "Update available: v%s → v%s\n\n", check.Update.CurrentVersion, check.Update.LatestVersion)

	if check.Update.Changelog != "" {
		fmt.Fprintf(w, "Changelog:\n")
		for _, line := range strings.Split(check.Update.Changelog, "\n") {
			fmt.Fprintf(w, "  %s\n", line)
		}
		fmt.Fprintln(w)
	}

	if checkOnly {
		return nil
	}

	if !autoConfirm {
		fmt.Fprintf(w, "Run 'koris update --yes' to apply this update.\n")
		return nil
	}

	// Step 2: Apply the update
	fmt.Fprintf(w, "Applying update...\n")

	applyReq, err := http.NewRequest(http.MethodPost, baseURL+"/internal/update/apply", nil)
	if err != nil {
		return fmt.Errorf("cannot connect to panel: %w", err)
	}
	applyResp, err := client.Do(applyReq)
	if err != nil {
		return fmt.Errorf("cannot connect to panel: %w", err)
	}
	defer applyResp.Body.Close()

	applyBody, err := io.ReadAll(applyResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read apply response: %w", err)
	}

	if applyResp.StatusCode != 200 {
		return fmt.Errorf("update apply failed (status %d): %s", applyResp.StatusCode, string(applyBody))
	}

	var apply updateApplyResponse
	if err := json.Unmarshal(applyBody, &apply); err != nil {
		return fmt.Errorf("failed to parse apply response: %w", err)
	}

	if !apply.OK {
		return fmt.Errorf("update apply failed: %s", apply.Error)
	}

	fmt.Fprintf(w, "✓ %s\n", apply.Message)
	return nil
}

func runUpdate(c *CLI, args []string) error {
	autoConfirm := false
	checkOnly := false
	for _, arg := range args {
		if arg == "--yes" || arg == "-y" {
			autoConfirm = true
		}
		if arg == "--check" {
			checkOnly = true
		}
	}

	// Step 1: Check for updates
	resp, err := c.makeRequest(http.MethodGet, "/internal/update/check")
	if err != nil {
		return fmt.Errorf("cannot connect to panel: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("panel returned status %d: %s", resp.StatusCode, string(body))
	}

	var check updateCheckResponse
	if err := json.Unmarshal(body, &check); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !check.OK {
		return fmt.Errorf("update check failed: %s", check.Error)
	}

	// JSON output mode — just print the check result.
	if c.JSONOutput() {
		_, err := fmt.Fprintln(c.Output(), string(body))
		return err
	}

	w := c.Output()

	if !check.Update.Available {
		fmt.Fprintf(w, "✓ Panel is up to date (v%s)\n", check.Update.CurrentVersion)
		return nil
	}

	// Display update information
	fmt.Fprintf(w, "Update available: v%s → v%s\n\n", check.Update.CurrentVersion, check.Update.LatestVersion)

	if check.Update.Changelog != "" {
		fmt.Fprintf(w, "Changelog:\n")
		for _, line := range strings.Split(check.Update.Changelog, "\n") {
			fmt.Fprintf(w, "  %s\n", line)
		}
		fmt.Fprintln(w)
	}

	if checkOnly {
		return nil
	}

	if !autoConfirm {
		fmt.Fprintf(w, "Run 'koris update --yes' to apply this update.\n")
		return nil
	}

	// Step 2: Apply the update
	fmt.Fprintf(w, "Applying update...\n")

	applyResp, err := c.makeRequest(http.MethodPost, "/internal/update/apply")
	if err != nil {
		return fmt.Errorf("cannot connect to panel: %w", err)
	}
	defer applyResp.Body.Close()

	applyBody, err := io.ReadAll(applyResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read apply response: %w", err)
	}

	if applyResp.StatusCode != 200 {
		return fmt.Errorf("update apply failed (status %d): %s", applyResp.StatusCode, string(applyBody))
	}

	var apply updateApplyResponse
	if err := json.Unmarshal(applyBody, &apply); err != nil {
		return fmt.Errorf("failed to parse apply response: %w", err)
	}

	if !apply.OK {
		return fmt.Errorf("update apply failed: %s", apply.Error)
	}

	fmt.Fprintf(w, "✓ %s\n", apply.Message)
	return nil
}
