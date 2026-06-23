package cli

import "strings"

// ParseArgs parses a slice of arguments into structured components.
// It identifies the command, optional subcommand, flags, and positional arguments.
//
// Supported patterns:
//   - koris status                     → cmd="status"
//   - koris nodes                      → cmd="nodes"
//   - koris users list                 → cmd="users", subCmd="list"
//   - koris cleanup --dry-run          → cmd="cleanup", flags={"dry-run": ""}
//   - koris users list --status=active → cmd="users", subCmd="list", flags={"status": "active"}
//   - koris nodes info 42              → cmd="nodes", subCmd="info", positional=["42"]
//   - koris --json status              → cmd="status", flags={"json": ""}
//
// Flag formats:
//   - --flag=value  (long flag with inline value)
//   - --flag        (boolean flag, no value)
//   - -f            (short boolean flag)
//
// For commands that need --flag value (space-separated) parsing within their
// Run handler, use ParseFlagsFromArgs on the args slice passed to Run.
func ParseArgs(args []string) (cmd string, subCmd string, flags map[string]string, positional []string) {
	flags = make(map[string]string)

	if len(args) == 0 {
		return
	}

	// First non-flag argument is the command.
	// Pre-command flags are always treated as boolean (no value consumption).
	idx := 0
	for idx < len(args) && isFlag(args[idx]) {
		idx = parseBoolFlag(args, idx, flags)
	}

	if idx >= len(args) {
		return
	}

	cmd = args[idx]
	idx++

	// Second non-flag argument is treated as a potential subcommand.
	// We peek ahead: if the next arg is not a flag, it could be a subcommand.
	if idx < len(args) && !isFlag(args[idx]) {
		subCmd = args[idx]
		idx++
	}

	// Parse remaining arguments.
	// Post-command flags use --flag=value for values. Bare --flag is boolean.
	for idx < len(args) {
		if isFlag(args[idx]) {
			idx = parseBoolFlag(args, idx, flags)
		} else {
			positional = append(positional, args[idx])
			idx++
		}
	}

	return
}

// isFlag returns true if the argument looks like a flag (starts with - or --).
func isFlag(arg string) bool {
	return len(arg) > 1 && arg[0] == '-'
}

// parseBoolFlag parses a flag that only gets its value from --flag=value syntax.
// Bare --flag or -f flags are treated as boolean (empty string value).
// This avoids ambiguity where the next positional arg could be mistaken for a flag value.
func parseBoolFlag(args []string, idx int, flags map[string]string) int {
	arg := args[idx]

	if strings.HasPrefix(arg, "--") {
		name := arg[2:]
		if eqIdx := strings.IndexByte(name, '='); eqIdx >= 0 {
			flags[name[:eqIdx]] = name[eqIdx+1:]
			return idx + 1
		}
		flags[name] = ""
		return idx + 1
	}

	// Short flag: -f (always boolean in ParseArgs context).
	name := arg[1:]
	flags[name] = ""
	return idx + 1
}

// parseFlag parses a single flag starting at args[idx] and returns the next index.
// It handles --flag=value, --flag value, --flag (boolean), -f value, and -f (boolean).
func parseFlag(args []string, idx int, flags map[string]string) int {
	arg := args[idx]

	if strings.HasPrefix(arg, "--") {
		// Long flag.
		name := arg[2:]
		if eqIdx := strings.IndexByte(name, '='); eqIdx >= 0 {
			// --flag=value
			flags[name[:eqIdx]] = name[eqIdx+1:]
			return idx + 1
		}
		// --flag or --flag value
		// Peek at next arg to determine if it's a value or another flag.
		if idx+1 < len(args) && !isFlag(args[idx+1]) {
			// Could be --flag value, but we need to distinguish from boolean flags.
			// Heuristic: if the next arg doesn't start with -, treat it as a value.
			flags[name] = args[idx+1]
			return idx + 2
		}
		// Boolean flag.
		flags[name] = ""
		return idx + 1
	}

	// Short flag: -f or -f value
	name := arg[1:]
	if idx+1 < len(args) && !isFlag(args[idx+1]) {
		flags[name] = args[idx+1]
		return idx + 2
	}
	// Boolean short flag.
	flags[name] = ""
	return idx + 1
}

// ParseFlagsFromArgs extracts flag values from a pre-parsed args slice.
// This is a convenience for command Run functions that receive the
// rebuilt args from Execute.
func ParseFlagsFromArgs(args []string) (flags map[string]string, positional []string) {
	flags = make(map[string]string)
	for i := 0; i < len(args); i++ {
		if isFlag(args[i]) {
			i = parseFlag(args, i, flags) - 1 // -1 because loop will increment
		} else {
			positional = append(positional, args[i])
		}
	}
	return
}
