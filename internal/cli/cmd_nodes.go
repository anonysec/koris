package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// nodesResponse represents the JSON payload returned by GET /internal/nodes.
type nodesResponse struct {
	OK    bool       `json:"ok"`
	Nodes []nodeItem `json:"nodes"`
}

type nodeItem struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	IP          string  `json:"ip"`
	HealthScore float64 `json:"health_score"`
	Status      string  `json:"status"`
	LastSeen    string  `json:"last_seen"`
}

// NodesCommand returns a Command for `koris nodes`.
// It connects to the running panel via Unix socket or HTTP fallback,
// fetches the node list, and displays a table with ID, Name, IP,
// Health Score, Status, and Last Seen columns.
func NodesCommand(c *CLI) Command {
	return Command{
		Name:        "nodes",
		Description: "List all nodes with health score and status",
		Run: func(args []string) error {
			return runNodes(c)
		},
	}
}

func runNodes(c *CLI) error {
	resp, err := c.makeRequest(http.MethodGet, "/internal/nodes")
	if err != nil {
		return fmt.Errorf("cannot connect to panel: %w", err)
	}
	defer resp.Body.Close()

	return renderNodes(c, resp)
}

// runNodesWithURL fetches nodes from a specific base URL (used in tests).
func runNodesWithURL(c *CLI, baseURL string) error {
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, baseURL+"/internal/nodes", nil)
	if err != nil {
		return fmt.Errorf("cannot connect to panel: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot connect to panel: %w", err)
	}
	defer resp.Body.Close()

	return renderNodes(c, resp)
}

func renderNodes(c *CLI, resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("panel returned status %d: %s", resp.StatusCode, string(body))
	}

	var data nodesResponse
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
	table := NewTable("ID", "Name", "IP", "Health Score", "Status", "Last Seen")
	for _, n := range data.Nodes {
		table.AddRow(
			fmt.Sprintf("%d", n.ID),
			n.Name,
			n.IP,
			fmt.Sprintf("%.2f", n.HealthScore),
			n.Status,
			n.LastSeen,
		)
	}

	if table.RowCount() == 0 {
		fmt.Fprintln(w, "No nodes found.")
		return nil
	}

	table.Render(w)
	return nil
}
