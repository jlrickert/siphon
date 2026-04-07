package parity_test

import (
	"encoding/json"
	"testing"

	"github.com/jlrickert/siphon/pkg/siphon"
	"github.com/stretchr/testify/require"
)

func TestParity_ConnectionsList(t *testing.T) {
	t.Parallel()
	runParityTests(t, []ParityTestCase{
		{
			Name:     "list connections returns same data",
			CLIArgs:  []string{"connections", "list"},
			MCPTool:  "connections_list",
			MCPInput: map[string]any{},
			Compare: func(t *testing.T, cliOut, mcpOut string) {
				t.Helper()
				// CLI output is a table, MCP is JSON. Both should
				// contain the "dev" connection.
				require.Contains(t, cliOut, "dev")
				require.Contains(t, mcpOut, "dev")

				// MCP output should be valid JSON.
				var connections []siphon.ConnectionInfo
				err := json.Unmarshal([]byte(mcpOut), &connections)
				require.NoError(t, err)
				require.NotEmpty(t, connections)
			},
		},
	})
}

func TestParity_ConnectionsAdd(t *testing.T) {
	t.Parallel()
	runParityTests(t, []ParityTestCase{
		{
			Name: "add connection via CLI",
			CLIArgs: []string{
				"connections", "add", "cli-conn",
				"--engine", "sqlite",
				"--path", "/tmp/cli-test.db",
			},
			SkipMCP: true,
			Compare: func(t *testing.T, cliOut, mcpOut string) {
				t.Helper()
				require.Contains(t, cliOut, "added")
			},
		},
		{
			Name:    "add connection via MCP",
			SkipCLI: true,
			MCPTool: "connections_add",
			MCPInput: map[string]any{
				"name":   "mcp-conn",
				"engine": "sqlite",
				"path":   "/tmp/mcp-test.db",
			},
			Compare: func(t *testing.T, cliOut, mcpOut string) {
				t.Helper()
				require.Contains(t, mcpOut, "added")
			},
		},
	})
}

func TestParity_ConnectionsRemoveNotFound(t *testing.T) {
	t.Parallel()
	runParityTests(t, []ParityTestCase{
		{
			Name:    "remove nonexistent returns error",
			CLIArgs: []string{"connections", "remove", "nonexistent"},
			MCPTool: "connections_remove",
			MCPInput: map[string]any{
				"name": "nonexistent",
			},
			WantErr:         true,
			WantErrContains: "not found",
		},
	})
}
