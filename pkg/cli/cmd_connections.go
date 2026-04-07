package cli

import (
	"fmt"

	"github.com/jlrickert/siphon/pkg/siphon"
	"github.com/spf13/cobra"
)

// NewConnectionsCmd creates the connections command tree with subcommands:
// list, add, remove, test.
func NewConnectionsCmd(deps *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "connections",
		Aliases: []string{"conn"},
		Short:   "Manage database connections",
	}

	cmd.AddCommand(
		newConnectionsListCmd(deps),
		newConnectionsAddCmd(deps),
		newConnectionsRemoveCmd(deps),
		newConnectionsTestCmd(deps),
	)

	return cmd
}

func newConnectionsListCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured connections",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := deps.Siphon.ListConnections(cmd.Context(), &siphon.ListConnectionsOptions{})
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "connections list: not yet implemented")
			return nil
		},
	}
}

func newConnectionsAddCmd(deps *Deps) *cobra.Command {
	var opts siphon.AddConnectionOptions

	cmd := &cobra.Command{
		Use:   "add NAME",
		Short: "Add a database connection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Name = args[0]
			return deps.Siphon.AddConnection(cmd.Context(), &opts)
		},
	}

	cmd.Flags().StringVar((*string)(&opts.Engine), "engine", "", "database engine (mariadb, postgresql, sqlite)")
	cmd.Flags().StringVar(&opts.Host, "host", "", "database host")
	cmd.Flags().IntVar(&opts.Port, "port", 0, "database port")
	cmd.Flags().StringVar(&opts.User, "user", "", "database user")
	cmd.Flags().StringVar(&opts.Password, "password", "", "database password")
	cmd.Flags().StringVar(&opts.PasswordEnv, "password-env", "", "env var containing password")
	cmd.Flags().StringVar(&opts.Database, "database", "", "default database")
	cmd.Flags().StringVar(&opts.Path, "path", "", "SQLite file path")

	return cmd
}

func newConnectionsRemoveCmd(deps *Deps) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "remove NAME",
		Aliases: []string{"rm"},
		Short:   "Remove a database connection",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return deps.Siphon.RemoveConnection(cmd.Context(), &siphon.RemoveConnectionOptions{
				Name:  args[0],
				Force: force,
			})
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation")

	return cmd
}

func newConnectionsTestCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "test NAME",
		Short: "Test connectivity to a connection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return deps.Siphon.TestConnection(cmd.Context(), &siphon.TestConnectionOptions{
				Name: args[0],
			})
		},
	}
}
