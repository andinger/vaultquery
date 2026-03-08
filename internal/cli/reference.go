package cli

import (
	_ "embed"
	"fmt"

	"github.com/spf13/cobra"
)

//go:embed reference.md
var referenceDoc string

func newReferenceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reference",
		Short: "Print the vaultquery reference documentation to stdout",
		Long:  "Prints a comprehensive reference for vaultquery usage, DQL syntax, and built-in functions. Pipe to a file to create a local reference: vaultquery reference > ~/.claude/references/vaultquery.md",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Print(referenceDoc)
			return nil
		},
	}
}
