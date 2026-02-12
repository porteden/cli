package main

import (
	"github.com/porteden/cli/internal/commands"
	"github.com/porteden/cli/internal/version"
)

func main() {
	// Check for updates in the background
	version.CheckForUpdate()

	// Execute the CLI
	commands.Execute()
}
