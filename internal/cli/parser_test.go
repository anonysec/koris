package cli

import (
	"reflect"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantCmd   string
		wantSub   string
		wantFlags map[string]string
		wantPos   []string
	}{
		{
			name:      "empty args",
			args:      []string{},
			wantCmd:   "",
			wantSub:   "",
			wantFlags: map[string]string{},
			wantPos:   nil,
		},
		{
			name:      "single command",
			args:      []string{"status"},
			wantCmd:   "status",
			wantSub:   "",
			wantFlags: map[string]string{},
			wantPos:   nil,
		},
		{
			name:      "command with subcommand",
			args:      []string{"users", "list"},
			wantCmd:   "users",
			wantSub:   "list",
			wantFlags: map[string]string{},
			wantPos:   nil,
		},
		{
			name:      "command with boolean flag",
			args:      []string{"cleanup", "--dry-run"},
			wantCmd:   "cleanup",
			wantSub:   "",
			wantFlags: map[string]string{"dry-run": ""},
			wantPos:   nil,
		},
		{
			name:      "command with inline flag value",
			args:      []string{"users", "list", "--status=active"},
			wantCmd:   "users",
			wantSub:   "list",
			wantFlags: map[string]string{"status": "active"},
			wantPos:   nil,
		},
		{
			name:      "command with space-separated flag value",
			args:      []string{"logs", "--tail=100"},
			wantCmd:   "logs",
			wantSub:   "",
			wantFlags: map[string]string{"tail": "100"},
			wantPos:   nil,
		},
		{
			name:      "short flag with value",
			args:      []string{"logs", "-n", "50"},
			wantCmd:   "logs",
			wantSub:   "",
			wantFlags: map[string]string{"n": ""},
			wantPos:   []string{"50"},
		},
		{
			name:      "short boolean flag",
			args:      []string{"cleanup", "-v"},
			wantCmd:   "cleanup",
			wantSub:   "",
			wantFlags: map[string]string{"v": ""},
			wantPos:   nil,
		},
		{
			name:      "multiple flags",
			args:      []string{"cleanup", "--older-than=90d", "--confirm"},
			wantCmd:   "cleanup",
			wantSub:   "",
			wantFlags: map[string]string{"older-than": "90d", "confirm": ""},
			wantPos:   nil,
		},
		{
			name:      "subcommand with flags",
			args:      []string{"users", "list", "--status=expired", "--json"},
			wantCmd:   "users",
			wantSub:   "list",
			wantFlags: map[string]string{"status": "expired", "json": ""},
			wantPos:   nil,
		},
		{
			name:      "command with positional args after subcommand",
			args:      []string{"nodes", "info", "42"},
			wantCmd:   "nodes",
			wantSub:   "info",
			wantFlags: map[string]string{},
			wantPos:   []string{"42"},
		},
		{
			name:      "mixed flags and positional",
			args:      []string{"users", "info", "--json", "admin"},
			wantCmd:   "users",
			wantSub:   "info",
			wantFlags: map[string]string{"json": ""},
			wantPos:   []string{"admin"},
		},
		{
			name:      "global json flag before command",
			args:      []string{"--json", "status"},
			wantCmd:   "status",
			wantSub:   "",
			wantFlags: map[string]string{"json": ""},
			wantPos:   nil,
		},
		{
			name:      "flag with empty value via equals",
			args:      []string{"config", "--key="},
			wantCmd:   "config",
			wantSub:   "",
			wantFlags: map[string]string{"key": ""},
			wantPos:   nil,
		},
		{
			name:      "flag value with special characters",
			args:      []string{"config", "--url=http://127.0.0.1:8080/path?q=1"},
			wantCmd:   "config",
			wantSub:   "",
			wantFlags: map[string]string{"url": "http://127.0.0.1:8080/path?q=1"},
			wantPos:   nil,
		},
		{
			name:      "flag value with spaces via equals",
			args:      []string{"config", "--name=my server"},
			wantCmd:   "config",
			wantSub:   "",
			wantFlags: map[string]string{"name": "my server"},
			wantPos:   nil,
		},
		{
			name:      "multiple short flags",
			args:      []string{"logs", "-v", "-f"},
			wantCmd:   "logs",
			wantSub:   "",
			wantFlags: map[string]string{"v": "", "f": ""},
			wantPos:   nil,
		},
		{
			name:      "real world: koris cleanup --dry-run",
			args:      []string{"cleanup", "--dry-run"},
			wantCmd:   "cleanup",
			wantSub:   "",
			wantFlags: map[string]string{"dry-run": ""},
			wantPos:   nil,
		},
		{
			name:      "real world: koris cleanup --older-than=90d --confirm",
			args:      []string{"cleanup", "--older-than=90d", "--confirm"},
			wantCmd:   "cleanup",
			wantSub:   "",
			wantFlags: map[string]string{"older-than": "90d", "confirm": ""},
			wantPos:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, sub, flags, pos := ParseArgs(tt.args)
			if cmd != tt.wantCmd {
				t.Errorf("cmd = %q, want %q", cmd, tt.wantCmd)
			}
			if sub != tt.wantSub {
				t.Errorf("subCmd = %q, want %q", sub, tt.wantSub)
			}
			if !reflect.DeepEqual(flags, tt.wantFlags) {
				t.Errorf("flags = %v, want %v", flags, tt.wantFlags)
			}
			if !reflect.DeepEqual(pos, tt.wantPos) {
				t.Errorf("positional = %v, want %v", pos, tt.wantPos)
			}
		})
	}
}

func TestParseFlagsFromArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantFlags map[string]string
		wantPos   []string
	}{
		{
			name:      "empty",
			args:      []string{},
			wantFlags: map[string]string{},
			wantPos:   nil,
		},
		{
			name:      "only positional",
			args:      []string{"foo", "bar"},
			wantFlags: map[string]string{},
			wantPos:   []string{"foo", "bar"},
		},
		{
			name:      "flags and positional",
			args:      []string{"--status=active", "john"},
			wantFlags: map[string]string{"status": "active"},
			wantPos:   []string{"john"},
		},
		{
			name:      "boolean flag",
			args:      []string{"--dry-run"},
			wantFlags: map[string]string{"dry-run": ""},
			wantPos:   nil,
		},
		{
			name:      "space-separated long flag value",
			args:      []string{"--tail", "100"},
			wantFlags: map[string]string{"tail": "100"},
			wantPos:   nil,
		},
		{
			name:      "short flag with space-separated value",
			args:      []string{"-n", "50"},
			wantFlags: map[string]string{"n": "50"},
			wantPos:   nil,
		},
		{
			name:      "multiple space-separated flags",
			args:      []string{"--status", "active", "--limit", "25"},
			wantFlags: map[string]string{"status": "active", "limit": "25"},
			wantPos:   nil,
		},
		{
			name:      "mixed inline and space-separated flags",
			args:      []string{"--format=json", "--tail", "200", "--verbose"},
			wantFlags: map[string]string{"format": "json", "tail": "200", "verbose": ""},
			wantPos:   nil,
		},
		{
			name:      "flag value with special characters",
			args:      []string{"--filter=user@host:8080/path"},
			wantFlags: map[string]string{"filter": "user@host:8080/path"},
			wantPos:   nil,
		},
		{
			name:      "positional between flags",
			args:      []string{"--json", "admin", "--verbose"},
			wantFlags: map[string]string{"json": "admin", "verbose": ""},
			wantPos:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags, pos := ParseFlagsFromArgs(tt.args)
			if !reflect.DeepEqual(flags, tt.wantFlags) {
				t.Errorf("flags = %v, want %v", flags, tt.wantFlags)
			}
			if !reflect.DeepEqual(pos, tt.wantPos) {
				t.Errorf("positional = %v, want %v", pos, tt.wantPos)
			}
		})
	}
}
