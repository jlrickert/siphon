package cli

import (
	"fmt"

	"github.com/jlrickert/cli-toolkit/toolkit"
	"github.com/jlrickert/siphon/pkg/siphon"
	"github.com/spf13/cobra"
)

// Deps holds shared dependencies for all CLI subcommands.
type Deps struct {
	Runtime    *toolkit.Runtime
	ConfigPath string
	LogLevel   string
	LogFile    string

	Siphon *siphon.Siphon
}

// NewRootCmd builds the root cobra command, wires persistent flags, and
// initializes services.
func NewRootCmd(deps *Deps) *cobra.Command {
	if deps == nil {
		deps = &Deps{}
	}

	cmd := &cobra.Command{
		Use:           "siphon",
		Short:         "Database backup, restore, transfer, and management tool",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Allow pre-initialized Siphon (e.g., from parity tests).
			if deps.Siphon != nil {
				return nil
			}

			rt := deps.Runtime
			if rt == nil {
				return fmt.Errorf("runtime is required")
			}

			wd, err := rt.Getwd()
			if err != nil {
				return err
			}
			s, err := siphon.New(siphon.SiphonOptions{
				Root:       wd,
				ConfigPath: deps.ConfigPath,
				Runtime:    rt,
			})
			if err != nil {
				return err
			}
			deps.Siphon = s
			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(&deps.ConfigPath, "config", "c", "", "path to config file")
	cmd.PersistentFlags().StringVar(&deps.LogLevel, "log-level", "", "minimum log level (default \"error\")")
	cmd.PersistentFlags().StringVar(&deps.LogFile, "log-file", "", "write logs to file (default stderr)")

	cmd.AddCommand(
		NewVersionCmd(deps),
		NewConfigCmd(deps),
		NewConnectionsCmd(deps),
		NewBackupCmd(deps),
		NewRestoreCmd(deps),
		NewTransferCmd(deps),
		NewSQLCmd(deps),
		NewMCPCmd(deps),
		NewScheduleCmd(deps),
	)

	return cmd
}
