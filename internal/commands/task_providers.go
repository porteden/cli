package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// providerSpec parameterises the per-provider top-level command set.
// One spec per provider; commands are built by newProviderCommand and
// delegate to the shared run* helpers in tasks.go with the provider
// code baked in. Adding a sixth provider is one line.
type providerSpec struct {
	name           string // command name: "notion", "monday", ...
	short          string
	providerCode   string // backend provider code: "NOTION", "MONDAY", ...
	supportsBlocks bool   // only Notion today
}

func newProviderCommand(spec providerSpec) *cobra.Command {
	root := &cobra.Command{
		Use:   spec.name,
		Short: spec.short,
		Long: fmt.Sprintf(`%s. Provider auto-set to %s; equivalent to
"porteden tasks <subcommand> --provider %s" but tighter at the prompt.`,
			spec.short, spec.providerCode, spec.providerCode),
	}

	root.AddCommand(
		newProviderBoardsCmd(spec),
		newProviderBoardCmd(spec),
		newProviderItemsCmd(spec),
		newProviderItemCmd(spec),
		newProviderCreateCmd(spec),
		newProviderUpdateCmd(spec),
		newProviderDeleteCmd(spec),
		newProviderSearchCmd(spec),
		newProviderCommentsCmd(spec),
		newProviderCommentCmd(spec),
	)
	if spec.supportsBlocks {
		root.AddCommand(
			newProviderBlocksCmd(spec),
			newProviderBlocksAppendCmd(spec),
		)
	}
	return root
}

func newProviderBoardsCmd(spec providerSpec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "boards",
		Short: "List boards",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			return runListBoards(client, spec.providerCode, cmd)
		},
	}
	addTaskBoardListFlags(cmd)
	return cmd
}

func newProviderBoardCmd(spec providerSpec) *cobra.Command {
	return &cobra.Command{
		Use:   "board <boardId>",
		Short: "Get a board's metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			return runGetBoard(client, spec.providerCode, args[0], cmd)
		},
	}
}

func newProviderItemsCmd(spec providerSpec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "items <boardId>",
		Short: "List items on a board",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			return runListItems(client, spec.providerCode, args[0], cmd)
		},
	}
	addTaskItemListFlags(cmd)
	return cmd
}

func newProviderItemCmd(spec providerSpec) *cobra.Command {
	return &cobra.Command{
		Use:   "item <itemId>",
		Short: "Get a single item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			return runGetItem(client, spec.providerCode, args[0], cmd)
		},
	}
}

func newProviderCreateCmd(spec providerSpec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <boardId>",
		Short: "Create a new item on a board",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			return runCreateItem(client, spec.providerCode, args[0], cmd)
		},
	}
	addTaskCreateFlags(cmd)
	return cmd
}

func newProviderUpdateCmd(spec providerSpec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <itemId>",
		Short: "Update an item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			return runUpdateItem(client, spec.providerCode, args[0], cmd)
		},
	}
	addTaskUpdateFlags(cmd)
	return cmd
}

func newProviderDeleteCmd(spec providerSpec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <itemId>",
		Short: "Delete (or archive, for Notion) an item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			yes, _ := cmd.Flags().GetBool("yes")
			return runDeleteItem(client, spec.providerCode, args[0], cmd, yes)
		},
	}
	addTaskDeleteFlags(cmd)
	return cmd
}

func newProviderSearchCmd(spec providerSpec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search items across in-scope boards",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			return runSearchTasks(client, spec.providerCode, cmd)
		},
	}
	addTaskSearchFlags(cmd)
	return cmd
}

func newProviderCommentsCmd(spec providerSpec) *cobra.Command {
	return &cobra.Command{
		Use:   "comments <itemId>",
		Short: "List comments on an item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			return runListComments(client, spec.providerCode, args[0], cmd)
		},
	}
}

func newProviderCommentCmd(spec providerSpec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment <itemId>",
		Short: "Add a comment to an item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			return runAddComment(client, spec.providerCode, args[0], cmd)
		},
	}
	addTaskCommentFlags(cmd)
	return cmd
}

func newProviderBlocksCmd(spec providerSpec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blocks <itemId>",
		Short: "List page-body blocks",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			return runListBlocks(client, spec.providerCode, args[0], cmd)
		},
	}
	addTaskBlockListFlags(cmd)
	return cmd
}

func newProviderBlocksAppendCmd(spec providerSpec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blocks-append <itemId>",
		Short: "Append page-body blocks to an item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			return runAppendBlocks(client, spec.providerCode, args[0], cmd)
		},
	}
	addTaskBlocksAppendFlags(cmd)
	return cmd
}

// Per-provider top-level commands. Initialised at package load so root.go
// can register them alongside the other commands.
var (
	notionCmd = newProviderCommand(providerSpec{
		name:           "notion",
		short:          "Notion task commands",
		providerCode:   "NOTION",
		supportsBlocks: true,
	})
	mondayCmd = newProviderCommand(providerSpec{
		name:         "monday",
		short:        "Monday.com task commands",
		providerCode: "MONDAY",
	})
	asanaCmd = newProviderCommand(providerSpec{
		name:         "asana",
		short:        "Asana task commands",
		providerCode: "ASANA",
	})
	jiraCmd = newProviderCommand(providerSpec{
		name:         "jira",
		short:        "Jira Cloud task commands",
		providerCode: "JIRA_CLOUD",
	})
	linearCmd = newProviderCommand(providerSpec{
		name:         "linear",
		short:        "Linear task commands",
		providerCode: "LINEAR",
	})
)
