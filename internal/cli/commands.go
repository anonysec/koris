package cli

// RegisterDefaultCommands registers all built-in commands with the CLI instance.
// Call this after creating a CLI to enable the standard command set.
func RegisterDefaultCommands(c *CLI) {
	c.RegisterCommand(StatusCommand(c))
	c.RegisterCommand(NodesCommand(c))
	c.RegisterCommand(UsersCommand(c))
	c.RegisterCommand(AdminCommand(c))
	c.RegisterCommand(CleanupCommand(c))
	c.RegisterCommand(WorkersCommand(c))
	c.RegisterCommand(LogsCommand(c))
	c.RegisterCommand(UpdateCommand(c))
	c.RegisterCommand(CertCommand(c))
}
