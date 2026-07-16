package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CertCommand returns a Command for `koris cert ...`.
//
// It manages the panel TLS certificate and is the terminal/CLI path the user
// uses after a default (plaintext HTTP, loopback-only) install to expose HTTPS.
// Three subcommands are supported:
//
//	koris cert selfsign            -> PANEL_TLS_MODE=selfsigned (auto-generated cert)
//	koris cert letsencrypt --domain=example.com [--email=a@b.c]
//	                             -> PANEL_TLS_MODE=acme (Let's Encrypt autocert)
//	koris cert path --cert=/p/cert.pem --key=/p/key.pem
//	                             -> PANEL_TLS_MODE=manual (operator-provided files)
//
// All subcommands write the resolved PANEL_TLS_* variables into the panel env
// file (default $KORIS_HOME/.env, or /opt/koris/.env) and then prompt for a
// restart. If the panel is running inside Docker the command attempts
// `docker restart koris`; otherwise it prints the restart command to run.
func CertCommand(c *CLI) Command {
	return Command{
		Name:        "cert",
		Description: "Manage the panel TLS certificate (selfsign | letsencrypt | path)",
		Run: func(args []string) error {
			for _, a := range args {
				if a == "--help" || a == "-h" || a == "help" {
					return printCertUsage(c)
				}
			}
			return printCertUsage(c)
		},
		SubCommands: []Command{
			{
				Name:        "selfsign",
				Description: "Use an auto-generated self-signed certificate (DEV; browsers warn)",
				Run: func(args []string) error {
					return setCertMode(c, map[string]string{
						"PANEL_TLS_ENABLED": "true",
						"PANEL_TLS_MODE":    "selfsigned",
					})
				},
			},
			{
				Name:        "letsencrypt",
				Description: "Obtain a Let's Encrypt certificate (requires --domain)",
				Flags: []Flag{
					{Name: "domain", HasValue: true, Description: "Domain name for the certificate (required)"},
					{Name: "email", HasValue: true, Description: "Contact email for ACME (optional)"},
				},
				Run: func(args []string) error {
					domain, email := "", ""
					for _, a := range args {
						if v, ok := strings.CutPrefix(a, "--domain="); ok {
							domain = v
						}
						if v, ok := strings.CutPrefix(a, "--email="); ok {
							email = v
						}
					}
					if domain == "" || domain == "localhost" {
						return fmt.Errorf("letsencrypt requires --domain=<a real domain> (not localhost)")
					}
					vars := map[string]string{
						"PANEL_TLS_ENABLED": "true",
						"PANEL_TLS_MODE":    "acme",
						"PANEL_TLS_DOMAIN":  domain,
					}
					if email != "" {
						vars["PANEL_TLS_EMAIL"] = email
					}
					return setCertMode(c, vars)
				},
			},
			{
				Name:        "path",
				Description: "Use operator-provided cert/key files (manual mode)",
				Flags: []Flag{
					{Name: "cert", HasValue: true, Description: "Path to the certificate PEM file (required)"},
					{Name: "key", HasValue: true, Description: "Path to the private key PEM file (required)"},
				},
				Run: func(args []string) error {
					certPath, keyPath := "", ""
					for _, a := range args {
						if v, ok := strings.CutPrefix(a, "--cert="); ok {
							certPath = v
						}
						if v, ok := strings.CutPrefix(a, "--key="); ok {
							keyPath = v
						}
					}
					if certPath == "" || keyPath == "" {
						return fmt.Errorf("path requires --cert=<cert.pem> and --key=<key.pem>")
					}
					if _, err := os.Stat(certPath); err != nil {
						return fmt.Errorf("cert file not found: %s", certPath)
					}
					if _, err := os.Stat(keyPath); err != nil {
						return fmt.Errorf("key file not found: %s", keyPath)
					}
					return setCertMode(c, map[string]string{
						"PANEL_TLS_ENABLED": "true",
						"PANEL_TLS_MODE":    "manual",
						"PANEL_TLS_CERT":    certPath,
						"PANEL_TLS_KEY":     keyPath,
					})
				},
			},
		},
	}
}

