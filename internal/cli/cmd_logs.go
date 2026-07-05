package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// logsResponse represents the JSON payload returned by GET /internal/logs.
type logsResponse struct {
	OK      bool       `json:"ok"`
	Entries []logEntry `json:"entries"`
}

type logEntry struct {
	Time      string         `json:"time"`
	Level     string         `json:"level"`
	Component string         `json:"component"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields,omitempty"`
}

// LogsCommand returns a Command for `koris logs`.
// It connects to the running panel via Unix socket or HTTP fallback,
// fetches recent log entries from the ring buffer, and displays them
// formatted as: TIME [LEVEL] [component] message fields
func LogsCommand(c *CLI) Command {
	return Command{
		Name:        "logs",
		Description: "Display recent log entries from the panel process",
		Flags: []Flag{
			{Name: "tail", Short: "n", Description: "Number of recent entries to show (default 100)"},
		},
		Run: func(args []string) error {
			return runLogs(c, args)
		},
	}
}

func runLogs(c *CLI, args []string) error {
	tail := 100
	for _, arg := range args {
		if strings.HasPrefix(arg, "--tail=") {
			val := strings.TrimPrefix(arg, "--tail=")
			n, err := strconv.Atoi(val)
			if err != nil || n < 1 {
				return fmt.Errorf("invalid --tail value: %s", val)
			}
			tail = n
		}
	}

	path := fmt.Sprintf("/internal/logs?tail=%d", tail)
	resp, err := c.makeRequest(http.MethodGet, path)
	if err != nil {
		return fmt.Errorf("cannot connect to panel: %w", err)
	}
	defer resp.Body.Close()

	return renderLogs(c, resp)
}

// runLogsWithURL fetches logs from a specific base URL (used in tests).
func runLogsWithURL(c *CLI, baseURL string, args []string) error {
	tail := 100
	for _, arg := range args {
		if strings.HasPrefix(arg, "--tail=") {
			val := strings.TrimPrefix(arg, "--tail=")
			n, err := strconv.Atoi(val)
			if err != nil || n < 1 {
				return fmt.Errorf("invalid --tail value: %s", val)
			}
			tail = n
		}
	}

	client := &http.Client{}
	url := fmt.Sprintf("%s/internal/logs?tail=%d", baseURL, tail)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("cannot connect to panel: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot connect to panel: %w", err)
	}
	defer resp.Body.Close()

	return renderLogs(c, resp)
}

func renderLogs(c *CLI, resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("panel returned status %d: %s", resp.StatusCode, string(body))
	}

	var data logsResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// JSON output mode.
	if c.JSONOutput() {
		_, err := fmt.Fprintln(c.Output(), string(body))
		return err
	}

	// Formatted output.
	w := c.Output()
	if len(data.Entries) == 0 {
		fmt.Fprintln(w, "No log entries found.")
		return nil
	}

	for _, entry := range data.Entries {
		formatLogEntry(w, entry)
	}
	return nil
}

// formatLogEntry writes a single log entry in the format:
// TIME [LEVEL] [component] message key=value key=value...
func formatLogEntry(w io.Writer, entry logEntry) {
	var sb strings.Builder
	sb.WriteString(entry.Time)
	sb.WriteString(" [")
	sb.WriteString(entry.Level)
	sb.WriteString("] [")
	sb.WriteString(entry.Component)
	sb.WriteString("] ")
	sb.WriteString(entry.Message)

	if len(entry.Fields) > 0 {
		for k, v := range entry.Fields {
			sb.WriteString(fmt.Sprintf(" %s=%v", k, v))
		}
	}
	sb.WriteString("\n")
	fmt.Fprint(w, sb.String())
}
