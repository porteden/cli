package commands

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/porteden/cli/internal/api"
	"github.com/porteden/cli/internal/auth"
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
	Short: "Create a new Google Sheet, optionally seeded with CSV content",
	Long: `Creates a new Google Spreadsheet. Pass --csv-file (or --csv) to seed
the first tab from a CSV in a single round-trip; the server imports CSV into
the Workspace sheet format automatically.

Examples:
  porteden sheets create --name "Q1 Budget"
  porteden sheets create --name "Sales 2026" --csv-file ./sales.csv
  porteden sheets create --name "Data" --folder google:0B7_FOLDER`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		folder, _ := cmd.Flags().GetString("folder")
		csv, _ := cmd.Flags().GetString("csv")
		csvFile, _ := cmd.Flags().GetString("csv-file")

		if name == "" {
			return errors.New("--name is required")
		}
		if csv != "" && csvFile != "" {
			return errors.New("--csv and --csv-file are mutually exclusive")
		}

		content := csv
		if csvFile != "" {
			data, err := os.ReadFile(csvFile)
			if err != nil {
				return fmt.Errorf("cannot read csv file: %w", err)
			}
			content = string(data)
		} else if csv != "" {
			content = strings.ReplaceAll(csv, `\n`, "\n")
		}

		contentMime := ""
		if content != "" {
			contentMime = "text/csv"
		}

		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		result, err := createDriveFileOrBlank(client, name, "application/vnd.google-apps.spreadsheet", content, contentMime, folder, "")
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

Or write multiple ranges atomically in one call with --updates (a JSON array of
{"range","values"} objects; up to 50 ranges / 50,000 cells). --updates is
mutually exclusive with --range/--values/--csv/--csv-file.

Use --raw to send literal values without formula evaluation.

Examples:
  porteden sheets write google:SHEETID --range "Sheet1!A1:B2" --values '[["Name","Score"],["Alice",95]]'
  porteden sheets write google:SHEETID --range "Sheet1!A1:B2" --csv "Name,Score\nAlice,95"
  porteden sheets write google:SHEETID --range "Sheet1!A1" --csv-file ./data.csv
  porteden sheets write google:SHEETID --updates '[{"range":"Summary!A1:B1","values":[["Metric","Value"]]},{"range":"Detail!A1","values":[["x"]]}]'`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rawMode, _ := cmd.Flags().GetBool("raw")
		inputOption := "USER_ENTERED"
		if rawMode {
			inputOption = "RAW"
		}

		req, err := buildWriteRequest(cmd, inputOption)
		if err != nil {
			return err
		}

		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		result, err := client.WriteSheetValues(args[0], req)
		if err != nil {
			return formatError(err)
		}

		output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

// ==================== BULK CONTENT (ALL TABS) ====================

var sheetsContentCmd = &cobra.Command{
	Use:   "content <fileId>",
	Short: "Read all tabs in one call (or supply explicit ranges)",
	Long: `Reads every tab of the spreadsheet in a single upstream call.
By default each tab is clipped to 200 rows × 26 columns; clipped tabs
include a fullRange string you can feed straight into ` + "`porteden sheets read --range`" + `.

Use --ranges to bypass the default and pass explicit A1 ranges through verbatim
(no per-tab cap applies).

Examples:
  porteden sheets content google:SHEETID -jc
  porteden sheets content google:SHEETID --max-rows 500
  porteden sheets content google:SHEETID --ranges "Summary!A1:C100,Raw Data!A:Z"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rangesStr, _ := cmd.Flags().GetString("ranges")
		maxRows, _ := cmd.Flags().GetInt("max-rows")

		var ranges []string
		if rangesStr != "" {
			for _, r := range strings.Split(rangesStr, ",") {
				r = strings.TrimSpace(r)
				if r != "" {
					ranges = append(ranges, r)
				}
			}
		}

		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		result, err := client.GetSheetBulkContent(args[0], ranges, maxRows)
		if err != nil {
			return formatError(err)
		}

		output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

// ==================== TAB MANAGEMENT ====================

