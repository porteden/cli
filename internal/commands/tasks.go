package commands

import (
	"bufio"
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

var tasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "Task management commands (Monday, Asana, Jira, Linear, Notion)",
	Long: `Provider-agnostic task management across Monday.com, Asana, Jira Cloud,
Linear, and Notion. The backend exposes a unified board/item/comment model
behind a token firewall.

When the account has exactly one task provider connected, the provider is
auto-resolved. With more than one connected, pass --provider explicitly or
use the provider-specific top-level commands (porteden notion, monday,
asana, jira, linear) which preset it for you.

Examples:
  porteden tasks providers -jc
  porteden tasks boards --provider NOTION -jc
  porteden tasks items <boardId> --provider NOTION -q "dark mode" -jc
  porteden tasks create <boardId> --name "Fix login" --fields "status=To Do"
  porteden tasks search -q "auth" --provider NOTION -jc`,
}

// ==================== PROVIDERS ====================

var tasksProvidersCmd = &cobra.Command{
	Use:   "providers",
	Short: "List task providers connected to the account",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		return runListProviders(client, cmd)
	},
}

// ==================== BOARDS ====================

var tasksBoardsCmd = &cobra.Command{
	Use:   "boards",
	Short: "List boards in scope for the token",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		provider, _ := cmd.Flags().GetString("provider")
		return runListBoards(client, provider, cmd)
	},
}

var tasksBoardCmd = &cobra.Command{
	Use:   "board <boardId>",
	Short: "Get a board's metadata (groups + columns)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		provider, _ := cmd.Flags().GetString("provider")
		return runGetBoard(client, provider, args[0], cmd)
	},
}

// ==================== ITEMS ====================

var tasksItemsCmd = &cobra.Command{
	Use:   "items <boardId>",
	Short: "List items on a board",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		provider, _ := cmd.Flags().GetString("provider")
		return runListItems(client, provider, args[0], cmd)
	},
}

var tasksItemCmd = &cobra.Command{
	Use:   "item <itemId>",
	Short: "Get a single item",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		provider, _ := cmd.Flags().GetString("provider")
		return runGetItem(client, provider, args[0], cmd)
	},
}

var tasksCreateCmd = &cobra.Command{
	Use:   "create <boardId>",
	Short: "Create a new item on a board",
	Long: `Create a new item. --fields key=value can be repeated for column values.

Examples:
  porteden tasks create <boardId> --name "Fix login bug" --fields "status=To Do" --fields "priority=Critical"
  porteden tasks create <boardId> --name "Plan check" --group "Status|status|To Do"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		provider, _ := cmd.Flags().GetString("provider")
		return runCreateItem(client, provider, args[0], cmd)
	},
}

var tasksUpdateCmd = &cobra.Command{
	Use:   "update <itemId>",
	Short: "Update an item (PATCH)",
	Long: `Update column values on an item. --fields key=value can be repeated.
Empty --fields returns 400; the backend silently drops keys not in the
token's writability mask and lists them in rejectedFields on the response.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		provider, _ := cmd.Flags().GetString("provider")
		return runUpdateItem(client, provider, args[0], cmd)
	},
}

var tasksDeleteCmd = &cobra.Command{
	Use:   "delete <itemId>",
	Short: "Delete (or archive, for Notion) an item",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		provider, _ := cmd.Flags().GetString("provider")
		yes, _ := cmd.Flags().GetBool("yes")
		return runDeleteItem(client, provider, args[0], cmd, yes)
	},
}

// ==================== SEARCH ====================

var tasksSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search items across in-scope boards",
	Long: `Search item titles across boards in the token's scope.
--boards restricts to a comma-separated subset of board IDs. IDs not in
scope are silently dropped. The endpoint is single-shot — bump --limit
(max 200) to widen results, or pass --all as a shortcut for --limit 200.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		provider, _ := cmd.Flags().GetString("provider")
		return runSearchTasks(client, provider, cmd)
	},
}

// ==================== COMMENTS ====================

var tasksCommentsCmd = &cobra.Command{
	Use:   "comments <itemId>",
	Short: "List comments on an item",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		provider, _ := cmd.Flags().GetString("provider")
		return runListComments(client, provider, args[0], cmd)
	},
}

var tasksCommentCmd = &cobra.Command{
	Use:   "comment <itemId>",
	Short: "Add a comment to an item",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		provider, _ := cmd.Flags().GetString("provider")
		return runAddComment(client, provider, args[0], cmd)
	},
}

// ==================== BLOCKS (Notion-only) ====================

var tasksBlocksCmd = &cobra.Command{
	Use:   "blocks <itemId>",
	Short: "List page-body blocks (Notion only)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		provider, _ := cmd.Flags().GetString("provider")
		return runListBlocks(client, provider, args[0], cmd)
	},
}

var tasksBlocksAppendCmd = &cobra.Command{
	Use:   "blocks-append <itemId>",
	Short: "Append page-body blocks to an item (Notion only)",
	Long: `Append blocks to a Notion page. Pass --blocks with an inline JSON
