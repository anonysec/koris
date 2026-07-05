package cli

// Command represents a CLI command with optional subcommands and flags.
type Command struct {
	// Name is the command name as typed by the user (e.g. "status", "nodes").
	Name string

	// Description is a short one-line summary shown in help output.
	Description string

	// SubCommands are nested commands (e.g. "users list", "users info").
	SubCommands []Command

	// Flags defines the accepted flags for this command.
	Flags []Flag

	// Run is the function called when this command is matched.
	// It receives the remaining arguments (flags + positional) after parsing.
	// A nil Run indicates the command only has subcommands.
	Run func(args []string) error
}

// Flag describes a command-line flag (e.g. --dry-run, --status=active).
type Flag struct {
	// Name is the long flag name without the -- prefix (e.g. "dry-run").
	Name string

	// Short is the optional single-character shorthand without - prefix (e.g. "n").
	Short string

	// Description is a short explanation shown in help output.
	Description string

	// HasValue indicates whether the flag expects a value (--flag=value).
	// When false, the flag is treated as a boolean toggle.
	HasValue bool

	// Default is the default value when the flag is not provided.
	Default string
}
