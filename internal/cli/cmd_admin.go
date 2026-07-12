package cli

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/anonysec/koris/internal/auth"
	"github.com/anonysec/koris/internal/db"
	"golang.org/x/term"
)

// AdminCommand returns a Command for `koris admin ...`.
//
// Unlike the other CLI commands (which talk to the running panel over the
// local socket/HTTP), these commands open the database directly using
// PANEL_PG_DSN / PANEL_DB_DSN. That means they work even when the panel is
// stopped or you are locked out — no session cookie or live server required.
// This is the supported way to recover a lost admin password from the terminal.
func AdminCommand(c *CLI) Command {
	return Command{
		Name:        "admin",
		Description: "Manage admin accounts (list, create, change password)",
		Run: func(args []string) error {
			// Top-level `koris admin` with no subcommand (or --help) shows usage.
			for _, a := range args {
				if a == "--help" || a == "-h" || a == "help" {
					return printAdminUsage(c)
				}
			}
			return printAdminUsage(c)
		},
		SubCommands: []Command{
			{
				Name:        "list",
				Description: "List all admin accounts",
				Run: func(args []string) error { return adminList(c) },
			},
			{
				Name:        "passwd",
				Description: "Change an admin password: admin passwd <username> [--password=...]",
				Run: func(args []string) error { return adminPasswd(c, args) },
			},
			{
				Name:        "create",
				Description: "Create an admin: admin create <username> [--password=...] [--role=owner|admin|reseller]",
				Run: func(args []string) error { return adminCreate(c, args) },
			},
		},
	}
}

func printAdminUsage(c *CLI) error {
	w := c.Output()
	fmt.Fprintf(w, "Usage: koris admin <subcommand> [flags]\n\n")
	fmt.Fprintf(w, "Manage admin accounts directly in the database. These commands work even\n")
	fmt.Fprintf(w, "when the panel is stopped or you are locked out (no session required).\n\n")
	fmt.Fprintf(w, "Subcommands:\n")
	fmt.Fprintf(w, "  %-10s %s\n", "list", "List all admin accounts (id, username, role, active)")
	fmt.Fprintf(w, "  %-10s %s\n", "passwd", "Change an admin password  (admin passwd <username> [--password=...])")
	fmt.Fprintf(w, "  %-10s %s\n", "create", "Create an admin          (admin create <username> [--password=...] [--role=...])")
	fmt.Fprintf(w, "\nFlags:\n")
	fmt.Fprintf(w, "  --password  New password (prompted securely if omitted)\n")
	fmt.Fprintf(w, "  --role      owner | admin | reseller  (create only; default: admin)\n")
	return nil
}

// openAdminDB connects to the panel database using the same DSN the panel uses.
func openAdminDB() (*sql.DB, error) {
	dsn := os.Getenv("PANEL_PG_DSN")
	if dsn == "" {
		dsn = os.Getenv("PANEL_DB_DSN")
	}
	if dsn == "" {
		return nil, fmt.Errorf("PANEL_PG_DSN / PANEL_DB_DSN not set; run this via 'koris admin' or inside the panel container")
	}
	return db.Open(dsn)
}

// parseAdminArgs splits a CLI args slice into --flag=value flags and positionals.
func parseAdminArgs(args []string) (flags map[string]string, positional []string) {
	flags = make(map[string]string)
	for _, a := range args {
		if strings.HasPrefix(a, "--") {
			name := a[2:]
			if i := strings.IndexByte(name, '='); i >= 0 {
				flags[name[:i]] = name[i+1:]
			} else {
				flags[name] = ""
			}
		} else {
			positional = append(positional, a)
		}
	}
	return
}

// readPassword prompts for a password, hiding input when run on a TTY.
func readPassword(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		return string(b), err
	}
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	return strings.TrimSpace(line), err
}

func adminList(c *CLI) error {
	d, err := openAdminDB()
	if err != nil {
		return err
	}
	defer d.Close()

	rows, err := d.Query(`SELECT id, username, role, is_active FROM admins ORDER BY id`)
	if err != nil {
		return fmt.Errorf("query admins: %w", err)
	}
	defer rows.Close()

	t := NewTable("ID", "Username", "Role", "Active")
	n := 0
	for rows.Next() {
		var id int64
		var username, role string
		var active bool
		if err := rows.Scan(&id, &username, &role, &active); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		t.AddRow(fmt.Sprintf("%d", id), username, role, boolActive(active))
		n++
	}
	if n == 0 {
		fmt.Fprintln(c.Output(), "No admin accounts found.")
		return nil
	}
	t.Render(c.Output())
	return nil
}

func boolActive(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func adminPasswd(c *CLI, args []string) error {
	flags, pos := parseAdminArgs(args)
	if len(pos) == 0 {
		return fmt.Errorf("usage: koris admin passwd <username> [--password=...]")
	}
	username := strings.TrimSpace(pos[0])

	password := flags["password"]
	if password == "" {
		p, err := readPassword("New password: ")
		if err != nil {
			return err
		}
		password = p
	}
	password = strings.TrimSpace(password)
	if len(password) < 6 {
		return fmt.Errorf("password must be at least 6 characters")
	}

	d, err := openAdminDB()
	if err != nil {
		return err
	}
	defer d.Close()

	var exists bool
	if err := d.QueryRow(`SELECT EXISTS(SELECT 1 FROM admins WHERE username=$1)`, username).Scan(&exists); err != nil {
		return fmt.Errorf("lookup admin: %w", err)
	}
	if !exists {
		return fmt.Errorf("admin %q does not exist", username)
	}

	h, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if _, err := d.Exec(`UPDATE admins SET password_hash=$1, updated_at=NOW() WHERE username=$2`, h, username); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	fmt.Fprintf(c.Output(), "Password updated for admin %q.\n", username)
	return nil
}

func adminCreate(c *CLI, args []string) error {
	flags, pos := parseAdminArgs(args)
	if len(pos) == 0 {
		return fmt.Errorf("usage: koris admin create <username> [--password=...] [--role=owner|admin|reseller]")
	}
	username := strings.TrimSpace(pos[0])
	if username == "" {
		return fmt.Errorf("username required")
	}

	role := strings.TrimSpace(flags["role"])
	if role == "" {
		role = "admin"
	}
	switch role {
	case "owner", "admin", "reseller":
	default:
		return fmt.Errorf("invalid --role %q (use owner, admin, or reseller)", role)
	}

	password := flags["password"]
	if password == "" {
		p, err := readPassword("New password: ")
		if err != nil {
			return err
		}
		password = p
	}
	password = strings.TrimSpace(password)
	if len(password) < 6 {
		return fmt.Errorf("password must be at least 6 characters")
	}

	d, err := openAdminDB()
	if err != nil {
		return err
	}
	defer d.Close()

	var exists bool
	if err := d.QueryRow(`SELECT EXISTS(SELECT 1 FROM admins WHERE username=$1)`, username).Scan(&exists); err != nil {
		return fmt.Errorf("lookup admin: %w", err)
	}
	if exists {
		return fmt.Errorf("admin %q already exists", username)
	}

	h, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if _, err := d.Exec(
		`INSERT INTO admins(username, password_hash, role, is_active) VALUES($1,$2,$3,TRUE)`,
		username, h, role,
	); err != nil {
		return fmt.Errorf("create admin: %w", err)
	}
	fmt.Fprintf(c.Output(), "Admin %q created (role=%s).\n", username, role)
	return nil
}
