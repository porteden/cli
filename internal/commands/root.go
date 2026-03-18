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
	Short:   "PortEden CLI - Calendar, email, and Google Drive from your terminal",
	Version: config.Version,
	Long: `PortEden CLI provides command-line access to calendars, email, and Google Drive.

Authentication:
  porteden auth login                    Authenticate via browser
  porteden auth login --token <key>      Authenticate with API key (non-interactive)
  porteden auth use <profile>            Switch active profile
  porteden auth status                   Check authentication status

Calendar:
  porteden calendar events       List/search events
  porteden calendar create       Create an event
  porteden calendar update       Update an event
  porteden calendar delete       Delete an event
  porteden calendar respond      Respond to invitation
  porteden calendar freebusy     Check free/busy times

Email:
  porteden email messages        List/search emails
  porteden email send            Send a new email
  porteden email reply           Reply to an email
  porteden email forward         Forward an email
  porteden email delete          Delete an email

Drive:
  porteden drive files           List/search files
  porteden drive upload          Upload a file
  porteden drive download        Get file download/export links
  porteden drive share           Share a file
  porteden drive delete          Move file to trash

Docs (Google Docs):
  porteden docs read             Read document content
  porteden docs edit             Edit document (append, insert, replace)
  porteden docs create           Create a new Google Doc

Sheets (Google Sheets):
  porteden sheets read           Read cell values
  porteden sheets write          Write cell values
  porteden sheets append         Append rows
  porteden sheets create         Create a new Google Sheet

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
	rootCmd.AddCommand(driveCmd)
	rootCmd.AddCommand(docsCmd)
	rootCmd.AddCommand(sheetsCmd)
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
