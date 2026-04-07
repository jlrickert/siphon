package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/jlrickert/siphon/pkg/siphon"
	"github.com/spf13/cobra"
)

// NewSQLCmd creates the sql command.
func NewSQLCmd(deps *Deps) *cobra.Command {
	var opts siphon.ExecuteSQLOptions

	cmd := &cobra.Command{
		Use:   "sql CONNECTION [QUERY]",
		Short: "Execute a SQL query",
		Long: `Execute a SQL query against a database connection.

Direct query:
  siphon sql mydb "SELECT * FROM users"

File input:
  siphon sql mydb -f schema.sql

Piped input:
  echo "SELECT 1" | siphon sql mydb

Interactive mode (when no query and TTY):
  siphon sql mydb`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Connection = args[0]
			opts.Surface = siphon.SurfaceCLI

			if len(args) > 1 {
				opts.Query = args[1]
			}

			// If no query and no file, check for piped input or interactive mode.
			if opts.Query == "" && opts.File == "" {
				rt := deps.Runtime
				stream := rt.Stream()

				if stream.IsPiped {
					// Read all of stdin.
					data, err := io.ReadAll(stream.In)
					if err != nil {
						return fmt.Errorf("reading stdin: %w", err)
					}
					opts.Query = strings.TrimSpace(string(data))
					if opts.Query == "" {
						return fmt.Errorf("no SQL input provided")
					}
				} else if stream.IsTTY {
					// Interactive mode.
					return runInteractive(cmd, deps, &opts)
				} else {
					return fmt.Errorf("no query provided; use positional arg, -f flag, or pipe input")
				}
			}

			result, err := deps.Siphon.ExecuteSQL(cmd.Context(), &opts)
			if err != nil {
				return err
			}

			return formatAndPrint(cmd, result, siphon.OutputFormat(opts.Format))
		},
	}

	cmd.Flags().StringVarP(&opts.File, "file", "f", "", "SQL file to execute")
	cmd.Flags().StringVar(&opts.Format, "format", "table", "output format (table, json, csv)")
	cmd.Flags().StringVar(&opts.Database, "database", "", "override database")
	cmd.Flags().BoolVar(&opts.Confirm, "confirm", false, "confirm destructive operation")

	_ = cmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"table", "csv", "json"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

// runInteractive runs a simple line-by-line interactive SQL session.
func runInteractive(cmd *cobra.Command, deps *Deps, opts *siphon.ExecuteSQLOptions) error {
	rt := deps.Runtime
	stream := rt.Stream()
	scanner := bufio.NewScanner(stream.In)

	for {
		// Print prompt to stderr so it doesn't mix with query output.
		_, _ = fmt.Fprint(stream.Err, "siphon> ")

		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "\\q" || line == "exit" || line == "quit" {
			break
		}

		queryOpts := &siphon.ExecuteSQLOptions{
			Connection: opts.Connection,
			Query:      line,
			Format:     opts.Format,
			Database:   opts.Database,
			Surface:    opts.Surface,
			Confirm:    opts.Confirm,
		}

		result, err := deps.Siphon.ExecuteSQL(cmd.Context(), queryOpts)
		if err != nil {
			_, _ = fmt.Fprintf(stream.Err, "Error: %v\n", err)
			continue
		}

		if err := formatAndPrint(cmd, result, siphon.OutputFormat(opts.Format)); err != nil {
			_, _ = fmt.Fprintf(stream.Err, "Error: %v\n", err)
		}
	}

	return scanner.Err()
}

// formatAndPrint writes a QueryResult to the command's stdout using the given format.
func formatAndPrint(cmd *cobra.Command, result *siphon.QueryResult, format siphon.OutputFormat) error {
	formatter, err := siphon.NewFormatter(format)
	if err != nil {
		return err
	}
	return formatter.Format(cmd.OutOrStdout(), result)
}
