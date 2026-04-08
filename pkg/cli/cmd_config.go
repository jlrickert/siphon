package cli

import (
	"fmt"
	"os/exec"

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

	cmd.AddCommand(newConfigInitCmd(deps))
	cmd.AddCommand(newConfigTemplateCmd(deps))
	cmd.AddCommand(newConfigEditCmd(deps))

	return cmd
}

// resolveScope returns the ConfigScope from mutually exclusive flags.
func resolveScope(user, project, local bool) (siphon.ConfigScope, error) {
	count := 0
	if user {
		count++
	}
	if project {
		count++
	}
	if local {
		count++
	}
	if count == 0 {
		return "", fmt.Errorf("one of --user, --project, or --local is required")
	}
	if count > 1 {
		return "", fmt.Errorf("--user, --project, and --local are mutually exclusive")
	}
	switch {
	case user:
		return siphon.ConfigScopeUser, nil
	case project:
		return siphon.ConfigScopeProject, nil
	default:
		return siphon.ConfigScopeLocal, nil
	}
}

func newConfigInitCmd(deps *Deps) *cobra.Command {
	var (
		user    bool
		project bool
		local   bool
		force   bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a config file at the specified scope",
		RunE: func(cmd *cobra.Command, args []string) error {
			scope, err := resolveScope(user, project, local)
			if err != nil {
				return err
			}

			result, err := deps.Siphon.ConfigInit(cmd.Context(), &siphon.ConfigInitOptions{
				Scope: scope,
				Force: force,
			})
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", result.Path)
			return nil
		},
	}

	cmd.Flags().BoolVar(&user, "user", false, "user config (~/.config/siphon/config.yaml)")
	cmd.Flags().BoolVar(&project, "project", false, "project config (.siphon/config.yaml)")
	cmd.Flags().BoolVar(&local, "local", false, "local config (.siphon/config.local.yaml, gitignored)")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite if file exists")
	cmd.MarkFlagsMutuallyExclusive("user", "project", "local")

	return cmd
}

func newConfigTemplateCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "template",
		Short: "Print an annotated config template to stdout",
		RunE: func(cmd *cobra.Command, args []string) error {
			tmpl, err := deps.Siphon.ConfigTemplate(cmd.Context(), &siphon.ConfigTemplateOptions{})
			if err != nil {
				return err
			}
			_, _ = fmt.Fprint(cmd.OutOrStdout(), tmpl)
			return nil
		},
	}
}

func newConfigEditCmd(deps *Deps) *cobra.Command {
	var (
		user    bool
		project bool
		local   bool
	)

	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Open config file in $EDITOR",
		RunE: func(cmd *cobra.Command, args []string) error {
			scope, err := resolveScope(user, project, local)
			if err != nil {
				return err
			}

			result, err := deps.Siphon.ConfigEdit(cmd.Context(), &siphon.ConfigEditOptions{
				Scope: scope,
			})
			if err != nil {
				return err
			}

			rt := deps.Runtime
			editor := rt.Env().Get("VISUAL")
			if editor == "" {
				editor = rt.Env().Get("EDITOR")
			}
			if editor == "" {
				editor = "vi"
			}

			editorCmd := exec.CommandContext(cmd.Context(), editor, result.Path)
			editorCmd.Stdin = rt.Stream().In
			editorCmd.Stdout = rt.Stream().Out
			editorCmd.Stderr = rt.Stream().Err
			return editorCmd.Run()
		},
	}

	cmd.Flags().BoolVar(&user, "user", false, "user config (~/.config/siphon/config.yaml)")
	cmd.Flags().BoolVar(&project, "project", false, "project config (.siphon/config.yaml)")
	cmd.Flags().BoolVar(&local, "local", false, "local config (.siphon/config.local.yaml, gitignored)")
	cmd.MarkFlagsMutuallyExclusive("user", "project", "local")

	return cmd
}
