package cli

import (
	"io"
	"net/http"
)

// Option is a functional option for configuring the CLI.
type Option func(*CLI)

// WithOutput sets the io.Writer where CLI output is written.
// Defaults to os.Stdout.
func WithOutput(w io.Writer) Option {
	return func(c *CLI) {
		c.output = w
	}
}

// WithSocketPath sets the Unix socket path for communicating with
// the running panel. Defaults to "/var/run/panel.sock".
func WithSocketPath(path string) Option {
	return func(c *CLI) {
		c.socketPath = path
	}
}

// WithHTTPClient sets the HTTP client used for API calls to the
// running panel (over Unix socket or TCP fallback).
func WithHTTPClient(client *http.Client) Option {
	return func(c *CLI) {
		c.client = client
	}
}

// WithJSONOutput forces JSON output mode. This has the same
// effect as passing --json on the command line.
func WithJSONOutput(enabled bool) Option {
	return func(c *CLI) {
		c.jsonOutput = enabled
	}
}