array, or --blocks-file pointing at a file containing the JSON array.
Each block: {"type": "...", "text": "...", "metadata": {...}}.

Supported types: paragraph, heading_1/2/3, bulleted_list_item,
numbered_list_item, to_do, code, quote, callout, toggle, divider.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		provider, _ := cmd.Flags().GetString("provider")
		return runAppendBlocks(client, provider, args[0], cmd)
	},
}

// ==================== SHARED RUN HELPERS ====================
// Called by both tasks*Cmd above and the per-provider commands in
// task_providers.go. The provider arg is passed through to the API
// client — when empty, the backend auto-resolves single-provider accounts.

func runListProviders(client *api.Client, cmd *cobra.Command) error {
	resp, err := client.GetTaskProviders()
	if err != nil {
		return formatError(err)
	}
	output.PrintWithOptions(&resp, getOutputFormat(cmd), output.PrintOptions{
		Compact: IsCompactMode(),
	})
	return nil
}

func runListBoards(client *api.Client, provider string, cmd *cobra.Command) error {
	params := api.TaskBoardListParams{Provider: provider}
	if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
		params.Limit = v
	}
	if v, _ := cmd.Flags().GetString("cursor"); v != "" {
		params.Cursor = v
	}
	if v, _ := cmd.Flags().GetInt("page"); v > 0 {
		params.Page = v
	}

	all, _ := cmd.Flags().GetBool("all")
	var resp *api.TaskBoardsResponse
	var err error
	if all {
		resp, err = client.GetAllTaskBoards(params)
		if resp != nil && (resp.NextCursor != nil || resp.NextPage != nil) {
			fmt.Fprintln(os.Stderr, "Warning: pagination cap reached (50 pages). Results may be incomplete.")
		}
	} else {
		resp, err = client.GetTaskBoards(params)
	}
	if err != nil {
		return formatError(err)
	}
	output.PrintWithOptions(resp, getOutputFormat(cmd), output.PrintOptions{
		Compact: IsCompactMode(),
	})
	return nil
}

func runGetBoard(client *api.Client, provider, boardID string, cmd *cobra.Command) error {
	resp, err := client.GetTaskBoard(boardID, provider)
	if err != nil {
		return formatError(err)
	}
	output.PrintWithOptions(resp, getOutputFormat(cmd), output.PrintOptions{
		Compact: IsCompactMode(),
	})
	return nil
}

func runListItems(client *api.Client, provider, boardID string, cmd *cobra.Command) error {
	params := buildTaskItemListParams(cmd, provider, boardID)

	all, _ := cmd.Flags().GetBool("all")
	var resp *api.TaskItemsResponse
	var err error
	if all {
		resp, err = client.GetAllBoardItems(params)
		if resp != nil && (resp.NextCursor != nil || resp.NextPage != nil) {
			fmt.Fprintln(os.Stderr, "Warning: pagination cap reached (50 pages). Results may be incomplete.")
		}
	} else {
		resp, err = client.GetBoardItems(params)
	}
	if err != nil {
		return formatError(err)
	}
	output.PrintWithOptions(resp, getOutputFormat(cmd), output.PrintOptions{
		Compact: IsCompactMode(),
	})
	return nil
}

