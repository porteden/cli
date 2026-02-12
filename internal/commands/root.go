package commands

import (
	"fmt"
	"os"

	"github.com/porteden/cli/internal/auth"
	"github.com/porteden/cli/internal/config"
	"github.com/porteden/cli/internal/debug"
	"github.com/porteden/cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	outputFormat  string
	profile       string
	colorMode     string
	compactOutput bool
)

var rootCmd = &cobra.Command{
	Use:     "porteden",
	Short:   "PortEden CLI - Calendar and email access from your terminal",
	Version: config.Version,
	Long: `PortEden CLI provides command-line access to your calendars and email.

Authentication:
  porteden auth login                    Authenticate via browser
  porteden auth login --token <key>      Authenticate with API key (non-interactive)
  porteden auth login --profile work     Authenticate to a named profile
  porteden auth use <profile>            Switch active profile
  porteden auth list                     List all profiles
  porteden auth status                   Check authentication status

Calendar:
  porteden calendar events       List/search events
  porteden calendar event        Get a single event
  porteden calendar create       Create an event
  porteden calendar update       Update an event
  porteden calendar delete       Delete an event
  porteden calendar respond      Respond to invitation
  porteden calendar freebusy     Check free/busy times
  porteden calendar by-contact   Events with a contact
  porteden calendar calendars    List calendars

Email:
  porteden email messages        List/search emails
  porteden email message         Get a single email
  porteden email thread          Get an email thread
  porteden email send            Send a new email
  porteden email reply           Reply to an email
  porteden email forward         Forward an email
  porteden email delete          Delete an email
  porteden email modify          Modify email properties

System:
  porteden update                Update to the latest version
  porteden uninstall             Uninstall the CLI`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Apply color settings
		switch colorMode {
		case "never":
			output.SetColorEnabled(false)
		case "always":
			output.SetColorEnabled(true)
			// "auto" uses the detection from init()
		}

		// Skip credential store initialization if PE_API_KEY is set (it takes precedence)
		if os.Getenv("PE_API_KEY") != "" {
			return
		}

		// Initialize credential store
		if err := auth.InitStore(); err != nil {
			// Only fatal if we actually need auth (not for help/version)
			if cmd.Name() != "help" && cmd.Name() != "version" && cmd.Parent() != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
	},
}

func init() {
	rootCmd.SetVersionTemplate("porteden " + config.FullVersion() + "\n")

	rootCmd.PersistentFlags().StringVar(&outputFormat, "format", "", "Output format: json, table, plain")
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "", "Profile name (default: 'default')")
	rootCmd.PersistentFlags().StringVar(&colorMode, "color", "auto", "Color mode: auto, always, never")
	// Bind verbose flag directly to debug.Verbose - single source of truth
	rootCmd.PersistentFlags().BoolVarP(&debug.Verbose, "verbose", "v", false, "Verbose output for debugging")

	rootCmd.PersistentFlags().BoolP("json", "j", false, "Output as JSON")
	rootCmd.PersistentFlags().BoolP("plain", "p", false, "Output as plain text (TSV)")
	rootCmd.PersistentFlags().BoolVarP(&compactOutput, "compact", "c", false, "Compact output for AI agents (filters noise, truncates fields)")

	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(calendarCmd)
	rootCmd.AddCommand(emailCmd)
	rootCmd.AddCommand(selfUpdateCmd)
	rootCmd.AddCommand(uninstallCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Helper function to get the active profile
func getProfile(cmd *cobra.Command) string {
	if profile != "" {
		return profile
	}
	if envProfile := os.Getenv("PE_PROFILE"); envProfile != "" {
		return envProfile
	}
	return "default"
}

// Helper function to get output format
func getOutputFormat(cmd *cobra.Command) output.Format {
	// Check flags first
	if json, _ := cmd.Flags().GetBool("json"); json {
		return output.FormatJSON
	}
	if plain, _ := cmd.Flags().GetBool("plain"); plain {
		return output.FormatPlain
	}
	if outputFormat != "" {
		return output.Format(outputFormat)
	}

	// Check environment variable
	if envFormat := os.Getenv("PE_FORMAT"); envFormat != "" {
		return output.Format(envFormat)
	}

	// Default to table
	return output.FormatTable
}

// IsCompactMode returns true if compact output mode is enabled
func IsCompactMode() bool {
	return compactOutput
}
