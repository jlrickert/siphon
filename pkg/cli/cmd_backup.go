package cli

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/jlrickert/siphon/pkg/siphon"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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
		Use:               "create CONN:DB",
		Short:             "Create a database backup",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: connectionNameCompletion(deps),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Connection, opts.Database = siphon.ParseColonSyntax(args[0])
			desc, err := deps.Siphon.Backup(cmd.Context(), &opts)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Backup created: %s\n", desc.ID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Engine:     %s\n", desc.Engine)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Type:       %s\n", desc.BackupType)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Connection: %s\n", desc.Connection)
			if desc.Database != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Database:   %s\n", desc.Database)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Size:       %d bytes\n", desc.Size)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Checksum:   %s\n", desc.Checksum)
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.BackupType, "type", "", "backup type (logical, physical, file)")
	cmd.Flags().StringVar(&opts.Compress, "compress", "none", "compression algorithm (zstd, gzip, none)")
	cmd.Flags().StringVar(&opts.Name, "name", "", "explicit backup directory name")
	cmd.Flags().StringVarP(&opts.Message, "message", "m", "", "backup message")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "backup repository")
	cmd.Flags().StringVar(&opts.Label, "label", "", "backup label")
	cmd.Flags().StringVar(&opts.Database, "database", "", "database name")
	cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "overwrite existing backup")

	_ = cmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"logical", "physical", "file"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("compress", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"zstd", "gzip", "none"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func newBackupInfoCmd(deps *Deps) *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "info PATH",
		Short: "Show backup metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			desc, err := deps.Siphon.BackupInfo(cmd.Context(), &siphon.BackupInfoOptions{
				Path: args[0],
			})
			if err != nil {
				return err
			}
			return printBackupDescriptor(cmd, desc, format)
		},
	}

	cmd.Flags().StringVar(&format, "format", "text", "output format (text, json, yaml)")
	_ = cmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"text", "json", "yaml"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func newBackupListCmd(deps *Deps) *cobra.Command {
	var opts siphon.ListBackupsOptions
	var format string
	var sortBy string

	cmd := &cobra.Command{
		Use:   "list [PATH]",
		Short: "List available backups",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Repo = args[0]
			}
			backups, err := deps.Siphon.ListBackups(cmd.Context(), &opts)
			if err != nil {
				return err
			}

			// Apply sort.
			sortBackupDescriptors(backups, sortBy)

			switch format {
			case "json":
				data, _ := json.MarshalIndent(backups, "", "  ")
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
			case "yaml":
				data, _ := yaml.Marshal(backups)
				_, _ = fmt.Fprint(cmd.OutOrStdout(), string(data))
			default:
				if len(backups) == 0 {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No backups found.")
					return nil
				}
				w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
				_, _ = fmt.Fprintln(w, "ID\tENGINE\tTYPE\tCONNECTION\tDATABASE\tSIZE\tTIMESTAMP\tLABEL")
				for _, b := range backups {
					_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
						b.ID, b.Engine, b.BackupType, b.Connection, b.Database,
						b.Size, b.Timestamp.Format(time.RFC3339), b.Label)
				}
				return w.Flush()
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Connection, "connection", "", "filter by connection")
	cmd.Flags().StringVar(&opts.Database, "database", "", "filter by database")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "filter by repository")
	cmd.Flags().IntVar(&opts.Limit, "limit", 0, "maximum results")
	cmd.Flags().StringVar(&format, "format", "text", "output format (text, json, yaml)")
	cmd.Flags().StringVar(&sortBy, "sort", "date", "sort by (date, size, engine)")

	_ = cmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"text", "json", "yaml"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"date", "size", "engine"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func newBackupVerifyCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "verify PATH",
		Short: "Verify backup integrity",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := deps.Siphon.VerifyBackup(cmd.Context(), &siphon.VerifyBackupOptions{
				Path: args[0],
			})
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Backup verified: OK")
			return nil
		},
	}
}

func newBackupLabelCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "label PATH LABEL",
		Short: "Set or update a backup label",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := deps.Siphon.LabelBackup(cmd.Context(), &siphon.LabelBackupOptions{
				BackupID: args[0],
				Label:    args[1],
			})
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Label set to %q.\n", args[1])
			return nil
		},
	}
}

// --- helpers ---

func printBackupDescriptor(cmd *cobra.Command, desc *siphon.BackupDescriptor, format string) error {
	switch format {
	case "json":
		data, _ := json.MarshalIndent(desc, "", "  ")
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
	case "yaml":
		data, _ := yaml.Marshal(desc)
		_, _ = fmt.Fprint(cmd.OutOrStdout(), string(data))
	default:
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ID:         %s\n", desc.ID)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Engine:     %s\n", desc.Engine)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Type:       %s\n", desc.BackupType)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Connection: %s\n", desc.Connection)
		if desc.Database != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Database:   %s\n", desc.Database)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Timestamp:  %s\n", desc.Timestamp.Format(time.RFC3339))
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Size:       %d bytes\n", desc.Size)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Checksum:   %s\n", desc.Checksum)
		if desc.Label != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Label:      %s\n", desc.Label)
		}
	}
	return nil
}

func sortBackupDescriptors(descs []siphon.BackupDescriptor, sortBy string) {
	switch sortBy {
	case "size":
		sortBackupsBySize(descs)
	case "engine":
		sortBackupsByEngine(descs)
	default: // "date"
		sortBackupsByDate(descs)
	}
}

func sortBackupsByDate(descs []siphon.BackupDescriptor) {
	for i := 0; i < len(descs); i++ {
		for j := i + 1; j < len(descs); j++ {
			if descs[j].Timestamp.After(descs[i].Timestamp) {
				descs[i], descs[j] = descs[j], descs[i]
			}
		}
	}
}

func sortBackupsBySize(descs []siphon.BackupDescriptor) {
	for i := 0; i < len(descs); i++ {
		for j := i + 1; j < len(descs); j++ {
			if descs[j].Size > descs[i].Size {
				descs[i], descs[j] = descs[j], descs[i]
			}
		}
	}
}

func sortBackupsByEngine(descs []siphon.BackupDescriptor) {
	for i := 0; i < len(descs); i++ {
		for j := i + 1; j < len(descs); j++ {
			if descs[j].Engine < descs[i].Engine {
				descs[i], descs[j] = descs[j], descs[i]
			}
		}
	}
}