// buildTaskItemListParams pulls the items-list flag set into a params struct.
// Extracted because items lists carry six knobs (limit/cursor/page/group/query/status)
// and inlining all six reads twice (one-shot vs --all branches) would crowd
// the caller; the smaller list helpers stay inline.
func buildTaskItemListParams(cmd *cobra.Command, provider, boardID string) api.TaskItemListParams {
	params := api.TaskItemListParams{Provider: provider, BoardID: boardID}
	if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
		params.Limit = v
	}
	if v, _ := cmd.Flags().GetString("cursor"); v != "" {
		params.Cursor = v
	}
	if v, _ := cmd.Flags().GetInt("page"); v > 0 {
		params.Page = v
	}
	if v, _ := cmd.Flags().GetString("group"); v != "" {
		params.GroupID = v
	}
	if v, _ := cmd.Flags().GetString("query"); v != "" {
		params.Query = v
	}
	if v, _ := cmd.Flags().GetString("status"); v != "" {
		params.Status = v
	}
	return params
}

func runGetItem(client *api.Client, provider, itemID string, cmd *cobra.Command) error {
	resp, err := client.GetTaskItem(itemID, provider)
	if err != nil {
		return formatError(err)
	}
	output.PrintWithOptions(resp, getOutputFormat(cmd), output.PrintOptions{
		Compact: IsCompactMode(),
	})
	return nil
}

func runCreateItem(client *api.Client, provider, boardID string, cmd *cobra.Command) error {
	name, _ := cmd.Flags().GetString("name")
	if strings.TrimSpace(name) == "" {
		return errors.New("--name is required")
	}
	group, _ := cmd.Flags().GetString("group")
	fieldsRaw, _ := cmd.Flags().GetStringArray("fields")
	fields, err := parseFieldsFlag(fieldsRaw)
	if err != nil {
		return err
	}

	req := api.CreateTaskItemRequest{Name: name}
	if group != "" {
		req.GroupID = &group
	}
	if len(fields) > 0 {
		req.Fields = fields
	}

	result, err := client.CreateTaskItem(boardID, provider, req)
	if err != nil {
		return formatError(err)
	}
	output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
		Compact: IsCompactMode(),
	})
	return nil
}

func runUpdateItem(client *api.Client, provider, itemID string, cmd *cobra.Command) error {
	fieldsRaw, _ := cmd.Flags().GetStringArray("fields")
	if len(fieldsRaw) == 0 {
		return errors.New("--fields is required (e.g. --fields \"status=Done\")")
	}
	fields, err := parseFieldsFlag(fieldsRaw)
	if err != nil {
		return err
	}

	result, err := client.UpdateTaskItem(itemID, provider, api.UpdateTaskItemRequest{Fields: fields})
	if err != nil {
		return formatError(err)
	}
	output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
		Compact: IsCompactMode(),
	})
	return nil
}

func runDeleteItem(client *api.Client, provider, itemID string, cmd *cobra.Command, yes bool) error {
	if !yes && auth.IsInteractiveTerminal() {
		fmt.Printf("Delete item '%s'? [y/N]: ", itemID)
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		choice := strings.TrimSpace(strings.ToLower(line))
		if choice != "y" && choice != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	result, err := client.DeleteTaskItem(itemID, provider)
	if err != nil {
		return formatError(err)
	}
	output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
		Compact: IsCompactMode(),
	})
	return nil
}

func runSearchTasks(client *api.Client, provider string, cmd *cobra.Command) error {
	query, _ := cmd.Flags().GetString("query")
	if strings.TrimSpace(query) == "" {
		return errors.New("-q/--query is required")
	}
	limit, _ := cmd.Flags().GetInt("limit")
	all, _ := cmd.Flags().GetBool("all")
	if all && limit < 200 {
		limit = 200
	}
	boardsRaw, _ := cmd.Flags().GetString("boards")

	params := api.TaskSearchParams{
		Provider: provider,
		Query:    query,
		Limit:    limit,
	}
	if boardsRaw != "" {
		for _, id := range strings.Split(boardsRaw, ",") {
			if id = strings.TrimSpace(id); id != "" {
				params.BoardIDs = append(params.BoardIDs, id)
			}
		}
	}

	resp, err := client.SearchTasks(params)
	if err != nil {
		return formatError(err)
	}
	output.PrintWithOptions(resp, getOutputFormat(cmd), output.PrintOptions{
		Compact: IsCompactMode(),
	})
	return nil
}

func runListComments(client *api.Client, provider, itemID string, cmd *cobra.Command) error {
	resp, err := client.GetItemComments(itemID, provider)
	if err != nil {
		return formatError(err)
	}
	output.PrintWithOptions(resp, getOutputFormat(cmd), output.PrintOptions{
		Compact: IsCompactMode(),
	})
	return nil
}

