package cli

import (
	"github.com/jlrickert/siphon/pkg/siphon"
	"github.com/spf13/cobra"
)

// NewTransferCmd creates the transfer command.
func NewTransferCmd(deps *Deps) *cobra.Command {
	var opts siphon.TransferOptions

	cmd := &cobra.Command{
		Use:   "transfer SOURCE DESTINATION",
		Short: "Transfer tables between databases",
		Long:  "Transfer tables from SOURCE to DESTINATION using CONN:DATABASE syntax.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Source = args[0]
			opts.Destination = args[1]
			return deps.Siphon.Transfer(cmd.Context(), &opts)
		},
	}

	cmd.Flags().StringSliceVar(&opts.Tables, "tables", nil, "tables to transfer (default: all)")
	cmd.Flags().BoolVar(&opts.Truncate, "truncate", false, "truncate destination tables first")
	cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "allow destructive transfer")

	return cmd
}
