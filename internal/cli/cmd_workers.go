package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// workersResponse represents the JSON payload returned by GET /internal/workers.
type workersResponse struct {
	OK      bool         `json:"ok"`
	Workers []workerItem `json:"workers"`
}

type workerItem struct {
	ID            int    `json:"id"`
	PID           int    `json:"pid"`
	Status        string `json:"status"`
	UptimeSeconds int64  `json:"uptime_seconds"`
	Restarts      int    `json:"restarts"`
}

// WorkersCommand returns a Command for `koris workers`.
// It connects to the running panel via Unix socket or HTTP fallback,
// fetches the worker list, and displays a table with PID, Status,
// Uptime, and Restart Count columns.
func WorkersCommand(c *CLI) Command {
	return Command{
		Name:        "workers",
		Description: "Display worker PIDs, status, uptime, and restart count",
		Run: func(args []string) error {
			return runWorkers(c)
		},
	}
}

func runWorkers(c *CLI) error {
	resp, err := c.makeRequest(http.MethodGet, "/internal/workers")
	if err != nil {
		return fmt.Errorf("cannot connect to panel: %w", err)
	}
	defer resp.Body.Close()

	return renderWorkers(c, resp)
}

// runWorkersWithURL fetches workers from a specific base URL (used in tests).
func runWorkersWithURL(c *CLI, baseURL string) error {
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, baseURL+"/internal/workers", nil)
	if err != nil {
		return fmt.Errorf("cannot connect to panel: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot connect to panel: %w", err)
	}
	defer resp.Body.Close()

	return renderWorkers(c, resp)
}

func renderWorkers(c *CLI, resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("panel returned status %d: %s", resp.StatusCode, string(body))
	}

	var data workersResponse
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
	table := NewTable("PID", "Status", "Uptime", "Restart Count")
	for _, worker := range data.Workers {
		table.AddRow(
			fmt.Sprintf("%d", worker.PID),
			worker.Status,
			formatWorkerUptime(worker.UptimeSeconds),
			fmt.Sprintf("%d", worker.Restarts),
		)
	}

	if table.RowCount() == 0 {
		fmt.Fprintln(w, "No workers found.")
		return nil
	}

	table.Render(w)
	return nil
}

// formatWorkerUptime converts seconds to a human-readable duration string.
func formatWorkerUptime(seconds int64) string {
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
