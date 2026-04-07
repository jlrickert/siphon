package cli

import (
	"fmt"

	"github.com/jlrickert/siphon/pkg/siphon"
	"github.com/spf13/cobra"
)

// NewScheduleCmd creates the schedule command tree with subcommands:
// create, list, remove.
func NewScheduleCmd(deps *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedule",
		Short: "Manage backup schedules",
	}

	cmd.AddCommand(
		newScheduleCreateCmd(deps),
		newScheduleListCmd(deps),
		newScheduleRemoveCmd(deps),
	)

	return cmd
}

func newScheduleCreateCmd(deps *Deps) *cobra.Command {
	var opts siphon.CreateScheduleOptions

	cmd := &cobra.Command{
		Use:   "create NAME",
		Short: "Create a backup schedule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Name = args[0]
			return deps.Siphon.CreateSchedule(cmd.Context(), &opts)
		},
	}

	cmd.Flags().StringVar(&opts.Connection, "connection", "", "target connection")
	cmd.Flags().StringVar(&opts.Database, "database", "", "target database")
	cmd.Flags().StringVar(&opts.Cron, "cron", "", "cron expression")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "backup repository")
	cmd.Flags().StringVar(&opts.BackupType, "type", "logical", "backup type")
	cmd.Flags().IntVar(&opts.Retain, "retain", 0, "number of backups to retain")

	return cmd
}

func newScheduleListCmd(deps *Deps) *cobra.Command {
	var opts siphon.ListSchedulesOptions

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List backup schedules",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := deps.Siphon.ListSchedules(cmd.Context(), &opts)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "schedule list: not yet implemented")
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Connection, "connection", "", "filter by connection")

	return cmd
}

func newScheduleRemoveCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:     "remove NAME",
		Aliases: []string{"rm"},
		Short:   "Remove a backup schedule",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return deps.Siphon.RemoveSchedule(cmd.Context(), &siphon.RemoveScheduleOptions{
				Name: args[0],
			})
		},
	}
}
