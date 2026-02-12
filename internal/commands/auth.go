package commands

import (
	"bufio"
	"fmt"
	"os"

	"github.com/porteden/cli/internal/api"
	"github.com/porteden/cli/internal/auth"
	"github.com/porteden/cli/internal/debug"
	"github.com/porteden/cli/internal/output"
	"github.com/spf13/cobra"
)

// requireStore ensures credential store is initialized for commands that need write access.
// Returns error if PE_API_KEY is set, since it bypasses the stored credential system.
func requireStore() error {
	if os.Getenv("PE_API_KEY") != "" {
		return fmt.Errorf("this command requires credential store access but PE_API_KEY is set\n" +
			"PE_API_KEY bypasses the credential store and cannot be used with profile management commands.\n" +
			"Either:\n" +
			"  - Unset PE_API_KEY to use profile management\n" +
			"  - Use PE_API_KEY for direct API access (incompatible with profiles)")
	}
	return auth.InitStore()
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
}

var loginCmd = &cobra.Command{
	Use:           "login",
	Short:         "Authenticate via browser or API key",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `Authenticate with PortEden.

Two authentication methods:
  1. Browser OAuth (default): Opens browser for secure login
  2. Direct token: Pass --token flag for non-interactive setup (CI/automation)

Examples:
  porteden auth login                    # Browser OAuth
  porteden auth login --token pe_xxx     # Direct token
  porteden auth login --profile work     # Named profile`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Login always needs credential store — bypass the PE_API_KEY check that
		// requireStore() enforces, since re-authenticating is legitimate
		// even when PE_API_KEY is set.
		if err := auth.InitStore(); err != nil {
			return err
		}

		token, _ := cmd.Flags().GetString("token")
		keyTitle, _ := cmd.Flags().GetString("title")
		profileName := getProfile(cmd)

		// Delete existing key before re-authenticating
		if existingKey, err := auth.GetStoredAPIKey(profileName); err == nil && existingKey != "" {
			if err := auth.DeleteAPIKey(profileName); err != nil {
				debug.Log("Failed to delete existing key for profile '%s': %v", profileName, err)
			}
		}

		// Direct token authentication — minimal output, no wizard
		if token != "" {
			if err := auth.StoreAPIKey(token, profileName); err != nil {
				return fmt.Errorf("failed to store API key: %w", err)
			}
			if err := auth.SetActiveProfile(profileName); err != nil {
				return fmt.Errorf("failed to set active profile: %w", err)
			}
			output.PrintSuccess(fmt.Sprintf("API key stored in profile '%s'", profileName))
			return nil
		}

		// Browser OAuth wizard flow
		if _, err := runLoginWizard(profileName, keyTitle); err != nil {
			return err
		}
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := getProfile(cmd)
		apiKey, err := auth.GetAPIKey(profileName)
		if err != nil {
			fmt.Printf("Not authenticated (profile: %s). Run 'porteden auth login' to authenticate.\n", profileName)
			return nil
		}

		client := api.NewClient(apiKey)
		status, err := client.GetAuthStatus()
		if err != nil {
			return err
		}

		fmt.Printf("Profile: %s\n", profileName)
		fmt.Printf("Authenticated as: %s\n", status.Email)
		fmt.Printf("Operator: %s\n", status.OperatorName)
		fmt.Printf("Key ID: %d\n", status.KeyID)
		if status.KeyTitle != "" {
			fmt.Printf("Key title: %s\n", status.KeyTitle)
		}
		fmt.Printf("Key created: %s\n", status.CreatedAt.Format("2006-01-02"))
		return nil
	},
}

var listProfilesCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireStore(); err != nil {
			return err
		}

		profiles, activeProfile, err := auth.ListProfiles()
		if err != nil {
			return err
		}

		if len(profiles) == 0 {
			fmt.Println("No profiles configured. Run 'porteden auth login' to create one.")
			return nil
		}

		fmt.Println("Available profiles:")
		for _, p := range profiles {
			marker := "  "
			if p == activeProfile {
				marker = "* "
			}
			fmt.Printf("%s%s\n", marker, p)
		}
		return nil
	},
}