func runAddComment(client *api.Client, provider, itemID string, cmd *cobra.Command) error {
	body, _ := cmd.Flags().GetString("body")
	if strings.TrimSpace(body) == "" {
		return errors.New("--body is required")
	}
	result, err := client.AddItemComment(itemID, provider, body)
	if err != nil {
		return formatError(err)
	}
	output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
		Compact: IsCompactMode(),
	})
	return nil
}

func runListBlocks(client *api.Client, provider, itemID string, cmd *cobra.Command) error {
	params := api.TaskBlockListParams{Provider: provider, ItemID: itemID}
	if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
		params.Limit = v
	}
	if v, _ := cmd.Flags().GetString("cursor"); v != "" {
		params.Cursor = v
	}

	all, _ := cmd.Flags().GetBool("all")
	var resp *api.TaskBlockListResponse
	var err error
	if all {
		resp, err = client.GetAllItemBlocks(params)
		if resp != nil && resp.HasMore {
			fmt.Fprintln(os.Stderr, "Warning: pagination cap reached (50 pages). Results may be incomplete.")
		}
	} else {
		resp, err = client.GetItemBlocks(params)
	}
	if err != nil {
		return formatError(err)
	}
	output.PrintWithOptions(resp, getOutputFormat(cmd), output.PrintOptions{
		Compact: IsCompactMode(),
	})
	return nil
}

func runAppendBlocks(client *api.Client, provider, itemID string, cmd *cobra.Command) error {
	inline, _ := cmd.Flags().GetString("blocks")
	file, _ := cmd.Flags().GetString("blocks-file")
	blocks, err := parseBlocksInput(inline, file)
	if err != nil {
		return err
	}

	result, err := client.AppendItemBlocks(itemID, provider, blocks)
	if err != nil {
		return formatError(err)
	}
	output.PrintWithOptions(result, getOutputFormat(cmd), output.PrintOptions{
		Compact: IsCompactMode(),
	})
	return nil
}

// ==================== PRIVATE HELPERS ====================

// parseFieldsFlag parses a repeatable --fields key=value flag into a map.
// Errors on missing '=' or duplicate keys (silently overwriting would
// surprise the caller on update; explicit collision is safer).
func parseFieldsFlag(pairs []string) (map[string]string, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(pairs))
	for _, raw := range pairs {
		eq := strings.IndexByte(raw, '=')
		if eq < 0 {
			return nil, fmt.Errorf("--fields %q: expected key=value", raw)
		}
		key := strings.TrimSpace(raw[:eq])
		if key == "" {
			return nil, fmt.Errorf("--fields %q: empty key", raw)
		}
		if _, exists := out[key]; exists {
			return nil, fmt.Errorf("--fields: duplicate key %q", key)
		}
		out[key] = raw[eq+1:]
	}
	return out, nil
}

// parseBlocksInput decodes --blocks (inline JSON) or --blocks-file (path to
// a JSON array of AppendBlockInput). Exactly one of the two must be set.
func parseBlocksInput(inline, file string) ([]api.AppendBlockInput, error) {
	if inline == "" && file == "" {
		return nil, errors.New("provide --blocks (inline JSON array) or --blocks-file")
	}
	if inline != "" && file != "" {
		return nil, errors.New("--blocks and --blocks-file are mutually exclusive")
	}

	raw := []byte(inline)
	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("cannot read --blocks-file: %w", err)
		}
		raw = data
	}

	var blocks []api.AppendBlockInput
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, fmt.Errorf("invalid blocks JSON: %w", err)
	}
	if len(blocks) == 0 {
		return nil, errors.New("blocks array is empty")
	}
	return blocks, nil
}

// ==================== FLAG HELPERS ====================
// Shared with task_providers.go so the per-provider commands register the
// same flag set as the equivalent tasks subcommand.

func addTaskProviderFlag(cmd *cobra.Command) {
	cmd.Flags().String("provider", "", "Provider code (MONDAY, ASANA, JIRA_CLOUD, LINEAR, NOTION). Auto-resolved when only one is connected.")
}

func addTaskBoardListFlags(cmd *cobra.Command) {
	cmd.Flags().Int("limit", 25, "Results per page (1-200)")
	cmd.Flags().String("cursor", "", "Pagination cursor (cursor providers)")
	cmd.Flags().Int("page", 0, "Page number (offset providers)")
	cmd.Flags().Bool("all", false, "Auto-paginate (safety cap: 50 pages)")
}

