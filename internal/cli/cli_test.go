package cli

import (
	"bytes"
	"errors"
	"net/http"
	"testing"
)

func TestNew_Defaults(t *testing.T) {
	c := New()
	if c.output == nil {
		t.Error("output should default to os.Stdout, got nil")
	}
	if c.socketPath != "/var/run/panel.sock" {
		t.Errorf("socketPath = %q, want /var/run/panel.sock", c.socketPath)
	}
	if c.client == nil {
		t.Error("client should default to http.DefaultClient, got nil")
	}
	if c.jsonOutput {
		t.Error("jsonOutput should default to false")
	}
}

func TestNew_WithOptions(t *testing.T) {
	var buf bytes.Buffer
	client := &http.Client{}

	c := New(
		WithOutput(&buf),
		WithSocketPath("/tmp/test.sock"),
		WithHTTPClient(client),
		WithJSONOutput(true),
	)

	if c.output != &buf {
		t.Error("WithOutput not applied")
	}
	if c.socketPath != "/tmp/test.sock" {
		t.Errorf("socketPath = %q, want /tmp/test.sock", c.socketPath)
	}
	if c.client != client {
		t.Error("WithHTTPClient not applied")
	}
	if !c.jsonOutput {
		t.Error("WithJSONOutput not applied")
	}
}

func TestCLI_RegisterCommand(t *testing.T) {
	c := New()
	c.RegisterCommand(Command{Name: "status", Description: "Show status"})
	c.RegisterCommand(Command{Name: "nodes", Description: "List nodes"})

	if len(c.commands) != 2 {
		t.Fatalf("commands len = %d, want 2", len(c.commands))
	}
	if c.commands[0].Name != "status" {
		t.Errorf("commands[0].Name = %q, want status", c.commands[0].Name)
	}
	if c.commands[1].Name != "nodes" {
		t.Errorf("commands[1].Name = %q, want nodes", c.commands[1].Name)
	}
}

func TestCLI_Execute_EmptyArgs(t *testing.T) {
	var buf bytes.Buffer
	c := New(WithOutput(&buf))
	c.RegisterCommand(Command{Name: "status", Description: "Show status"})

	err := c.Execute([]string{})
	if err != nil {
		t.Fatalf("Execute([]) returned error: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("Available commands")) {
		t.Error("empty args should print usage")
	}
}

func TestCLI_Execute_UnknownCommand(t *testing.T) {
	c := New()
	err := c.Execute([]string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
	if err.Error() != "unknown command: nonexistent" {
		t.Errorf("error = %q, want 'unknown command: nonexistent'", err.Error())
	}
}

func TestCLI_Execute_CommandDispatch(t *testing.T) {
	var called bool
	var receivedArgs []string

	c := New()
	c.RegisterCommand(Command{
		Name: "status",
		Run: func(args []string) error {
			called = true
			receivedArgs = args
			return nil
		},
	})

	err := c.Execute([]string{"status"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !called {
		t.Error("command Run was not called")
	}
	if len(receivedArgs) != 0 {
		t.Errorf("args = %v, want empty", receivedArgs)
	}
}

func TestCLI_Execute_CommandWithFlags(t *testing.T) {
	var receivedArgs []string

	c := New()
	c.RegisterCommand(Command{
		Name: "cleanup",
		Run: func(args []string) error {
			receivedArgs = args
			return nil
		},
	})

	err := c.Execute([]string{"cleanup", "--dry-run"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	// The --dry-run flag should be passed through to the command.
	found := false
	for _, arg := range receivedArgs {
		if arg == "--dry-run" {
			found = true
		}
	}
	if !found {
		t.Errorf("args = %v, expected --dry-run to be passed", receivedArgs)
	}
}

func TestCLI_Execute_SubCommand(t *testing.T) {
	var subCalled bool

	c := New()
	c.RegisterCommand(Command{
		Name: "users",
		SubCommands: []Command{
			{
				Name: "list",
				Run: func(args []string) error {
					subCalled = true
					return nil
				},
			},
		},
	})

	err := c.Execute([]string{"users", "list"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !subCalled {
		t.Error("subcommand Run was not called")
	}
}

func TestCLI_Execute_SubCommandWithFlags(t *testing.T) {
	var receivedArgs []string

	c := New()
	c.RegisterCommand(Command{
		Name: "users",
		SubCommands: []Command{
			{
				Name: "list",
				Run: func(args []string) error {
					receivedArgs = args
					return nil
				},
			},
		},
	})

	err := c.Execute([]string{"users", "list", "--status=active"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	found := false
	for _, arg := range receivedArgs {
		if arg == "--status=active" {
			found = true
		}
	}
	if !found {
		t.Errorf("args = %v, expected --status=active", receivedArgs)
	}
}

func TestCLI_Execute_JSONFlag(t *testing.T) {
	c := New()
	c.RegisterCommand(Command{
		Name: "status",
		Run:  func(args []string) error { return nil },
	})

	err := c.Execute([]string{"status", "--json"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !c.jsonOutput {
		t.Error("--json flag should set jsonOutput=true")
	}
}

func TestCLI_Execute_HelpFlag(t *testing.T) {
	var buf bytes.Buffer
	c := New(WithOutput(&buf))
	c.RegisterCommand(Command{Name: "status", Description: "Show status"})

	err := c.Execute([]string{"--help"})
	if err != nil {
		t.Fatalf("Execute(--help) returned error: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("Available commands")) {
		t.Error("--help should print usage")
	}
}

func TestCLI_Execute_CommandError(t *testing.T) {
	c := New()
	c.RegisterCommand(Command{
		Name: "fail",
		Run: func(args []string) error {
			return errors.New("something broke")
		},
	})

	err := c.Execute([]string{"fail"})
	if err == nil {
		t.Fatal("expected error from failing command")
	}
	if err.Error() != "something broke" {
		t.Errorf("error = %q, want 'something broke'", err.Error())
	}
}

func TestCLI_Execute_UnknownSubCommand(t *testing.T) {
	var called bool
	c := New()
	c.RegisterCommand(Command{
		Name: "nodes",
		SubCommands: []Command{
			{Name: "info", Run: func(args []string) error { return nil }},
		},
		Run: func(args []string) error {
			// Unknown subcommand treated as positional, passed to parent Run.
			called = true
			return nil
		},
	})

	err := c.Execute([]string{"nodes", "unknown"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !called {
		t.Error("parent Run should be called when subcommand not found")
	}
}

func TestCLI_Accessors(t *testing.T) {
	var buf bytes.Buffer
	client := &http.Client{}
	c := New(
		WithOutput(&buf),
		WithSocketPath("/tmp/panel.sock"),
		WithHTTPClient(client),
		WithJSONOutput(true),
	)

	if c.Output() != &buf {
		t.Error("Output() mismatch")
	}
	if c.SocketPath() != "/tmp/panel.sock" {
		t.Errorf("SocketPath() = %q, want /tmp/panel.sock", c.SocketPath())
	}
	if c.Client() != client {
		t.Error("Client() mismatch")
	}
	if !c.JSONOutput() {
		t.Error("JSONOutput() should be true")
	}
}
