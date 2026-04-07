package cli

import (
	"fmt"

	"github.com/jlrickert/siphon/pkg/siphon"
	"github.com/spf13/cobra"
)

// NewConfigCmd creates the config command tree.
func NewConfigCmd(deps *Deps) *cobra.Command {
	var explain bool

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show resolved configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := deps.Siphon.Config(cmd.Context(), &siphon.ConfigOptions{
				Explain: explain,
			})
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "config: not yet implemented")
			return nil
		},
	}

	cmd.Flags().BoolVar(&explain, "explain", false, "show per-field provenance")

	return cmd
}
