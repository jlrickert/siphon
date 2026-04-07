package parity_test

import (
	"encoding/json"
	"testing"

	"github.com/jlrickert/siphon/pkg/siphon"
	"github.com/stretchr/testify/require"
)

func TestParity_ScheduleList_Empty(t *testing.T) {
	t.Parallel()
	runParityTests(t, []ParityTestCase{
		{
			Name:     "list schedules returns empty when none configured",
			CLIArgs:  []string{"schedule", "list"},
			MCPTool:  "schedule_list",
			MCPInput: map[string]any{},
			Compare: func(t *testing.T, cliOut, mcpOut string) {
				t.Helper()
				// CLI says "No schedules configured."
				require.Contains(t, cliOut, "No schedules")
				// MCP returns empty JSON array.
				var schedules []siphon.ScheduleInfo
				err := json.Unmarshal([]byte(mcpOut), &schedules)
				require.NoError(t, err)
				require.Empty(t, schedules)
			},
		},
	})
}

func TestParity_ScheduleCreateRemove(t *testing.T) {
	t.Parallel()

	t.Run("create via CLI then list via MCP", func(t *testing.T) {
		t.Parallel()
		env := newParityEnv(t)

		// Create schedule via CLI.
		cliOut, cliErr := env.runCLI(
			"schedule", "create", "nightly-parity",
			"--connection", "dev",
			"--time", "03:00",
			"--interval", "daily",
			"--launchd",
		)
		require.NoError(t, cliErr, "CLI create failed: %s", cliOut)
		require.Contains(t, cliOut, "nightly-parity")

		// List via MCP and verify the schedule exists.
		mcpOut, mcpErr := env.runMCP("schedule_list", map[string]any{})
		require.NoError(t, mcpErr, "MCP list failed: %s", mcpOut)

		var schedules []siphon.ScheduleInfo
		err := json.Unmarshal([]byte(mcpOut), &schedules)
		require.NoError(t, err)
		require.Len(t, schedules, 1)
		require.Equal(t, "nightly-parity", schedules[0].Name)
		require.Equal(t, "dev", schedules[0].Connection)
	})

	t.Run("create via MCP then remove via CLI", func(t *testing.T) {
		t.Parallel()
		env := newParityEnv(t)

		// Create via MCP.
		mcpOut, mcpErr := env.runMCP("schedule_create", map[string]any{
			"name":       "mcp-sched",
			"connection": "dev",
			"time":       "04:00",
			"interval":   "weekly",
			"backend":    "launchd",
		})
		require.NoError(t, mcpErr, "MCP create failed: %s", mcpOut)

		// Remove via CLI.
		cliOut, cliErr := env.runCLI("schedule", "remove", "mcp-sched")
		require.NoError(t, cliErr, "CLI remove failed: %s", cliOut)
		require.Contains(t, cliOut, "removed")

		// Verify it's gone via MCP list.
		mcpOut, mcpErr = env.runMCP("schedule_list", map[string]any{})
		require.NoError(t, mcpErr)
		var schedules []siphon.ScheduleInfo
		err := json.Unmarshal([]byte(mcpOut), &schedules)
		require.NoError(t, err)
		require.Empty(t, schedules)
	})
}

func TestParity_ScheduleCreate_ConnectionNotFound(t *testing.T) {
	t.Parallel()
	runParityTests(t, []ParityTestCase{
		{
			Name: "create schedule with nonexistent connection",
			CLIArgs: []string{
				"schedule", "create", "bad",
				"--connection", "nonexistent",
			},
			MCPTool: "schedule_create",
			MCPInput: map[string]any{
				"name":       "bad",
				"connection": "nonexistent",
			},
			WantErr:         true,
			WantErrContains: "connection not found",
		},
	})
}

func TestParity_ScheduleRemove_NotFound(t *testing.T) {
	t.Parallel()
	runParityTests(t, []ParityTestCase{
		{
			Name:    "remove nonexistent schedule",
			CLIArgs: []string{"schedule", "remove", "ghost"},
			MCPTool: "schedule_remove",
			MCPInput: map[string]any{
				"name": "ghost",
			},
			WantErr:         true,
			WantErrContains: "schedule not found",
		},
	})
}
