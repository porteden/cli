package commands

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/porteden/cli/internal/api"
	"github.com/porteden/cli/internal/output"
	"github.com/spf13/cobra"
)

var sheetsCmd = &cobra.Command{
	Use:   "sheets",
	Short: "Google Sheets commands",
	Long: `Read and write Google Sheets spreadsheets.

File IDs are always provider-prefixed (e.g., google:1BxiMVs0XRA5...).
Use -jc flags for AI-optimized output.

Examples:
  porteden sheets info google:SHEETID -jc
  porteden sheets read google:SHEETID --range "Sheet1!A1:C10" -jc
  porteden sheets write google:SHEETID --range "Sheet1!A1:B2" --values '[["Name","Score"],["Alice",95]]'
  porteden sheets append google:SHEETID --range "Sheet1!A:B" --csv "Bob,87"`,
}

// ==================== CREATE ====================

var sheetsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new Google Sheet",
	Long: `Creates a new blank Google Spreadsheet via the Drive upload endpoint.

Examples:
  porteden sheets create --name "Q1 Budget"
  porteden sheets create --name "Data" --folder google:0B7_FOLDER`,
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

		result, err := client.UploadDriveFile(name, "application/vnd.google-apps.spreadsheet", folder, "", []byte{})
		if err != nil {
			return formatError(err)
		}

		output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

// ==================== INFO ====================