var useProfileCmd = &cobra.Command{
	Use:   "use <profile>",
	Short: "Switch active profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireStore(); err != nil {
			return err
		}

		profileName := args[0]

		if err := auth.SetActiveProfile(profileName); err != nil {
			return err
		}

		fmt.Printf("Switched to profile '%s'\n", profileName)
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Revoke current API key and remove local credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireStore(); err != nil {
			return err
		}

		profileName := getProfile(cmd)
		apiKey, err := auth.GetAPIKey(profileName)
		if err != nil {
			return fmt.Errorf("not authenticated (profile: %s)", profileName)
		}

		client := api.NewClient(apiKey)
		if err := client.Logout(); err != nil {
			fmt.Printf("Warning: failed to revoke API key on server: %v\n", err)
		}

		if err := auth.DeleteAPIKey(profileName); err != nil {
			return fmt.Errorf("failed to remove local credentials: %w", err)
		}

		fmt.Printf("Logged out from profile '%s'\n", profileName)
		return nil
	},
}

// runLoginWizard runs the full interactive login wizard with banner, steps, and completion.
// Returns the API key on success.
func runLoginWizard(profileName, keyTitle string) (string, error) {
	totalSteps := 2
	if auth.IsInteractiveTerminal() {
		totalSteps = 3 // includes export step
	}

	// Banner & welcome
	output.PrintBanner()
	fmt.Println("  Let's connect your PortEden account.")
	fmt.Println(output.ColorGray("  We'll open your browser to sign in securely."))
	fmt.Println()

	// "Press Enter to continue" for interactive terminals
	if auth.IsInteractiveTerminal() {
		fmt.Print(output.ColorGray("  Press Enter to continue..."))
		if _, err := bufio.NewReader(os.Stdin).ReadBytes('\n'); err != nil {
			debug.Log("Failed to read stdin input: %v", err)
		}
		fmt.Println()
	}

	// Step 1: Open browser
	output.PrintStep(1, totalSteps, "Opening browser...")
	progress := &auth.LoginProgress{
		OnBrowserOpen: func(loginURL string) {
			output.PrintInfo("If it doesn't open, visit: " + loginURL)
		},
		OnWaiting: func() {
			fmt.Println()
			output.PrintStep(2, totalSteps, "Waiting for browser authentication... "+output.ColorGray("Please complete sign-in in your browser."))
		},
	}

	apiKey, err := auth.Login(profileName, "", keyTitle, progress)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n  %s %v\n", output.ColorRed("✗"), err)
		return "", fmt.Errorf("login failed")
	}

	fmt.Println()
	output.PrintSuccess("Authenticated successfully!")
	fmt.Println()
	fmt.Println("  Your API key:")
	fmt.Println()
	fmt.Printf("    %s\n", output.ColorBold(apiKey))
	fmt.Println()
	fmt.Println(output.ColorGray("  * Add this key to your gateway configuration where OpenClaw expects it:"))
	fmt.Println(output.ColorGray("    skills.entries.porteden.env.PE_API_KEY in ~/.openclaw/openclaw.json"))

	// Step 3: Export (interactive only)
	if auth.IsInteractiveTerminal() {
		fmt.Println()
		output.PrintStep(3, totalSteps, "Additional setup")
		dest := auth.PromptExportDestination(os.Stdin, os.Stdout)
		if dest != auth.ExportNone {
			if err := auth.ExportAPIKey(apiKey, dest); err != nil {
				fmt.Fprintf(os.Stderr, "  Warning: %v\n", err)
			}
		}
	}

	// Completion
	output.PrintCompletion(profileName)
	return apiKey, nil
}

func init() {
	loginCmd.Flags().String("token", "", "API key for direct authentication (non-interactive)")
	loginCmd.Flags().String("title", "", "Title for the API key (e.g., 'Work Laptop')")
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(statusCmd)
	authCmd.AddCommand(listProfilesCmd)
	authCmd.AddCommand(useProfileCmd)
	authCmd.AddCommand(logoutCmd)
}
