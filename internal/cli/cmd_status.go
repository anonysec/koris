package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// statusResponse represents the JSON payload returned by GET /internal/status.
type statusResponse struct {
	OK            bool    `json:"ok"`
	Version       string  `json:"version"`
	UptimeSeconds float64 `json:"uptime_seconds"`
	Workers       int     `json:"workers"`
	DBPool        struct {
		MaxOpen int `json:"max_open"`
		Open    int `json:"open"`
		InUse   int `json:"in_use"`
		Idle    int `json:"idle"`
	} `json:"db_pool"`
	Nodes struct {
		Online  int `json:"online"`
		Stale   int `json:"stale"`
		Offline int `json:"offline"`
	} `json:"nodes"`
}

// StatusCommand returns a Command for `koris status`.
// It connects to the running panel via Unix socket or HTTP fallback,
// fetches internal status, and displays version, uptime, workers,
// DB pool stats, and node counts.
func StatusCommand(c *CLI) Command {
	return Command{
		Name:        "status",
		Description: "Show panel status (version, uptime, workers, DB pool, nodes)",
		Run: func(args []string) error {
			return runStatus(c)
		},
	}
}

func runStatus(c *CLI) error {
	resp, err := c.makeRequest(http.MethodGet, "/internal/status")
	if err != nil {
		return fmt.Errorf("cannot connect to panel: %w", err)
	}
	defer resp.Body.Close()

	return renderStatus(c, resp)
}

// runStatusWithURL fetches status from a specific base URL (used in tests).
func runStatusWithURL(c *CLI, baseURL string) error {
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, baseURL+"/internal/status", nil)
	if err != nil {
		return fmt.Errorf("cannot connect to panel: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot connect to panel: %w", err)
	}
	defer resp.Body.Close()

	return renderStatus(c, resp)
}

func renderStatus(c *CLI, resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("panel returned status %d: %s", resp.StatusCode, string(body))
	}

	var status statusResponse
	if err := json.Unmarshal(body, &status); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// JSON output mode.
	if c.JSONOutput() {
		_, err := fmt.Fprintln(c.Output(), string(body))
		return err
	}

	// Formatted table output.
	w := c.Output()
	fmt.Fprintf(w, "Panel Status\n")
	fmt.Fprintf(w, "============\n\n")

	// General info table.
	info := NewTable("Field", "Value")
	info.AddRow("Version", status.Version)
	info.AddRow("Uptime", formatUptime(status.UptimeSeconds))
	info.AddRow("Workers", fmt.Sprintf("%d", status.Workers))
	info.Render(w)

	fmt.Fprintln(w)

	// DB pool table.
	fmt.Fprintf(w, "Database Pool\n")
	pool := NewTable("Metric", "Value")
	pool.AddRow("Max Open", fmt.Sprintf("%d", status.DBPool.MaxOpen))
	pool.AddRow("Open", fmt.Sprintf("%d", status.DBPool.Open))
	pool.AddRow("In Use", fmt.Sprintf("%d", status.DBPool.InUse))
	pool.AddRow("Idle", fmt.Sprintf("%d", status.DBPool.Idle))
	pool.Render(w)

	fmt.Fprintln(w)

	// Nodes table.
	fmt.Fprintf(w, "Nodes\n")
	nodes := NewTable("Status", "Count")
	nodes.AddRow("Online", fmt.Sprintf("%d", status.Nodes.Online))
	nodes.AddRow("Stale", fmt.Sprintf("%d", status.Nodes.Stale))
	nodes.AddRow("Offline", fmt.Sprintf("%d", status.Nodes.Offline))
	nodes.Render(w)

	return nil
}

// formatUptime converts seconds to a human-readable duration string.
func formatUptime(seconds float64) string {
	d := time.Duration(seconds) * time.Second
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
