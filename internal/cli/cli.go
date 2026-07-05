// Package cli provides a command-line interface framework for panel management.
// It handles argument parsing, command dispatch, and formatted output for
// commands like `koris status`, `koris nodes`, `koris users list`, etc.
//
// Usage:
//
//	c := cli.New(
//	    cli.WithOutput(os.Stdout),
//	    cli.WithSocketPath("/var/run/panel.sock"),
//	)
//	c.RegisterCommand(cli.Command{Name: "status", Run: statusFn})
//	c.Execute(os.Args[1:])
package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// CLI provides the command-line interface for panel management.
// It maintains a registry of commands and handles argument parsing
// and dispatch.
type CLI struct {
	output     io.Writer
	commands   []Command
	jsonOutput bool
	client     *http.Client
	socketPath string
}

// New creates a new CLI instance with the given options applied.
// Defaults: output=os.Stdout, socketPath="/var/run/panel.sock".
func New(opts ...Option) *CLI {
	c := &CLI{
		output:     os.Stdout,
		socketPath: "/var/run/panel.sock",
		client:     http.DefaultClient,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// RegisterCommand adds a command to the CLI registry.
func (c *CLI) RegisterCommand(cmd Command) {
	c.commands = append(c.commands, cmd)
}

// Execute parses the given arguments and dispatches to the matching command.
// It handles global flags like --json before command dispatch.
func (c *CLI) Execute(args []string) error {
	if len(args) == 0 {
		return c.printUsage()
	}

	cmdName, subCmd, flags, positional := ParseArgs(args)

	// Handle global flags.
	if _, ok := flags["json"]; ok {
		c.jsonOutput = true
		delete(flags, "json")
	}

	// Handle help flag.
	if _, ok := flags["help"]; ok {
		return c.printUsage()
	}
	if cmdName == "help" {
		return c.printUsage()
	}

	// Find matching command.
	cmd, found := c.findCommand(cmdName)
	if !found {
		return fmt.Errorf("unknown command: %s", cmdName)
	}

	// If there's a subcommand, look for it in the command's SubCommands.
	if subCmd != "" {
		subCommand, subFound := findSubCommand(cmd, subCmd)
		if subFound {
			if subCommand.Run == nil {
				return fmt.Errorf("command %s %s has no handler", cmdName, subCmd)
			}
			return subCommand.Run(buildRunArgs(flags, positional))
		}
		// If subcommand not found, treat it as a positional argument.
		positional = append([]string{subCmd}, positional...)
	}

	if cmd.Run == nil {
		return c.printCommandUsage(cmd)
	}

	return cmd.Run(buildRunArgs(flags, positional))
}

// JSONOutput returns whether --json flag was set.
func (c *CLI) JSONOutput() bool {
	return c.jsonOutput
}

// Output returns the configured output writer.
func (c *CLI) Output() io.Writer {
	return c.output
}

// SocketPath returns the configured Unix socket path.
func (c *CLI) SocketPath() string {
	return c.socketPath
}

// Client returns the configured HTTP client.
func (c *CLI) Client() *http.Client {
	return c.client
}

// findCommand looks up a command by name in the registry.
func (c *CLI) findCommand(name string) (Command, bool) {
	for _, cmd := range c.commands {
		if cmd.Name == name {
			return cmd, true
		}
	}
	return Command{}, false
}

// findSubCommand looks up a subcommand within a command.
func findSubCommand(cmd Command, name string) (Command, bool) {
	for _, sub := range cmd.SubCommands {
		if sub.Name == name {
			return sub, true
		}
	}
	return Command{}, false
}

// buildRunArgs reconstructs an argument slice from flags and positional args
// suitable for passing to a command's Run function.
func buildRunArgs(flags map[string]string, positional []string) []string {
	var args []string
	for k, v := range flags {
		if v == "" {
			args = append(args, "--"+k)
		} else {
			args = append(args, "--"+k+"="+v)
		}
	}
	args = append(args, positional...)
	return args
}

// printUsage outputs the list of available commands.
func (c *CLI) printUsage() error {
	var sb strings.Builder
	sb.WriteString("Usage: koris <command> [subcommand] [flags]\n\n")
	sb.WriteString("Available commands:\n")
	for _, cmd := range c.commands {
		sb.WriteString(fmt.Sprintf("  %-14s %s\n", cmd.Name, cmd.Description))
	}
	sb.WriteString("\nGlobal flags:\n")
	sb.WriteString("  --json         Output in JSON format\n")
	sb.WriteString("  --help         Show this help message\n")
	_, err := fmt.Fprint(c.output, sb.String())
	return err
}

// printCommandUsage outputs help for a specific command.
func (c *CLI) printCommandUsage(cmd Command) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Usage: koris %s", cmd.Name))
	if len(cmd.SubCommands) > 0 {
		sb.WriteString(" <subcommand>")
	}
	sb.WriteString(" [flags]\n\n")
	if cmd.Description != "" {
		sb.WriteString(cmd.Description + "\n\n")
	}
	if len(cmd.SubCommands) > 0 {
		sb.WriteString("Subcommands:\n")
		for _, sub := range cmd.SubCommands {
			sb.WriteString(fmt.Sprintf("  %-14s %s\n", sub.Name, sub.Description))
		}
	}
	if len(cmd.Flags) > 0 {
		sb.WriteString("\nFlags:\n")
		for _, f := range cmd.Flags {
			name := "--" + f.Name
			if f.Short != "" {
				name = "-" + f.Short + ", " + name
			}
			sb.WriteString(fmt.Sprintf("  %-20s %s\n", name, f.Description))
		}
	}
	_, err := fmt.Fprint(c.output, sb.String())
	return err
}
