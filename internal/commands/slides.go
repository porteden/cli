package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/porteden/cli/internal/output"
	"github.com/spf13/cobra"
)

var slidesCmd = &cobra.Command{
	Use:   "slides",
	Short: "Google Slides commands",
	Long: `Read and manage Google Slides presentations.

File IDs are always provider-prefixed (e.g., google:1BxiMVs0XRA5...).
Use -jc flags for AI-optimized output.

Examples:
  porteden slides info google:DECKID -jc
  porteden slides read google:DECKID
  porteden slides read google:DECKID --format structured -j
  porteden slides create --name "Q1 Review"
  porteden slides share google:DECKID --type user --role reader --email user@example.com`,
}

// ==================== INFO ====================

var slidesInfoCmd = &cobra.Command{
	Use:   "info <fileId>",
	Short: "Get presentation metadata (slide index + first-line titles)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		result, err := client.GetSlidesMetadata(args[0])
		if err != nil {
			return formatError(err)
		}

		output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

// ==================== READ ====================

var slidesReadCmd = &cobra.Command{
	Use:   "read <fileId>",
	Short: "Read presentation content (slide text + speaker notes)",
	Long: `Read deck content. Default --format text returns slide bodies joined
with "---" separators, with speaker notes appended under each slide as
[Speaker notes]. --format structured returns the raw Google Slides API JSON.

Examples:
  porteden slides read google:DECKID
  porteden slides read google:DECKID --format structured -j`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")

		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		result, err := client.GetSlidesContent(args[0], format)
		if err != nil {
			return formatError(err)
		}

		output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

// ==================== CREATE ====================

var slidesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new Google Slides deck, optionally seeded with text",
	Long: `Creates a new Google Slides presentation. Pass --content (inline) or
--content-file to seed the first slide(s) from a text outline.

Examples:
  porteden slides create --name "Q1 Review"
  porteden slides create --name "Kickoff" --content-file ./outline.txt`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		folder, _ := cmd.Flags().GetString("folder")
		content, _ := cmd.Flags().GetString("content")
		contentFile, _ := cmd.Flags().GetString("content-file")
		contentMime, _ := cmd.Flags().GetString("content-mime-type")

		if name == "" {
			return errors.New("--name is required")
		}
		if content != "" && contentFile != "" {
			return errors.New("--content and --content-file are mutually exclusive")
		}
		if contentFile != "" {
			data, err := os.ReadFile(contentFile)
			if err != nil {
				return fmt.Errorf("cannot read content file: %w", err)
			}
			content = string(data)
		}

		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		result, err := createDriveFileOrBlank(client, name, "application/vnd.google-apps.presentation", content, contentMime, folder, "")
		if err != nil {
			return formatError(err)
		}

		output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

// ==================== FILE MANAGEMENT WRAPPERS ====================

var slidesRenameCmd = &cobra.Command{
	Use:   "rename <fileId>",
	Short: "Rename a Google Slides deck",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		return runRenameFile(client, args[0], cmd)
	},
}

var slidesDeleteCmd = &cobra.Command{
	Use:   "delete <fileId>",
	Short: "Move a Google Slides deck to trash",
	Long:  `Moves the presentation to Google Drive trash. This is not permanent — it can be restored.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		yes, _ := cmd.Flags().GetBool("yes")
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		return runDeleteFile(client, args[0], yes)
	},
}

var slidesShareCmd = &cobra.Command{
	Use:   "share <fileId>",
	Short: "Share a Google Slides deck",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		return runShareFile(client, args[0], cmd)
	},
}

var slidesPermissionsCmd = &cobra.Command{
	Use:   "permissions <fileId>",
	Short: "List sharing permissions for a Google Slides deck",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		return runGetPermissions(client, args[0], cmd)
	},
}

var slidesDownloadCmd = &cobra.Command{
	Use:   "download <fileId>",
	Short: "Get export links for a Google Slides deck",
	Long: `Returns export URLs for downloading the presentation (pptx, pdf, txt).
No binary content is streamed — the response contains URLs only.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		return runGetFileLinks(client, args[0], cmd)
	},
}

func init() {
	// read flags
	slidesReadCmd.Flags().String("format", "text", "Content format: text (default) or structured")

	// create flags
	slidesCreateCmd.Flags().String("name", "", "Presentation name")
	slidesCreateCmd.Flags().String("folder", "", "Target folder ID (provider-prefixed). Omit for root.")
	slidesCreateCmd.Flags().String("content", "", "Inline UTF-8 content to seed the deck")
	slidesCreateCmd.Flags().String("content-file", "", "Path to a UTF-8 text file to seed the deck")
	slidesCreateCmd.Flags().String("content-mime-type", "", "MIME of supplied content (e.g., text/markdown). Defaults to text/plain.")

	// rename flags
	slidesRenameCmd.Flags().String("name", "", "New presentation name")

	// delete flags
	slidesDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	// share flags
	addShareFlags(slidesShareCmd)

	// Register sub-commands
	slidesCmd.AddCommand(slidesInfoCmd)
	slidesCmd.AddCommand(slidesReadCmd)
	slidesCmd.AddCommand(slidesCreateCmd)
	slidesCmd.AddCommand(slidesRenameCmd)
	slidesCmd.AddCommand(slidesDeleteCmd)
	slidesCmd.AddCommand(slidesShareCmd)
	slidesCmd.AddCommand(slidesPermissionsCmd)
	slidesCmd.AddCommand(slidesDownloadCmd)
}
