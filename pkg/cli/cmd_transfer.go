package cli

import (
	"fmt"
	"strings"

	"github.com/jlrickert/siphon/pkg/siphon"
	"github.com/spf13/cobra"
)

// NewTransferCmd creates the transfer command.
func NewTransferCmd(deps *Deps) *cobra.Command {
	var opts siphon.TransferOptions
	var tablesFlag string
	var groupsFlag []string
	var excludeGroupsFlag []string

	cmd := &cobra.Command{
		Use:   "transfer SOURCE TARGET",
		Short: "Transfer tables between databases",
		Long: `Transfer tables from SOURCE to TARGET using CONN:DATABASE syntax.

Positional args use colon syntax: CONN:DATABASE
  siphon transfer prod:myapp local:dev

Tables are transferred in foreign key dependency order. Use --on-conflict
to control how existing rows are handled.

Examples:
  siphon transfer prod:myapp local:dev
  siphon transfer prod:myapp local:dev --tables users,orders
  siphon transfer prod:myapp local:dev --group core --exclude-group logs
  siphon transfer prod:myapp local:dev --all-tables --on-conflict overwrite`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Source = args[0]
			opts.Target = args[1]
			opts.Surface = siphon.SurfaceCLI

			// Parse comma-separated tables.
			if tablesFlag != "" {
				opts.Tables = strings.Split(tablesFlag, ",")
			}

			opts.Groups = groupsFlag
			opts.ExcludeGroups = excludeGroupsFlag

			if err := deps.Siphon.Transfer(cmd.Context(), &opts); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Transfer completed successfully.")
			return nil
		},
	}

	cmd.Flags().StringVar(&tablesFlag, "tables", "", "comma-separated table list")
	cmd.Flags().StringSliceVar(&groupsFlag, "group", nil, "named table group (repeatable)")
	cmd.Flags().StringSliceVar(&excludeGroupsFlag, "exclude-group", nil, "named table group to exclude (repeatable)")
	cmd.Flags().BoolVar(&opts.AllTables, "all-tables", false, "transfer all tables")
	cmd.Flags().StringVar(&opts.OnConflict, "on-conflict", "skip", "conflict strategy: skip, overwrite, merge")
	cmd.Flags().BoolVar(&opts.Confirm, "confirm", false, "confirm destructive operation")

	_ = cmd.RegisterFlagCompletionFunc("on-conflict", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"skip", "overwrite", "merge"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}
