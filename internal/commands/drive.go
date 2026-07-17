package commands

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/porteden/cli/internal/api"
	"github.com/porteden/cli/internal/auth"
	"github.com/porteden/cli/internal/output"
	"github.com/spf13/cobra"
)

var driveCmd = &cobra.Command{
	Use:   "drive",
	Short: "Google Drive commands",
	Long: `Manage Google Drive files and folders.

File IDs are always provider-prefixed (e.g., google:1BxiMVs0XRA5...).
Use -jc flags for AI-optimized output.

Examples:
  porteden drive files -jc
  porteden drive files -q "budget report" --all -jc
  porteden drive file google:FILEID -jc
  porteden drive upload --file ./report.pdf --name "Q1 Report.pdf"
  porteden drive share google:FILEID --type user --role reader --email user@example.com`,
}

// ==================== FILES LIST ====================

var driveFilesCmd = &cobra.Command{
	Use:   "files",
	Short: "List/search drive files",
	Long: `List and search Google Drive files.

Examples:
  porteden drive files -jc
  porteden drive files -q "budget" --all -jc
  porteden drive files --folder google:0B7_FOLDER_ID -jc
  porteden drive files --mime-type application/pdf -jc
  porteden drive files --shared-with-me -jc`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		params := buildDriveListParams(cmd)
		fetchAll, _ := cmd.Flags().GetBool("all")

		var response *api.DriveFilesResponse
		if fetchAll {
			response, err = client.GetAllDriveFiles(params)
			if response != nil && response.HasMore {
				fmt.Fprintln(os.Stderr, "Warning: pagination cap reached (50 pages). Results may be incomplete.")
			}
		} else {
			response, err = client.GetDriveFiles(params)
		}
		if err != nil {
			return formatError(err)
		}

		output.PrintWithOptions(response, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

// ==================== FILE METADATA ====================

var driveFileCmd = &cobra.Command{
	Use:   "file <fileId>",
	Short: "Get file metadata",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		result, err := client.GetDriveFile(args[0])
		if err != nil {
			return formatError(err)
		}

		output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

// ==================== READ CONTENT ====================

var driveContentCmd = &cobra.Command{
	Use:   "content <fileId>",
	Short: "Read the text content of a file",
	Long: `Returns the text content of a file in one call.

- Google Docs are exported to text/plain and returned inline.
- text-like files (text/*, JSON, XML, YAML, CSV) are returned as-is.
- Spreadsheets steer to: porteden sheets content <fileId>
- Presentations steer to: porteden slides read <fileId>
- Binary files return readable=false with a webViewLink — open in browser.

The HTTP status is always 200; readability is conveyed via the response.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		result, err := client.GetDriveFileContent(args[0])
		if err != nil {
			return formatError(err)
		}

		output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

// ==================== CREATE (inline content) ====================

var driveCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a file with inline UTF-8 text content",
	Long: `Create a new file in Google Drive with inline UTF-8 text content (no binary stream).

For Workspace target MIME types (document, spreadsheet, presentation) Drive
auto-imports the supplied content. For text/JSON/XML/YAML/CSV the file is
stored as-is. Use ` + "`porteden drive upload`" + ` for binary files.

Examples:
  porteden drive create --name "Notes.md" --mime-type text/markdown --content "# Notes"
  porteden drive create --name "Sprint Plan" --mime-type application/vnd.google-apps.document \
      --content-file ./sprint-plan.md --content-mime-type text/markdown
  porteden drive create --name "Data.csv" --mime-type text/csv --content-file ./data.csv`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		mimeType, _ := cmd.Flags().GetString("mime-type")
		content, _ := cmd.Flags().GetString("content")
		contentFile, _ := cmd.Flags().GetString("content-file")
		contentMime, _ := cmd.Flags().GetString("content-mime-type")
		folder, _ := cmd.Flags().GetString("folder")
		description, _ := cmd.Flags().GetString("description")

		if name == "" {
			return errors.New("--name is required")
		}
		if mimeType == "" {
			return errors.New("--mime-type is required")
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

		result, err := createDriveFileOrBlank(client, name, mimeType, content, contentMime, folder, description)
		if err != nil {
			return formatError(err)
		}

		output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

// buildCreateFileRequest assembles the inline-content create request, leaving
// optional pointer fields nil when the caller omitted them.
func buildCreateFileRequest(name, mimeType, content, contentMime, folder, description string) api.CreateDriveFileWithContentRequest {
	req := api.CreateDriveFileWithContentRequest{
		Name:     name,
		MimeType: mimeType,
		Content:  content,
	}
	if contentMime != "" {
		req.ContentMimeType = &contentMime
	}
	if folder != "" {
		req.FolderID = &folder
	}
	if description != "" {
		req.Description = &description
	}
	return req
}

// createDriveFileOrBlank creates a file via POST /files. The BE's metadata-only
// branch handles empty / whitespace-only content natively — it issues
// `files.create` against Drive with no media import, which materialises a real
// blank Workspace file of the requested target mime. No client-side placeholder
// is needed.
func createDriveFileOrBlank(client *api.Client, name, mimeType, content, contentMime, folder, description string) (*api.DriveOperationResult, error) {
	return client.CreateDriveFile(buildCreateFileRequest(name, mimeType, content, contentMime, folder, description))
}

// ==================== DOWNLOAD LINKS ====================

var driveDownloadCmd = &cobra.Command{
	Use:   "download <fileId>",
	Short: "Get file view/download/export links",
	Long: `Returns view, download, and export URLs for a file.
No binary content is streamed — the response is always JSON containing URLs.

For regular files: returns webViewLink and downloadUrl.
For Google Workspace files (Docs, Sheets, Slides): returns webViewLink and exportLinks.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		return runGetFileLinks(client, args[0], cmd)
	},
}

// ==================== PERMISSIONS ====================

var drivePermissionsCmd = &cobra.Command{
	Use:   "permissions <fileId>",
	Short: "List file sharing permissions",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		return runGetPermissions(client, args[0], cmd)
	},
}

// ==================== UPLOAD ====================

var driveUploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a file to Google Drive",
	Long: `Upload a local file to Google Drive.

Examples:
  porteden drive upload --file ./report.pdf --name "Q1 Report.pdf"
  porteden drive upload --file ./data.csv --name "Data.csv" --folder google:0B7_FOLDER
  porteden drive upload --file ./image.png --mime-type image/png --name "Photo.png"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, _ := cmd.Flags().GetString("file")
		fileName, _ := cmd.Flags().GetString("name")
		mimeType, _ := cmd.Flags().GetString("mime-type")
		folder, _ := cmd.Flags().GetString("folder")
		description, _ := cmd.Flags().GetString("description")

		if filePath == "" {
			return errors.New("--file is required")
		}
		if fileName == "" {
			return errors.New("--name is required")
		}

		fileBytes, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("cannot read file: %w", err)
		}

		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		result, err := client.UploadDriveFile(fileName, mimeType, folder, description, fileBytes)
		if err != nil {
			return formatError(err)
		}

		output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

// ==================== MKDIR ====================

var driveMkdirCmd = &cobra.Command{
	Use:   "mkdir",
	Short: "Create a new folder",
	Long: `Create a new folder in Google Drive.

Examples:
  porteden drive mkdir --name "Project Files"
  porteden drive mkdir --name "Reports" --parent google:0B7_PARENT_FOLDER`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		parent, _ := cmd.Flags().GetString("parent")
		description, _ := cmd.Flags().GetString("description")

		if name == "" {
			return errors.New("--name is required")
		}

		req := api.CreateFolderRequest{Name: name}
		if parent != "" {
			req.ParentFolderID = &parent
		}
		if description != "" {
			req.Description = &description
		}

		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		result, err := client.CreateDriveFolder(req)
		if err != nil {
			return formatError(err)
		}

		output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

// ==================== RENAME ====================

var driveRenameCmd = &cobra.Command{
	Use:   "rename <fileId>",
	Short: "Rename a file or folder",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		return runRenameFile(client, args[0], cmd)
	},
}

// ==================== MOVE ====================

var driveMoveCmd = &cobra.Command{
	Use:   "move <fileId>",
	Short: "Move a file to another folder",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		destination, _ := cmd.Flags().GetString("destination")
		if destination == "" {
			return errors.New("--destination is required (provider-prefixed folder ID, e.g. google:0B7_abc...)")
		}

		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		result, err := client.MoveDriveFile(args[0], api.MoveFileRequest{DestinationFolderID: destination})
		if err != nil {
			return formatError(err)
		}

		output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

// ==================== COPY ====================

var driveCopyCmd = &cobra.Command{
	Use:   "copy <fileId>",
	Short: "Duplicate a file",
	Long: `Creates a copy of a Drive file (Google Docs/Sheets/Slides included) and
returns the new file's id. Folders cannot be copied.

Examples:
  porteden drive copy google:FILEID --name "Working copy"
  porteden drive copy google:FILEID --folder google:0B7_DEST_FOLDER`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		return runCopyFile(client, args[0], cmd)
	},
}

// ==================== DELETE ====================

var driveDeleteCmd = &cobra.Command{
	Use:   "delete <fileId>",
	Short: "Move a file to trash",
	Long:  `Moves the file to Google Drive trash. This is not permanent — it can be restored.`,
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

// ==================== SHARE ====================

var driveShareCmd = &cobra.Command{
	Use:   "share <fileId>",
	Short: "Share a file with a user, group, domain, or anyone",
	Long: `Share a Google Drive file.

Examples:
  porteden drive share google:FILEID --type user --role reader --email user@example.com
  porteden drive share google:FILEID --type domain --role reader --domain example.com
  porteden drive share google:FILEID --type anyone --role reader`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		return runShareFile(client, args[0], cmd)
	},
}

// ==================== SHARED HANDLER FUNCTIONS ====================
// These are called by docs.go and sheets.go wrapper commands too.

func runDeleteFile(client *api.Client, fileID string, yes bool) error {
	if !yes && auth.IsInteractiveTerminal() {
		fmt.Printf("Move file '%s' to trash? [y/N]: ", fileID)
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		choice := strings.TrimSpace(strings.ToLower(line))
		if choice != "y" && choice != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}
	if err := client.DeleteDriveFile(fileID); err != nil {
		return formatError(err)
	}
	fmt.Printf("Moved to trash: %s\n", fileID)
	return nil
}

func runShareFile(client *api.Client, fileID string, cmd *cobra.Command) error {
	shareType, _ := cmd.Flags().GetString("type")
	role, _ := cmd.Flags().GetString("role")
	email, _ := cmd.Flags().GetString("email")
	domain, _ := cmd.Flags().GetString("domain")
	noNotify, _ := cmd.Flags().GetBool("no-notify")
	message, _ := cmd.Flags().GetString("message")

	if shareType == "" {
		return errors.New("--type is required (user, group, domain, anyone)")
	}
	if role == "" {
		return errors.New("--role is required (reader, writer, commenter)")
	}

	req := api.ShareFileRequest{Type: shareType, Role: role}
	if email != "" {
		req.EmailAddress = &email
	}
	if domain != "" {
		req.Domain = &domain
	}
	if noNotify {
		f := false
		req.SendNotification = &f
	}
	if message != "" {
		req.Message = &message
	}

	result, err := client.ShareDriveFile(fileID, req)
	if err != nil {
		return formatError(err)
	}

	output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
		Compact: IsCompactMode(),
	})
	return nil
}

func runGetPermissions(client *api.Client, fileID string, cmd *cobra.Command) error {
	result, err := client.GetDrivePermissions(fileID)
	if err != nil {
		return formatError(err)
	}
	output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
		Compact: IsCompactMode(),
	})
	return nil
}

func runGetFileLinks(client *api.Client, fileID string, cmd *cobra.Command) error {
	result, err := client.GetDriveFileLinks(fileID)
	if err != nil {
		return formatError(err)
	}
	output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
		Compact: IsCompactMode(),
	})
	return nil
}

func runRenameFile(client *api.Client, fileID string, cmd *cobra.Command) error {
	newName, _ := cmd.Flags().GetString("name")
	if newName == "" {
		return errors.New("--name is required")
	}
	result, err := client.RenameDriveFile(fileID, api.RenameFileRequest{NewName: newName})
	if err != nil {
		return formatError(err)
	}
	output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
		Compact: IsCompactMode(),
	})
	return nil
}

// runCopyFile duplicates a Drive file. Shared by drive/sheets/docs/slides copy commands.
func runCopyFile(client *api.Client, fileID string, cmd *cobra.Command) error {
	name, _ := cmd.Flags().GetString("name")
	folder, _ := cmd.Flags().GetString("folder")
	req := api.CopyFileRequest{}
	if name != "" {
		req.Name = &name
	}
	if folder != "" {
		req.ParentFolderID = &folder
	}
	result, err := client.CopyDriveFile(fileID, req)
	if err != nil {
		return formatError(err)
	}
	output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
		Compact: IsCompactMode(),
	})
	return nil
}

// addCopyFlags registers the copy flags on a command (used by drive, sheets, docs, slides copy commands)
func addCopyFlags(cmd *cobra.Command) {
	cmd.Flags().String("name", "", `Name for the copy (omit for Google's "Copy of …" default)`)
	cmd.Flags().String("folder", "", "Destination folder ID (provider-prefixed). Omit to copy into the source's folder.")
}

// addShareFlags registers the share flags on a command (used by drive, docs, sheets share commands)
func addShareFlags(cmd *cobra.Command) {
	cmd.Flags().String("type", "", "Share type: user, group, domain, anyone")
	cmd.Flags().String("role", "", "Permission role: reader, writer, commenter")
	cmd.Flags().String("email", "", "Email address (required for user/group types)")
	cmd.Flags().String("domain", "", "Domain to share with (required for domain type)")
	cmd.Flags().Bool("no-notify", false, "Skip email notification to recipient")
	cmd.Flags().String("message", "", "Custom notification message")
}

// buildDriveListParams builds DriveListParams from command flags
func buildDriveListParams(cmd *cobra.Command) api.DriveListParams {
	params := api.DriveListParams{Limit: 25}

	if q, _ := cmd.Flags().GetString("query"); q != "" {
		params.Q = q
	}
	if folder, _ := cmd.Flags().GetString("folder"); folder != "" {
		params.FolderID = folder
	}
	if mime, _ := cmd.Flags().GetString("mime-type"); mime != "" {
		params.MimeType = mime
	}
	if name, _ := cmd.Flags().GetString("name"); name != "" {
		params.Name = name
	}
	if trashed, _ := cmd.Flags().GetBool("trashed"); trashed {
		params.TrashedOnly = true
	}
	if shared, _ := cmd.Flags().GetBool("shared-with-me"); shared {
		params.SharedWithMe = true
	}
	if after, _ := cmd.Flags().GetString("modified-after"); after != "" {
		params.ModifiedAfter = after
	}
	if before, _ := cmd.Flags().GetString("modified-before"); before != "" {
		params.ModifiedBefore = before
	}
	if limit, _ := cmd.Flags().GetInt("limit"); limit > 0 {
		params.Limit = limit
	}
	if pageToken, _ := cmd.Flags().GetString("page-token"); pageToken != "" {
		params.PageToken = pageToken
	}
	if orderBy, _ := cmd.Flags().GetString("order-by"); orderBy != "" {
		params.OrderBy = orderBy
	}

	return params
}

func init() {
	// files flags
	driveFilesCmd.Flags().StringP("query", "q", "", "Free-text search query")
	driveFilesCmd.Flags().String("folder", "", "Restrict to files in folder (provider-prefixed ID)")
	driveFilesCmd.Flags().String("mime-type", "", "Filter by MIME type (e.g., application/pdf)")
	driveFilesCmd.Flags().String("name", "", "Filter by file name (partial match)")
	driveFilesCmd.Flags().Bool("trashed", false, "Only return trashed files")
	driveFilesCmd.Flags().Bool("shared-with-me", false, "Only files shared with the connected user")
	driveFilesCmd.Flags().String("modified-after", "", "Files modified after date (ISO 8601)")
	driveFilesCmd.Flags().String("modified-before", "", "Files modified before date (ISO 8601)")
	driveFilesCmd.Flags().Int("limit", 25, "Results per page (1-100)")
	driveFilesCmd.Flags().Bool("all", false, "Auto-paginate to fetch all results")
	driveFilesCmd.Flags().String("page-token", "", "Opaque cursor from a previous response's nextPageToken")
	driveFilesCmd.Flags().String("order-by", "modified_time", "Sort field: name, modified_time, created_time, size")

	// upload flags
	driveUploadCmd.Flags().String("file", "", "Local file path to upload")
	driveUploadCmd.Flags().String("name", "", "File name on Google Drive")
	driveUploadCmd.Flags().String("mime-type", "", "MIME type (auto-detected if omitted)")
	driveUploadCmd.Flags().String("folder", "", "Target folder ID (provider-prefixed)")
	driveUploadCmd.Flags().String("description", "", "File description")

	// create (inline content) flags
	driveCreateCmd.Flags().String("name", "", "File name to assign in Drive")
	driveCreateCmd.Flags().String("mime-type", "", "Target MIME type (e.g., application/vnd.google-apps.document, text/markdown)")
	driveCreateCmd.Flags().String("content", "", "Inline UTF-8 content")
	driveCreateCmd.Flags().String("content-file", "", "Path to a UTF-8 text file with content")
	driveCreateCmd.Flags().String("content-mime-type", "", "MIME type of supplied content when it differs from --mime-type (e.g., text/markdown)")
	driveCreateCmd.Flags().String("folder", "", "Target folder ID (provider-prefixed). Omit for root.")
	driveCreateCmd.Flags().String("description", "", "File description")

	// mkdir flags
	driveMkdirCmd.Flags().String("name", "", "Folder name")
	driveMkdirCmd.Flags().String("parent", "", "Parent folder ID (provider-prefixed). Omit for root.")
	driveMkdirCmd.Flags().String("description", "", "Folder description")

	// rename flags
	driveRenameCmd.Flags().String("name", "", "New name for the file or folder")

	// move flags
	driveMoveCmd.Flags().String("destination", "", "Destination folder ID (provider-prefixed)")

	// delete flags
	driveDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	// share flags
	addShareFlags(driveShareCmd)

	// copy flags
	addCopyFlags(driveCopyCmd)

	// Register sub-commands
	driveCmd.AddCommand(driveFilesCmd)
	driveCmd.AddCommand(driveFileCmd)
	driveCmd.AddCommand(driveContentCmd)
	driveCmd.AddCommand(driveCreateCmd)
	driveCmd.AddCommand(driveDownloadCmd)
	driveCmd.AddCommand(drivePermissionsCmd)
	driveCmd.AddCommand(driveUploadCmd)
	driveCmd.AddCommand(driveMkdirCmd)
	driveCmd.AddCommand(driveRenameCmd)
	driveCmd.AddCommand(driveMoveCmd)
	driveCmd.AddCommand(driveCopyCmd)
	driveCmd.AddCommand(driveDeleteCmd)
	driveCmd.AddCommand(driveShareCmd)
}