var sheetsReadTabCmd = &cobra.Command{
	Use:   "read-tab <fileId>",
	Short: "Read the entire used range of one tab",
	Long: `Reads a single tab's whole used range in one call — no A1 range to compute.
Identify the tab by --title (primary) or --sheet-id. Provide exactly one.

Examples:
  porteden sheets read-tab google:SHEETID --title "Q2 Forecast" -jc
  porteden sheets read-tab google:SHEETID --sheet-id 0 -jc`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		title, _ := cmd.Flags().GetString("title")
		hasSheetID := cmd.Flags().Changed("sheet-id")

		if hasSheetID == (title != "") {
			return errors.New("provide exactly one of --title or --sheet-id")
		}

		var sheetID *int
		if hasSheetID {
			id, _ := cmd.Flags().GetInt("sheet-id")
			sheetID = &id
		}

		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		result, err := client.ReadSheetTab(args[0], title, sheetID)
		if err != nil {
			return formatError(err)
		}

		output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

var sheetsAddTabCmd = &cobra.Command{
	Use:   "add-tab <fileId>",
	Short: "Add a new tab (worksheet) to a spreadsheet",
	Long: `Adds a new tab and returns its assigned sheetId. The title must be unique
within the spreadsheet. --rows/--cols/--index are optional (Google defaults apply).

Examples:
  porteden sheets add-tab google:SHEETID --title "Q2 Forecast"
  porteden sheets add-tab google:SHEETID --title "Q2 Forecast" --rows 200 --cols 12 --index 2`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		title, _ := cmd.Flags().GetString("title")
		if title == "" {
			return errors.New("--title is required")
		}

		req := api.AddSheetTabRequest{Title: title}
		if cmd.Flags().Changed("rows") {
			n, _ := cmd.Flags().GetInt("rows")
			req.RowCount = &n
		}
		if cmd.Flags().Changed("cols") {
			n, _ := cmd.Flags().GetInt("cols")
			req.ColumnCount = &n
		}
		if cmd.Flags().Changed("index") {
			n, _ := cmd.Flags().GetInt("index")
			req.Index = &n
		}

		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		result, err := client.AddSheetTab(args[0], req)
		if err != nil {
			return formatError(err)
		}

		output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

var sheetsDeleteTabCmd = &cobra.Command{
	Use:   "delete-tab <fileId>",
	Short: "Delete one tab from a spreadsheet",
	Long: `Deletes one tab, identified by --sheet-id (primary) or --title. Provide
exactly one. A spreadsheet must retain at least one tab.

Examples:
  porteden sheets delete-tab google:SHEETID --title "Q2 Forecast" -y
  porteden sheets delete-tab google:SHEETID --sheet-id 1843502915 -y`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		title, _ := cmd.Flags().GetString("title")
		hasSheetID := cmd.Flags().Changed("sheet-id")

		if hasSheetID == (title != "") {
			return errors.New("provide exactly one of --title or --sheet-id")
		}

		var sheetID *int
		target := title
		if hasSheetID {
			id, _ := cmd.Flags().GetInt("sheet-id")
			sheetID = &id
			target = fmt.Sprintf("sheetId %d", id)
		}

		yes, _ := cmd.Flags().GetBool("yes")
		if !yes && auth.IsInteractiveTerminal() {
			fmt.Printf("Delete tab %q from %s? [y/N]: ", target, args[0])
			line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
			choice := strings.TrimSpace(strings.ToLower(line))
			if choice != "y" && choice != "yes" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		if err := client.DeleteSheetTab(args[0], sheetID, title); err != nil {
			return formatError(err)
		}
		fmt.Printf("Deleted tab %q from %s\n", target, args[0])
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

var sheetsCopyCmd = &cobra.Command{
	Use:   "copy <fileId>",
	Short: "Duplicate a Google Sheet",
	Long: `Creates a copy of the spreadsheet and returns the new file's id.

Examples:
  porteden sheets copy google:SHEETID --name "Q2 Forecast (copy)"
  porteden sheets copy google:SHEETID --folder google:0B7_FOLDER`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		return runCopyFile(client, args[0], cmd)
	},
}

// ==================== HELPERS ====================

// buildWriteRequest assembles a WriteSheetValuesRequest from either --updates
// (batch mode) or the single-range --range + values flags. The two are mutually
// exclusive.
func buildWriteRequest(cmd *cobra.Command, inputOption string) (api.WriteSheetValuesRequest, error) {
	updatesJSON, _ := cmd.Flags().GetString("updates")

	if updatesJSON != "" {
		rangeStr, _ := cmd.Flags().GetString("range")
		valuesJSON, _ := cmd.Flags().GetString("values")
		csvStr, _ := cmd.Flags().GetString("csv")
		csvFile, _ := cmd.Flags().GetString("csv-file")
		if rangeStr != "" || valuesJSON != "" || csvStr != "" || csvFile != "" {
			return api.WriteSheetValuesRequest{}, errors.New("--updates is mutually exclusive with --range/--values/--csv/--csv-file")
		}

		var updates []api.SheetRangeUpdate
		if err := json.Unmarshal([]byte(updatesJSON), &updates); err != nil {
			return api.WriteSheetValuesRequest{}, fmt.Errorf("invalid --updates JSON: %w", err)
		}
		if len(updates) == 0 {
			return api.WriteSheetValuesRequest{}, errors.New("--updates must contain at least one range update")
		}

		return api.WriteSheetValuesRequest{
			Updates:          updates,
			ValueInputOption: inputOption,
		}, nil
	}

	rangeStr, _ := cmd.Flags().GetString("range")
	if rangeStr == "" {
		return api.WriteSheetValuesRequest{}, errors.New("--range is required (e.g., Sheet1!A1:C3), or use --updates for a batch write")
	}

	values, err := parseSheetValues(cmd)
	if err != nil {
		return api.WriteSheetValuesRequest{}, err
	}

	return api.WriteSheetValuesRequest{
		Range:            rangeStr,
		Values:           values,
		ValueInputOption: inputOption,
	}, nil
}

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
	sheetsCreateCmd.Flags().String("csv", "", `Inline CSV content to seed the first tab (use \\n for row separator)`)
	sheetsCreateCmd.Flags().String("csv-file", "", "Path to a CSV file to seed the first tab")

	// read flags
	sheetsReadCmd.Flags().String("range", "", `Cell range in A1 notation (e.g., Sheet1!A1:C10 or Sheet1)`)

	// content (bulk) flags
	sheetsContentCmd.Flags().String("ranges", "", "Comma-separated A1 ranges (e.g., 'Sheet1,Summary!A:C'). Pass-through, no per-tab cap.")
	sheetsContentCmd.Flags().Int("max-rows", 0, "Per-tab row cap when --ranges is omitted (server default 200, max 5000)")

	// write flags
	sheetsWriteCmd.Flags().String("range", "", `Target range in A1 notation (e.g., Sheet1!A1:C3)`)
	sheetsWriteCmd.Flags().String("values", "", `Values as JSON 2D array (e.g., '[["Name","Score"],["Alice",95]]')`)
	sheetsWriteCmd.Flags().String("csv", "", `Values as CSV string (use \\n for row separator)`)
	sheetsWriteCmd.Flags().String("csv-file", "", "Path to a CSV file")
	sheetsWriteCmd.Flags().String("updates", "", `Batch: JSON array of {"range","values"} objects (mutually exclusive with --range/--values/--csv/--csv-file)`)
	sheetsWriteCmd.Flags().Bool("raw", false, "Use RAW input mode (disables formula evaluation)")

	// read-tab flags
	sheetsReadTabCmd.Flags().String("title", "", "Tab title (e.g., Q2 Forecast)")
	sheetsReadTabCmd.Flags().Int("sheet-id", 0, "Numeric sheetId of the tab")

	// add-tab flags
	sheetsAddTabCmd.Flags().String("title", "", "Tab title (required, unique within the spreadsheet)")
	sheetsAddTabCmd.Flags().Int("rows", 0, "Initial grid row count (Google default 1000)")
	sheetsAddTabCmd.Flags().Int("cols", 0, "Initial grid column count (Google default 26)")
	sheetsAddTabCmd.Flags().Int("index", 0, "Zero-based position among existing tabs (default: append)")

	// delete-tab flags
	sheetsDeleteTabCmd.Flags().String("title", "", "Tab title to delete")
	sheetsDeleteTabCmd.Flags().Int("sheet-id", 0, "Numeric sheetId of the tab to delete")
	sheetsDeleteTabCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

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

	// copy flags
	addCopyFlags(sheetsCopyCmd)

	// Register sub-commands
	sheetsCmd.AddCommand(sheetsCreateCmd)
	sheetsCmd.AddCommand(sheetsInfoCmd)
	sheetsCmd.AddCommand(sheetsContentCmd)
	sheetsCmd.AddCommand(sheetsReadCmd)
	sheetsCmd.AddCommand(sheetsReadTabCmd)
	sheetsCmd.AddCommand(sheetsWriteCmd)
	sheetsCmd.AddCommand(sheetsAppendCmd)
	sheetsCmd.AddCommand(sheetsAddTabCmd)
	sheetsCmd.AddCommand(sheetsDeleteTabCmd)
	sheetsCmd.AddCommand(sheetsRenameCmd)
	sheetsCmd.AddCommand(sheetsDeleteCmd)
	sheetsCmd.AddCommand(sheetsShareCmd)
	sheetsCmd.AddCommand(sheetsPermissionsCmd)
	sheetsCmd.AddCommand(sheetsDownloadCmd)
	sheetsCmd.AddCommand(sheetsCopyCmd)
}
