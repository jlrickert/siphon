package cli

import (
	"fmt"

	"github.com/jlrickert/siphon/pkg/siphon"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// NewConfigCmd creates the config command tree.
func NewConfigCmd(deps *Deps) *cobra.Command {
	var explain bool

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show resolved configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := deps.Siphon.Config(cmd.Context(), &siphon.ConfigOptions{
				Explain: explain,
			})
			if err != nil {
				return err
			}

			if explain {
				// Display config with provenance annotations.
				data, err := yaml.Marshal(resolved.Config)
				if err != nil {
					return fmt.Errorf("marshaling config: %w", err)
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "# Resolved configuration")
				_, _ = fmt.Fprint(cmd.OutOrStdout(), string(data))

				if len(resolved.Provenance) > 0 {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\n# Provenance (field -> source)")
					provData, err := yaml.Marshal(resolved.Provenance)
					if err != nil {
						return fmt.Errorf("marshaling provenance: %w", err)
					}
					_, _ = fmt.Fprint(cmd.OutOrStdout(), string(provData))
				}
				return nil
			}

			// Display resolved config as YAML.
			data, err := yaml.Marshal(resolved.Config)
			if err != nil {
				return fmt.Errorf("marshaling config: %w", err)
			}
			_, _ = fmt.Fprint(cmd.OutOrStdout(), string(data))
			return nil
		},
	}

	cmd.Flags().BoolVar(&explain, "explain", false, "show per-field provenance")

	return cmd
}
