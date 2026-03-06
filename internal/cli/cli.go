package cli

import (
	"github.com/spf13/cobra"
)

// NewRootCmd creates the root cobra command with all subcommands.
func NewRootCmd(version, commit, date string) *cobra.Command {
	root := &cobra.Command{
		Use:          "vaultquery",
		Short:        "Query Obsidian vault files by YAML frontmatter",
		Version:      version,
		SilenceUsage: true,
	}

	root.AddCommand(
		newQueryCmd(),
		newIndexCmd(),
		newReindexCmd(),
		newStatusCmd(),
	)

	root.PersistentFlags().String("vault", "", "path to vault root (default: current directory)")
	root.PersistentFlags().String("db", "", "path to SQLite database (default: XDG data dir)")

	return root
}

func newQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query [DQL]",
		Short: "Execute a DQL query against the vault index",
		Args:  cobra.ExactArgs(1),
		RunE:  runQuery,
	}
	cmd.Flags().Bool("sync", false, "sync the index before querying")
	return cmd
}

func newIndexCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "index",
		Short: "Build or update the vault index",
		RunE:  runIndex,
	}
}

func newReindexCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reindex",
		Short: "Drop and rebuild the vault index from scratch",
		RunE:  runReindex,
	}
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show index status (file count, last update, vault path)",
		RunE:  runStatus,
	}
}