func addTaskItemListFlags(cmd *cobra.Command) {
	cmd.Flags().Int("limit", 50, "Results per page (1-200)")
	cmd.Flags().String("cursor", "", "Pagination cursor (cursor providers)")
	cmd.Flags().Int("page", 0, "Page number (offset providers)")
	cmd.Flags().Bool("all", false, "Auto-paginate (safety cap: 50 pages)")
	cmd.Flags().String("group", "", "Filter to a group/section/status id")
	cmd.Flags().StringP("query", "q", "", "Free-text search across item titles")
	cmd.Flags().String("status", "", "Filter by status value")
}

func addTaskCreateFlags(cmd *cobra.Command) {
	cmd.Flags().String("name", "", "Item title (required, non-empty)")
	cmd.Flags().String("group", "", "Target group/section id")
	cmd.Flags().StringArray("fields", nil, "Column value as key=value (repeatable)")
}

func addTaskUpdateFlags(cmd *cobra.Command) {
	cmd.Flags().StringArray("fields", nil, "Column value as key=value (repeatable, at least one required)")
}

func addTaskDeleteFlags(cmd *cobra.Command) {
	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
}

func addTaskSearchFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("query", "q", "", "Search text across item titles (required)")
	cmd.Flags().Int("limit", 100, "Total result cap (1-200)")
	cmd.Flags().String("boards", "", "Comma-separated board IDs to restrict the search")
	cmd.Flags().Bool("all", false, "Bump --limit to its maximum (200)")
}

func addTaskCommentFlags(cmd *cobra.Command) {
	cmd.Flags().String("body", "", "Comment body (required, non-empty)")
}

func addTaskBlockListFlags(cmd *cobra.Command) {
	cmd.Flags().Int("limit", 100, "Results per page (1-100)")
	cmd.Flags().String("cursor", "", "Pagination cursor")
	cmd.Flags().Bool("all", false, "Auto-paginate (safety cap: 50 pages)")
}

func addTaskBlocksAppendFlags(cmd *cobra.Command) {
	cmd.Flags().String("blocks", "", "Inline JSON array of block objects")
	cmd.Flags().String("blocks-file", "", "Path to a JSON file with a block array")
}

func init() {
	// Register flags. Each subcommand gets --provider; per-operation flags
	// come from the shared helpers above (so the per-provider wrappers
	// register the same set).
	for _, c := range []*cobra.Command{
		tasksBoardsCmd, tasksBoardCmd, tasksItemsCmd, tasksItemCmd,
		tasksCreateCmd, tasksUpdateCmd, tasksDeleteCmd, tasksSearchCmd,
		tasksCommentsCmd, tasksCommentCmd, tasksBlocksCmd, tasksBlocksAppendCmd,
	} {
		addTaskProviderFlag(c)
	}
	addTaskBoardListFlags(tasksBoardsCmd)
	addTaskItemListFlags(tasksItemsCmd)
	addTaskCreateFlags(tasksCreateCmd)
	addTaskUpdateFlags(tasksUpdateCmd)
	addTaskDeleteFlags(tasksDeleteCmd)
	addTaskSearchFlags(tasksSearchCmd)
	addTaskCommentFlags(tasksCommentCmd)
	addTaskBlockListFlags(tasksBlocksCmd)
	addTaskBlocksAppendFlags(tasksBlocksAppendCmd)

	tasksCmd.AddCommand(tasksProvidersCmd)
	tasksCmd.AddCommand(tasksBoardsCmd)
	tasksCmd.AddCommand(tasksBoardCmd)
	tasksCmd.AddCommand(tasksItemsCmd)
	tasksCmd.AddCommand(tasksItemCmd)
	tasksCmd.AddCommand(tasksCreateCmd)
	tasksCmd.AddCommand(tasksUpdateCmd)
	tasksCmd.AddCommand(tasksDeleteCmd)
	tasksCmd.AddCommand(tasksSearchCmd)
	tasksCmd.AddCommand(tasksCommentsCmd)
	tasksCmd.AddCommand(tasksCommentCmd)
	tasksCmd.AddCommand(tasksBlocksCmd)
	tasksCmd.AddCommand(tasksBlocksAppendCmd)
}
