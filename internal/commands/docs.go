package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/porteden/cli/internal/api"
	"github.com/porteden/cli/internal/output"
	"github.com/spf13/cobra"
)

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Google Docs commands",
	Long: `Create, read, and edit Google Docs.

File IDs are always provider-prefixed (e.g., google:1BxiMVs0XRA5...).
Use -jc flags for AI-optimized output.

Examples:
  porteden docs create --name "My Document"
  porteden docs read google:DOCID
  porteden docs edit google:DOCID --append "New content"
  porteden docs edit google:DOCID --find "old text" --replace "new text"`,
}

// ==================== CREATE ====================

var docsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new Google Doc",
	Long: `Creates a new blank Google Doc via the Drive upload endpoint.

Examples:
  porteden docs create --name "Meeting Notes"
  porteden docs create --name "Project Brief" --folder google:0B7_FOLDER`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		folder, _ := cmd.Flags().GetString("folder")

		if name == "" {
			return errors.New("--name is required")
		}

		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		result, err := client.UploadDriveFile(name, "application/vnd.google-apps.document", folder, "", []byte{})
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

var docsReadCmd = &cobra.Command{
	Use:   "read <fileId>",
	Short: "Read document content",
	Long: `Read the text content of a Google Doc.

By default returns plain text. Use --format structured to get the full
Google Docs API JSON representation (headings, formatting, etc.).

Examples:
  porteden docs read google:DOCID
  porteden docs read google:DOCID --format structured -j`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")

		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		result, err := client.GetDocContent(args[0], format)
		if err != nil {
			return formatError(err)
		}

		output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

// ==================== EDIT ====================

var docsEditCmd = &cobra.Command{
	Use:   "edit <fileId>",
	Short: "Edit document content",
	Long: `Apply text editing operations to a Google Doc.

Operation types:
  --append "text"                 Append text at the end of the document
  --insert "text" --at <index>    Insert text at character position (1 = start)
  --find "text" --replace "text"  Find and replace all occurrences (repeatable)
  --ops-file path/to/ops.json     Apply operations from a JSON file (mutually exclusive with above)

Multiple find/replace pairs can be specified by repeating the flags:
  porteden docs edit google:DOCID --find "foo" --replace "bar" --find "baz" --replace "qux"

Ops file format:
  [{"type":"appendText","text":"Hello"},{"type":"replaceText","find":"old","replace":"new"}]`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		opsFile, _ := cmd.Flags().GetString("ops-file")
		appendText, _ := cmd.Flags().GetString("append")
		insertText, _ := cmd.Flags().GetString("insert")
		insertAt, _ := cmd.Flags().GetInt("at")
		findTexts, _ := cmd.Flags().GetStringArray("find")
		replaceTexts, _ := cmd.Flags().GetStringArray("replace")

		var ops []api.DocEditOperation

		if opsFile != "" {
			// Mutually exclusive with inline flags
			if appendText != "" || insertText != "" || len(findTexts) > 0 {
				return errors.New("--ops-file cannot be combined with --append, --insert, or --find/--replace")
			}
			data, err := os.ReadFile(opsFile)
			if err != nil {
				return fmt.Errorf("cannot read ops file: %w", err)
			}
			if err := json.Unmarshal(data, &ops); err != nil {
				return fmt.Errorf("invalid ops file JSON: %w", err)
			}
			if len(ops) == 0 {
				return errors.New("ops file contains no operations")
			}
		} else {
			// Validate find/replace parity
			if len(findTexts) != len(replaceTexts) {
				return fmt.Errorf("--find and --replace must be used in pairs (%d find vs %d replace)", len(findTexts), len(replaceTexts))
			}

			if appendText == "" && insertText == "" && len(findTexts) == 0 {
				return errors.New("specify at least one operation: --append, --insert, --find/--replace, or --ops-file")
			}

			if appendText != "" {
				ops = append(ops, api.DocEditOperation{
					Type: "appendText",
					Text: &appendText,
				})
			}
			if insertText != "" {
				idx := insertAt
				ops = append(ops, api.DocEditOperation{
					Type:  "insertText",
					Text:  &insertText,
					Index: &idx,
				})
			}
			for i, findText := range findTexts {
				ft := findText
				rt := replaceTexts[i]
				tr := true
				ops = append(ops, api.DocEditOperation{
					Type:      "replaceText",
					Find:      &ft,
					Replace:   &rt,
					MatchCase: &tr,
				})
			}
		}

		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		result, err := client.EditDoc(args[0], api.EditDocRequest{Operations: ops})
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

var docsRenameCmd = &cobra.Command{
	Use:   "rename <fileId>",
	Short: "Rename a Google Doc",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		return runRenameFile(client, args[0], cmd)
	},
}

var docsDeleteCmd = &cobra.Command{
	Use:   "delete <fileId>",
	Short: "Move a Google Doc to trash",
	Long:  `Moves the document to Google Drive trash. This is not permanent — it can be restored.`,
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

var docsShareCmd = &cobra.Command{
	Use:   "share <fileId>",
	Short: "Share a Google Doc",
	Long: `Share a Google Doc with a user, group, domain, or anyone.

Examples:
  porteden docs share google:DOCID --type user --role writer --email user@example.com
  porteden docs share google:DOCID --type anyone --role reader`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		return runShareFile(client, args[0], cmd)
	},
}

var docsPermissionsCmd = &cobra.Command{
	Use:   "permissions <fileId>",
	Short: "List sharing permissions for a Google Doc",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		return runGetPermissions(client, args[0], cmd)
	},
}

var docsDownloadCmd = &cobra.Command{
	Use:   "download <fileId>",
	Short: "Get export links for a Google Doc",
	Long: `Returns export URLs for downloading the document in various formats (pdf, docx, txt).
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
	// create flags
	docsCreateCmd.Flags().String("name", "", "Document name")
	docsCreateCmd.Flags().String("folder", "", "Target folder ID (provider-prefixed). Omit for root.")

	// read flags
	docsReadCmd.Flags().String("format", "text", "Content format: text (default) or structured")

	// edit flags
	docsEditCmd.Flags().String("append", "", "Text to append at the end of the document")
	docsEditCmd.Flags().String("insert", "", "Text to insert at a position")
	docsEditCmd.Flags().Int("at", 1, "Character index for --insert (1 = start of document body)")
	docsEditCmd.Flags().StringArray("find", nil, "Text to find (repeatable, paired with --replace)")
	docsEditCmd.Flags().StringArray("replace", nil, "Replacement text (repeatable, paired with --find)")
	docsEditCmd.Flags().String("ops-file", "", "Path to JSON file with operations array")

	// rename flags
	docsRenameCmd.Flags().String("name", "", "New document name")

	// delete flags
	docsDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	// share flags
	addShareFlags(docsShareCmd)

	// Register sub-commands
	docsCmd.AddCommand(docsCreateCmd)
	docsCmd.AddCommand(docsReadCmd)
	docsCmd.AddCommand(docsEditCmd)
	docsCmd.AddCommand(docsRenameCmd)
	docsCmd.AddCommand(docsDeleteCmd)
	docsCmd.AddCommand(docsShareCmd)
	docsCmd.AddCommand(docsPermissionsCmd)
	docsCmd.AddCommand(docsDownloadCmd)
}
