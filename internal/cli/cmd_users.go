package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// usersResponse represents the JSON payload returned by GET /internal/users.
type usersResponse struct {
	OK    bool       `json:"ok"`
	Users []userItem `json:"users"`
	Total int        `json:"total"`
	Page  int        `json:"page"`
	Limit int        `json:"limit"`
}

type userItem struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Status    string `json:"status"`
	Plan      string `json:"plan"`
	CreatedAt string `json:"created_at"`
}

// UsersCommand returns a Command for `koris users`.
// Without a subcommand it shows help. The `list` subcommand
// fetches users with optional --status filter and pagination.
func UsersCommand(c *CLI) Command {
	return Command{
		Name:        "users",
		Description: "Manage panel users",
		Flags:       []Flag{},
		SubCommands: []Command{
			{
				Name:        "list",
				Description: "List users with optional status filter",
				Flags: []Flag{
					{Name: "status", Description: "Filter by status (e.g. active, expired, disabled)", HasValue: true},
					{Name: "page", Description: "Page number (default 1)", HasValue: true, Default: "1"},
					{Name: "limit", Description: "Results per page (default 50)", HasValue: true, Default: "50"},
				},
				Run: func(args []string) error {
					return runUsersList(c, args)
				},
			},
		},
	}
}

func runUsersList(c *CLI, args []string) error {
	// Parse flags from args.
	flags := parseFlags(args)
	path := buildUsersPath(flags)

	resp, err := c.makeRequest(http.MethodGet, path)
	if err != nil {
		return fmt.Errorf("cannot connect to panel: %w", err)
	}
	defer resp.Body.Close()

	return renderUsers(c, resp)
}

// runUsersListWithURL fetches users from a specific base URL (used in tests).
func runUsersListWithURL(c *CLI, baseURL string, args []string) error {
	flags := parseFlags(args)
	path := buildUsersPath(flags)

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("cannot connect to panel: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot connect to panel: %w", err)
	}
	defer resp.Body.Close()

	return renderUsers(c, resp)
}

// buildUsersPath constructs the query string for the internal users endpoint.
func buildUsersPath(flags map[string]string) string {
	var params []string

	if status, ok := flags["status"]; ok && status != "" {
		params = append(params, "status="+status)
	}

	page := "1"
	if p, ok := flags["page"]; ok && p != "" {
		page = p
	}
	params = append(params, "page="+page)

	limit := "50"
	if l, ok := flags["limit"]; ok && l != "" {
		limit = l
	}
	params = append(params, "limit="+limit)

	return "/internal/users?" + strings.Join(params, "&")
}

// parseFlags extracts --key=value pairs from an args slice.
func parseFlags(args []string) map[string]string {
	flags := make(map[string]string)
	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			kv := strings.TrimPrefix(arg, "--")
			if idx := strings.Index(kv, "="); idx >= 0 {
				flags[kv[:idx]] = kv[idx+1:]
			} else {
				flags[kv] = ""
			}
		}
	}
	return flags
}

func renderUsers(c *CLI, resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("panel returned status %d: %s", resp.StatusCode, string(body))
	}

	var data usersResponse
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
	table := NewTable("ID", "Username", "Status", "Plan", "Created")
	for _, u := range data.Users {
		table.AddRow(
			fmt.Sprintf("%d", u.ID),
			u.Username,
			u.Status,
			u.Plan,
			u.CreatedAt,
		)
	}

	if table.RowCount() == 0 {
		fmt.Fprintln(w, "No users found.")
		return nil
	}

	table.Render(w)
	fmt.Fprintf(w, "\nShowing page %d (%d total users)\n", data.Page, data.Total)
	return nil
}
