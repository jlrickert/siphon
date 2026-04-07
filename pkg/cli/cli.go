package cli

import (
	"context"
	"fmt"

	"github.com/jlrickert/cli-toolkit/toolkit"
)

// Run is the main entry point for the siphon CLI. It creates the root command,
// executes it, and returns the exit code.
func Run(ctx context.Context, rt *toolkit.Runtime, args []string) (int, error) {
	if rt == nil {
		var err error
		rt, err = toolkit.NewRuntime()
		if err != nil {
			return 1, err
		}
	}
	if err := rt.Validate(); err != nil {
		return 1, err
	}

	streams := rt.Stream()
	deps := &Deps{
		Runtime: rt,
	}
	cmd := NewRootCmd(deps)
	cmd.SetArgs(args)
	cmd.SetIn(streams.In)
	cmd.SetOut(streams.Out)
	cmd.SetErr(streams.Err)

	execErr := cmd.ExecuteContext(ctx)
	if execErr != nil {
		_, _ = fmt.Fprintf(streams.Err, "Error: %s\n", execErr)
		return 1, execErr
	}
	return 0, nil
}
