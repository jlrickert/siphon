package cli

import (
	"fmt"
	"text/tabwriter"
	"time"

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

// connectionNameCompletion returns a ValidArgsFunction that completes
// connection names from the resolved configuration.
func connectionNameCompletion(deps *Deps) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if deps.Siphon == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		connections, err := deps.Siphon.ListConnections(cmd.Context(), &siphon.ListConnectionsOptions{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		names := make([]string, 0, len(connections))
		for _, c := range connections {
			names = append(names, c.Name)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}

func newConnectionsListCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured connections",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			connections, err := deps.Siphon.ListConnections(cmd.Context(), &siphon.ListConnectionsOptions{})
			if err != nil {
				return err
			}
			if len(connections) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No connections configured.")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "NAME\tENGINE\tHOST\tPORT\tUSER\tDATABASE\tPASSWORD")
			for _, c := range connections {
				host := c.Host
				port := ""
				if c.Port != 0 {
					port = fmt.Sprintf("%d", c.Port)
				}
				pw := "no"
				if c.HasPassword {
					pw = "yes"
				}
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					c.Name, c.Engine, host, port, c.User, c.Database, pw)
			}
			return w.Flush()
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
			if err := deps.Siphon.AddConnection(cmd.Context(), &opts); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Connection %q added.\n", opts.Name)
			return nil
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

	_ = cmd.RegisterFlagCompletionFunc("engine", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"mariadb", "postgresql", "sqlite"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func newConnectionsRemoveCmd(deps *Deps) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:               "remove NAME",
		Aliases:           []string{"rm"},
		Short:             "Remove a database connection",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: connectionNameCompletion(deps),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := deps.Siphon.RemoveConnection(cmd.Context(), &siphon.RemoveConnectionOptions{
				Name:  args[0],
				Force: force,
			}); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Connection %q removed.\n", args[0])
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation")

	return cmd
}

func newConnectionsTestCmd(deps *Deps) *cobra.Command {
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:               "test NAME",
		Short:             "Test connectivity to a connection",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: connectionNameCompletion(deps),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := deps.Siphon.TestConnection(cmd.Context(), &siphon.TestConnectionOptions{
				Name:    args[0],
				Timeout: timeout,
			}); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Connection %q: OK\n", args[0])
			return nil
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Second, "connection timeout")
	_ = cmd.RegisterFlagCompletionFunc("timeout", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"1s", "5s", "10s", "30s"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}
