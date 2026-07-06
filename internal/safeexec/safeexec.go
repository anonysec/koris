// Package safeexec provides validated command execution to prevent command injection.
// All exec.Command and exec.CommandContext calls should use this package.
package safeexec

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// allowedCommands is the set of commands that are permitted to execute.
// This acts as an allowlist to prevent arbitrary command execution.
var allowedCommands = map[string]bool{
	// System
	"systemctl": true, "journalctl": true, "killall": true, "pgrep": true,
	// Package management
	"apk": true, "apt-get": true, "dpkg": true,
	// Network
	"iptables": true, "ip6tables": true, "ip": true, "tc": true, "wg": true,
	"wg-quick": true, "ss": true,
	"ipsec": true,
	"xl2tpd": true,
	"haproxy": true,
	"nft": true,
	// VPN
	"openvpn": true, "easyrsa": true, "swanctl": true,
	// TLS/Certs
	"openssl": true, "certbot": true,
	// Database
	"pg_dump": true, "psql": true,
	// RADIUS
	"radclient": true,
	// Utilities
	"cat": true, "chmod": true, "cp": true, "mkdir": true, "rm": true,
	"sh": true, "bash": true,
}

// Validate checks that a command name is in the allowlist.
func Validate(name string) error {
	base := filepath.Base(name)
	if allowedCommands[base] || allowedCommands[name] {
		return nil
	}
	return fmt.Errorf("command %q not in allowlist", name)
}

// Command creates an exec.Cmd after validating the command name against the allowlist.
func Command(name string, args ...string) (*exec.Cmd, error) {
	if err := Validate(name); err != nil {
		return nil, err
	}
	return exec.Command(name, args...), nil // #nosec G204 -- command name validated by Validate() above
}

// CommandContext creates an exec.Cmd with context after validating the command name.
func CommandContext(ctx context.Context, name string, args ...string) (*exec.Cmd, error) {
	if err := Validate(name); err != nil {
		return nil, err
	}
	return exec.CommandContext(ctx, name, args...), nil // #nosec G204 -- command name validated by Validate() above
}

// MustCommand is like Command but panics on validation failure.
// Use for hardcoded command names that should always be valid.
func MustCommand(name string, args ...string) *exec.Cmd {
	cmd, err := Command(name, args...)
	if err != nil {
		panic(fmt.Sprintf("safeexec: %v", err))
	}
	return cmd
}

// MustCommandContext is like CommandContext but panics on validation failure.
func MustCommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd, err := CommandContext(ctx, name, args...)
	if err != nil {
		panic(fmt.Sprintf("safeexec: %v", err))
	}
	return cmd
}

// RegisterCommand adds a command to the allowlist at runtime.
// Use for dynamically discovered binary paths (e.g., /usr/sbin/ipsec).
func RegisterCommand(name string) {
	base := filepath.Base(name)
	allowedCommands[base] = true
	allowedCommands[name] = true
	// Also register without path
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		allowedCommands[name[idx+1:]] = true
	}
}
