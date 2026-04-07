package cli

import (
	"fmt"

	"github.com/jlrickert/siphon/pkg/siphon"
	"github.com/spf13/cobra"
)

// NewBackupCmd creates the backup command tree with subcommands:
// create, info, list, verify, label.
func NewBackupCmd(deps *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Manage database backups",
	}

	cmd.AddCommand(
		newBackupCreateCmd(deps),
		newBackupInfoCmd(deps),
		newBackupListCmd(deps),
		newBackupVerifyCmd(deps),
		newBackupLabelCmd(deps),
	)

	return cmd
}

func newBackupCreateCmd(deps *Deps) *cobra.Command {
	var opts siphon.BackupOptions

	cmd := &cobra.Command{
		Use:   "create CONN:DB",
		Short: "Create a database backup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Connection, opts.Database = siphon.ParseColonSyntax(args[0])
			return deps.Siphon.Backup(cmd.Context(), &opts)
		},
	}

	cmd.Flags().StringVar(&opts.BackupType, "type", "logical", "backup type (logical, physical, file)")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "backup repository")
	cmd.Flags().StringVar(&opts.Label, "label", "", "backup label")
	cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "overwrite existing backup")

	return cmd
}

func newBackupInfoCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "info BACKUP_ID",
		Short: "Show backup metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := deps.Siphon.BackupInfo(cmd.Context(), &siphon.BackupInfoOptions{
				BackupID: args[0],
			})
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "backup info: not yet implemented")
			return nil
		},
	}
}

func newBackupListCmd(deps *Deps) *cobra.Command {
	var opts siphon.ListBackupsOptions

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available backups",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := deps.Siphon.ListBackups(cmd.Context(), &opts)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "backup list: not yet implemented")
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Connection, "connection", "", "filter by connection")
	cmd.Flags().StringVar(&opts.Database, "database", "", "filter by database")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "filter by repository")
	cmd.Flags().IntVar(&opts.Limit, "limit", 0, "maximum results")

	return cmd
}

func newBackupVerifyCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "verify BACKUP_ID",
		Short: "Verify backup integrity",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return deps.Siphon.VerifyBackup(cmd.Context(), &siphon.VerifyBackupOptions{
				BackupID: args[0],
			})
		},
	}
}

func newBackupLabelCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "label BACKUP_ID LABEL",
		Short: "Set or update a backup label",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return deps.Siphon.LabelBackup(cmd.Context(), &siphon.LabelBackupOptions{
				BackupID: args[0],
				Label:    args[1],
			})
		},
	}
}