var sheetsInfoCmd = &cobra.Command{
	Use:   "info <fileId>",
	Short: "Get spreadsheet metadata",
	Long: `Returns the spreadsheet title and a list of sheet tabs with their dimensions.

Examples:
  porteden sheets info google:SHEETID -jc`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		result, err := client.GetSheetMetadata(args[0])
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

var sheetsReadCmd = &cobra.Command{
	Use:   "read <fileId>",
	Short: "Read cell values from a range",
	Long: `Read cell values from a Google Sheet range.

Range format: Sheet1!A1:C10 or Sheet1 or A1:B5
Use -j for JSON output suitable for processing with jq or LLM agents.

Examples:
  porteden sheets read google:SHEETID --range "Sheet1!A1:C10" -jc
  porteden sheets read google:SHEETID --range "Sheet1" -j`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rangeStr, _ := cmd.Flags().GetString("range")
		if rangeStr == "" {
			return errors.New("--range is required (e.g., Sheet1!A1:C10 or Sheet1)")
		}

		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		result, err := client.ReadSheetValues(args[0], rangeStr)
		if err != nil {
			return formatError(err)
		}

		output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

// ==================== WRITE ====================

var sheetsWriteCmd = &cobra.Command{
	Use:   "write <fileId>",
	Short: "Write cell values to a range",
	Long: `Write cell values to a Google Sheet range, overwriting existing content.

Provide values in one of three formats:
  --values '[["Name","Score"],["Alice",95]]'   JSON 2D array
  --csv "Name,Score\nAlice,95"                 CSV string (\\n = newline)
  --csv-file ./data.csv                        CSV file path

Use --raw to send literal values without formula evaluation.

Examples:
  porteden sheets write google:SHEETID --range "Sheet1!A1:B2" --values '[["Name","Score"],["Alice",95]]'
  porteden sheets write google:SHEETID --range "Sheet1!A1:B2" --csv "Name,Score\nAlice,95"
  porteden sheets write google:SHEETID --range "Sheet1!A1" --csv-file ./data.csv`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rangeStr, _ := cmd.Flags().GetString("range")
		if rangeStr == "" {
			return errors.New("--range is required (e.g., Sheet1!A1:C3)")
		}

		values, err := parseSheetValues(cmd)
		if err != nil {
			return err
		}

		rawMode, _ := cmd.Flags().GetBool("raw")
		inputOption := "USER_ENTERED"
		if rawMode {
			inputOption = "RAW"
		}

		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		result, err := client.WriteSheetValues(args[0], api.WriteSheetValuesRequest{
			Range:            rangeStr,
			Values:           values,
			ValueInputOption: inputOption,
		})
		if err != nil {
			return formatError(err)
		}

		output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

// ==================== APPEND ====================

var sheetsAppendCmd = &cobra.Command{
	Use:   "append <fileId>",
	Short: "Append rows after the last data row",
	Long: `Appends rows after the last row with data in the specified range.
Existing data is never overwritten.

Provide values in one of three formats:
  --values '[["Alice","alice@x.com"]]'   JSON 2D array
  --csv "Alice,alice@x.com"              CSV string
  --csv-file ./rows.csv                  CSV file path

Examples:
  porteden sheets append google:SHEETID --range "Sheet1!A:B" --csv "Alice,95"
  porteden sheets append google:SHEETID --range "Sheet1" --values '[["Bob",87]]'`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rangeStr, _ := cmd.Flags().GetString("range")
		if rangeStr == "" {
			return errors.New("--range is required (e.g., Sheet1!A:C or Sheet1)")
		}

		values, err := parseSheetValues(cmd)
		if err != nil {
			return err
		}

		rawMode, _ := cmd.Flags().GetBool("raw")
		inputOption := "USER_ENTERED"
		if rawMode {
			inputOption = "RAW"
		}

		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		result, err := client.AppendSheetRows(args[0], api.AppendSheetRowsRequest{
			Range:            rangeStr,
			Values:           values,
			ValueInputOption: inputOption,
		})
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

var sheetsRenameCmd = &cobra.Command{
	Use:   "rename <fileId>",
	Short: "Rename a Google Sheet",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		return runRenameFile(client, args[0], cmd)
	},
}

var sheetsDeleteCmd = &cobra.Command{
	Use:   "delete <fileId>",
	Short: "Move a Google Sheet to trash",
	Long:  `Moves the spreadsheet to Google Drive trash. This is not permanent — it can be restored.`,
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

var sheetsShareCmd = &cobra.Command{
	Use:   "share <fileId>",
	Short: "Share a Google Sheet",
	Long: `Share a Google Sheet with a user, group, domain, or anyone.

Examples:
  porteden sheets share google:SHEETID --type user --role writer --email user@example.com
  porteden sheets share google:SHEETID --type anyone --role reader`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		return runShareFile(client, args[0], cmd)
	},
}

var sheetsPermissionsCmd = &cobra.Command{
	Use:   "permissions <fileId>",
	Short: "List sharing permissions for a Google Sheet",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		return runGetPermissions(client, args[0], cmd)
	},
}

var sheetsDownloadCmd = &cobra.Command{
	Use:   "download <fileId>",
	Short: "Get export links for a Google Sheet",
	Long: `Returns export URLs for downloading the spreadsheet (xlsx, pdf, csv).
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

// ==================== HELPERS ====================

// parseSheetValues extracts a 2D [][]interface{} from --values, --csv, or --csv-file flags.
// Exactly one of the three must be provided.
func parseSheetValues(cmd *cobra.Command) ([][]interface{}, error) {
	valuesJSON, _ := cmd.Flags().GetString("values")
	csvStr, _ := cmd.Flags().GetString("csv")
	csvFile, _ := cmd.Flags().GetString("csv-file")

	provided := 0
	if valuesJSON != "" {
		provided++
	}
	if csvStr != "" {
		provided++
	}
	if csvFile != "" {
		provided++
	}

	if provided == 0 {
		return nil, errors.New("provide values via --values (JSON), --csv (string), or --csv-file (path)")
	}
	if provided > 1 {
		return nil, errors.New("--values, --csv, and --csv-file are mutually exclusive")
	}

	if valuesJSON != "" {
		var raw [][]interface{}
		if err := json.Unmarshal([]byte(valuesJSON), &raw); err != nil {
			return nil, fmt.Errorf("invalid --values JSON: %w", err)
		}
		return raw, nil
	}

	// CSV path
	var csvContent string
	if csvFile != "" {
		data, err := os.ReadFile(csvFile)
		if err != nil {
			return nil, fmt.Errorf("cannot read csv file: %w", err)
		}
		csvContent = string(data)
	} else {
		// Replace literal \n (two chars) with real newline for inline CSV convenience
		csvContent = strings.ReplaceAll(csvStr, `\n`, "\n")
	}

	r := csv.NewReader(strings.NewReader(csvContent))
	r.FieldsPerRecord = -1 // allow variable columns
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("invalid CSV: %w", err)
	}

	result := make([][]interface{}, len(records))
	for i, row := range records {
		result[i] = make([]interface{}, len(row))
		for j, cell := range row {
			result[i][j] = cell
		}
	}
	return result, nil
}

func init() {
	// create flags
	sheetsCreateCmd.Flags().String("name", "", "Spreadsheet name")
	sheetsCreateCmd.Flags().String("folder", "", "Target folder ID (provider-prefixed). Omit for root.")

	// read flags
	sheetsReadCmd.Flags().String("range", "", `Cell range in A1 notation (e.g., Sheet1!A1:C10 or Sheet1)`)

	// write flags
	sheetsWriteCmd.Flags().String("range", "", `Target range in A1 notation (e.g., Sheet1!A1:C3)`)
	sheetsWriteCmd.Flags().String("values", "", `Values as JSON 2D array (e.g., '[["Name","Score"],["Alice",95]]')`)
	sheetsWriteCmd.Flags().String("csv", "", `Values as CSV string (use \\n for row separator)`)
	sheetsWriteCmd.Flags().String("csv-file", "", "Path to a CSV file")
	sheetsWriteCmd.Flags().Bool("raw", false, "Use RAW input mode (disables formula evaluation)")

	// append flags
	sheetsAppendCmd.Flags().String("range", "", `Range to detect the table (e.g., Sheet1!A:C or Sheet1)`)
	sheetsAppendCmd.Flags().String("values", "", `Rows as JSON 2D array (e.g., '[["Alice","alice@x.com"]]')`)
	sheetsAppendCmd.Flags().String("csv", "", `Rows as CSV string (use \\n for multiple rows)`)
	sheetsAppendCmd.Flags().String("csv-file", "", "Path to a CSV file")
	sheetsAppendCmd.Flags().Bool("raw", false, "Use RAW input mode (disables formula evaluation)")

	// rename flags
	sheetsRenameCmd.Flags().String("name", "", "New spreadsheet name")

	// delete flags
	sheetsDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	// share flags
	addShareFlags(sheetsShareCmd)

	// Register sub-commands
	sheetsCmd.AddCommand(sheetsCreateCmd)
	sheetsCmd.AddCommand(sheetsInfoCmd)
	sheetsCmd.AddCommand(sheetsReadCmd)
	sheetsCmd.AddCommand(sheetsWriteCmd)
	sheetsCmd.AddCommand(sheetsAppendCmd)
	sheetsCmd.AddCommand(sheetsRenameCmd)
	sheetsCmd.AddCommand(sheetsDeleteCmd)
	sheetsCmd.AddCommand(sheetsShareCmd)
	sheetsCmd.AddCommand(sheetsPermissionsCmd)
	sheetsCmd.AddCommand(sheetsDownloadCmd)
}
