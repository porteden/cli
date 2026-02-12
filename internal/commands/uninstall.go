package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/porteden/cli/internal/output"
	"github.com/porteden/cli/internal/system"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the CLI",
	Long: `Uninstall the PortEden CLI.

The uninstall method is automatically detected based on how you installed the CLI:
  - Homebrew:     runs 'brew uninstall porteden'
  - Go / Script:  removes the binary file

Use --purge to also remove configuration and stored credentials.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		purge, _ := cmd.Flags().GetBool("purge")
		yes, _ := cmd.Flags().GetBool("yes")
		return runUninstall(purge, yes)
	},
}

func init() {
	uninstallCmd.Flags().Bool("purge", false, "Also remove configuration and stored credentials")
	uninstallCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
}

func runUninstall(purge, yes bool) error {
	method := system.DetectInstallMethod()

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine binary path: %w", err)
	}

	// Show what will happen
	fmt.Println("This will uninstall PortEden CLI:")
	switch method {
	case system.InstallHomebrew:
		fmt.Println("  - Run 'brew uninstall porteden'")
	default:
		fmt.Printf("  - Remove binary: %s\n", exePath)
	}
	if purge {
		home, _ := os.UserHomeDir()
		fmt.Printf("  - Remove config: %s\n", filepath.Join(home, ".config", "porteden"))
	}
	fmt.Println()

	// Confirm
	if !yes {
		fmt.Print("Continue? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Execute
	switch method {
	case system.InstallHomebrew:
		cmd := exec.Command("brew", "uninstall", "porteden")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("brew uninstall failed: %w", err)
		}
	default:
		if err := os.Remove(exePath); err != nil {
			return fmt.Errorf("failed to remove binary: %w", err)
		}
	}

	// Purge config if requested
	if purge {
		home, _ := os.UserHomeDir()
		configDir := filepath.Join(home, ".config", "porteden")
		if err := os.RemoveAll(configDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove config directory: %v\n", err)
		} else {
			output.PrintSuccess("Removed configuration directory")
		}
	}

	output.PrintSuccess("PortEden CLI has been uninstalled.")
	return nil
}
