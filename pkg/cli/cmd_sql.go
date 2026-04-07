package cli

import (
	"fmt"

	"github.com/jlrickert/siphon/pkg/siphon"
	"github.com/spf13/cobra"
)

// NewSQLCmd creates the sql command.
func NewSQLCmd(deps *Deps) *cobra.Command {
	var opts siphon.ExecuteSQLOptions

	cmd := &cobra.Command{
		Use:   "sql QUERY",
		Short: "Execute a SQL query",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Query = args[0]
			_, err := deps.Siphon.ExecuteSQL(cmd.Context(), &opts)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "sql: not yet implemented")
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Connection, "connection", "", "target connection")
	cmd.Flags().StringVar(&opts.Database, "database", "", "target database")
	cmd.Flags().StringVar(&opts.Format, "format", "table", "output format (table, json, csv)")

	return cmd
}
