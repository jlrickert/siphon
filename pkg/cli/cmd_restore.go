package cli

import (
	"fmt"
	"strings"

	"github.com/jlrickert/siphon/pkg/siphon"
	"github.com/spf13/cobra"
)

// NewRestoreCmd creates the restore command.
func NewRestoreCmd(deps *Deps) *cobra.Command {
	var opts siphon.RestoreOptions
	var tablesFlag string

	cmd := &cobra.Command{
		Use:   "restore CONNECTION PATH",
		Short: "Restore a database from a backup",
		Long: `Restore a database from a backup. This is a destructive operation that
requires the --force flag. The backup path supports @repo/name syntax.

Examples:
  siphon restore dev /path/to/backup --force
  siphon restore dev @nightly/my-backup --force
  siphon restore dev @nightly/my-backup --force --database otherdb
  siphon restore dev /path/to/backup --force --tables users,orders
  siphon restore dev /path/to/backup --force --type logical`,
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: connectionNameCompletion(deps),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Connection = args[0]
			opts.Path = args[1]
			opts.Surface = siphon.SurfaceCLI

			// Parse comma-separated tables.
			if tablesFlag != "" {
				opts.Tables = strings.Split(tablesFlag, ",")
			}

			if err := deps.Siphon.Restore(cmd.Context(), &opts); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Restore completed successfully.")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "required for destructive restore")
	cmd.Flags().StringVar(&opts.Database, "database", "", "override target database name")
	cmd.Flags().StringVar(&tablesFlag, "tables", "", "comma-separated list of tables for partial restore")
	cmd.Flags().StringVar(&opts.Type, "type", "", "explicit backup type (physical, logical, file)")

	_ = cmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"logical", "physical", "file"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}