// setCertMode writes the given PANEL_TLS_* variables into the panel env file
// and then triggers (or instructs) a restart so the new mode takes effect.
func setCertMode(c *CLI, vars map[string]string) error {
	w := c.Output()
	envPath := certEnvPath()

	// Ensure the dir exists.
	if dir := filepath.Dir(envPath); dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}

	if err := updateEnvFile(envPath, vars); err != nil {
		return fmt.Errorf("failed to write %s: %w", envPath, err)
	}
	fmt.Fprintf(w, "Updated %s:\n", envPath)
	for k, v := range vars {
		fmt.Fprintf(w, "  %s=%s\n", k, v)
	}

	// Restart the panel so the new cert mode takes effect.
	if restartPanel() {
		fmt.Fprintf(w, "\nPanel restarted. Verify with: curl -sk https://127.0.0.1:2096/api/health\n")
	} else {
		fmt.Fprintf(w, "\nRestart the panel to apply (Docker: `docker restart koris`; bare-metal: `systemctl restart koris` or re-run the binary).\n")
	}
	return nil
}

// certEnvPath resolves the panel env file to write cert settings into.
// Inside Docker the KORIS_HOME volume is mounted at /etc/koris, so prefer that;
// bare-metal uses /opt/koris. KORIS_HOME (if set) wins.
func certEnvPath() string {
	if home := os.Getenv("KORIS_HOME"); home != "" {
		return filepath.Join(home, ".env")
	}
	for _, p := range []string{"/etc/koris/.env", "/opt/koris/.env"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "/opt/koris/.env"
}

// updateEnvFile rewrites the env file, setting/adding the given keys while
// preserving all other lines (and comments).
func updateEnvFile(path string, vars map[string]string) error {
	existing := map[string]string{}
	var lines []string

	if f, err := os.Open(path); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if idx := strings.Index(line, "="); idx > 0 {
				key := strings.TrimSpace(line[:idx])
				if _, ok := vars[key]; ok {
					existing[key] = line
					continue // drop; will be re-added below
				}
			}
			lines = append(lines, line)
		}
		f.Close()
		if err := scanner.Err(); err != nil {
			return err
		}
	}

	// Append the new/updated vars.
	for k, v := range vars {
		lines = append(lines, fmt.Sprintf("%s=%s", k, v))
	}
	lines = append(lines, "")

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600)
}

// restartPanel attempts to restart the running panel. Returns true if it
// triggered a Docker restart.
func restartPanel() bool {
	// Docker deployment: container named "koris".
	if _, err := exec.LookPath("docker"); err == nil {
		cmd := exec.Command("docker", "ps", "--filter", "name=^koris$", "--format", "{{.Names}}")
		out, err := cmd.Output()
		if err == nil && strings.TrimSpace(string(out)) == "koris" {
			rc := exec.Command("docker", "restart", "koris")
			rc.Stdout = nil
			rc.Stderr = nil
			return rc.Run() == nil
		}
	}
	return false
}

func printCertUsage(c *CLI) error {
	w := c.Output()
	fmt.Fprintf(w, "Usage: koris cert <subcommand> [flags]\n\n")
	fmt.Fprintf(w, "Install a TLS certificate to expose the panel over HTTPS. The default\n")
	fmt.Fprintf(w, "install serves plaintext HTTP on 127.0.0.1 only; run one of these to switch.\n\n")
	fmt.Fprintf(w, "Subcommands:\n")
	fmt.Fprintf(w, "  selfsign            Auto-generate a self-signed cert (DEV; browsers warn)\n")
	fmt.Fprintf(w, "  letsencrypt         Let's Encrypt (ACME) — needs --domain=<real domain>\n")
	fmt.Fprintf(w, "  path                Use your own cert/key — needs --cert= --key=\n\n")
	fmt.Fprintf(w, "Flags:\n")
	fmt.Fprintf(w, "  --domain=<domain>   Domain for letsencrypt\n")
	fmt.Fprintf(w, "  --email=<email>     Contact email for letsencrypt (optional)\n")
	fmt.Fprintf(w, "  --cert=<path>       Certificate PEM for path mode\n")
	fmt.Fprintf(w, "  --key=<path>        Private key PEM for path mode\n")
	fmt.Fprintf(w, "  --home=<dir>        Override KORIS_HOME (env file location)\n")
	return nil
}
