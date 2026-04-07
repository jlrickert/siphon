package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewMCPCmd creates the mcp command that launches the MCP JSON-RPC server.
func NewMCPCmd(deps *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start the MCP JSON-RPC server",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Wire to pkg/mcp.NewServer and run over stdio.
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "mcp server: not yet implemented")
			return nil
		},
	}

	return cmd
}
