package cli

import (
	"github.com/jlrickert/siphon/pkg/siphon"
	"github.com/spf13/cobra"
)

// NewRestoreCmd creates the restore command.
func NewRestoreCmd(deps *Deps) *cobra.Command {
	var opts siphon.RestoreOptions

	cmd := &cobra.Command{
		Use:   "restore BACKUP_ID",
		Short: "Restore a database from a backup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.BackupID = args[0]
			return deps.Siphon.Restore(cmd.Context(), &opts)
		},
	}

	cmd.Flags().StringVar(&opts.Connection, "connection", "", "target connection")
	cmd.Flags().StringVar(&opts.Database, "database", "", "target database")
	cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "allow destructive restore")

	return cmd
}
