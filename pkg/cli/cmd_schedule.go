package cli

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

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
	var useLaunchd bool
	var useCron bool
	var keepCount int
	var keepAge string

	cmd := &cobra.Command{
		Use:   "create NAME",
		Short: "Create a backup schedule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Name = args[0]
			opts.Surface = siphon.SurfaceCLI

			// Determine backend from flags.
			if useLaunchd && useCron {
				return fmt.Errorf("--launchd and --cron are mutually exclusive")
			}
			if useCron {
				opts.Backend = "cron"
			} else {
				opts.Backend = "launchd"
			}

			// Build retention policy if specified.
			if keepCount > 0 || keepAge != "" {
				rp := &siphon.RetentionPolicy{}
				if keepCount > 0 {
					rp.KeepCount = &keepCount
				}
				if keepAge != "" {
					rp.KeepAge = &keepAge
				}
				opts.Retention = rp
			}

			if err := deps.Siphon.CreateSchedule(cmd.Context(), &opts); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Schedule %q created (%s, %s at %s).\n",
				opts.Name, opts.Backend, opts.Interval, opts.Time)
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Connection, "connection", "", "connection to backup (required)")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "backup repository")
	cmd.Flags().StringVar(&opts.Database, "database", "", "override database")
	cmd.Flags().StringVar(&opts.BackupType, "type", "", "backup type (physical/logical/file)")
	cmd.Flags().StringVar(&opts.Compress, "compress", "", "compression (zstd/gzip/none)")
	cmd.Flags().StringVar(&opts.Message, "message", "", "default message for backups")
	cmd.Flags().StringVar(&opts.BackupNameFormat, "backup-name-format", "", "name template override")
	cmd.Flags().StringVar(&opts.Time, "time", "02:00", "schedule time (HH:MM)")
	cmd.Flags().StringVar(&opts.Interval, "interval", "daily", "interval: daily, hourly, weekly")
	cmd.Flags().BoolVar(&useLaunchd, "launchd", false, "use launchd backend (macOS)")
	cmd.Flags().BoolVar(&useCron, "cron", false, "use cron backend (Linux)")
	cmd.Flags().IntVar(&keepCount, "keep-count", 0, "retention: keep N most recent")
	cmd.Flags().StringVar(&keepAge, "keep-age", "", "retention: keep backups newer than duration (e.g., 720h, 30d)")

	_ = cmd.MarkFlagRequired("connection")

	_ = cmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"logical", "physical", "file"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("compress", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"zstd", "gzip", "none"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("interval", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"daily", "hourly", "weekly"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func newScheduleListCmd(deps *Deps) *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List backup schedules",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			schedules, err := deps.Siphon.ListSchedules(cmd.Context(), &siphon.ListSchedulesOptions{
				Surface: siphon.SurfaceCLI,
			})
			if err != nil {
				return err
			}

			switch format {
			case "json":
				data, _ := json.MarshalIndent(schedules, "", "  ")
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
			default:
				if len(schedules) == 0 {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No schedules configured.")
					return nil
				}
				w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
				_, _ = fmt.Fprintln(w, "NAME\tCONNECTION\tTIME\tINTERVAL\tBACKEND\tACTIVE")
				for _, s := range schedules {
					active := "no"
					if s.Active {
						active = "yes"
					}
					_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
						s.Name, s.Connection, s.Time, s.Interval, s.Backend, active)
				}
				return w.Flush()
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "text", "output format (text, json)")
	_ = cmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"text", "json"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func newScheduleRemoveCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:     "remove NAME",
		Aliases: []string{"rm"},
		Short:   "Remove a backup schedule",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := deps.Siphon.RemoveSchedule(cmd.Context(), &siphon.RemoveScheduleOptions{
				Name:    args[0],
				Surface: siphon.SurfaceCLI,
			})
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Schedule %q removed.\n", args[0])
			return nil
		},
	}
}
