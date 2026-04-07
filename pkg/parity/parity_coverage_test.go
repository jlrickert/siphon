package parity_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/jlrickert/cli-toolkit/sandbox"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/jlrickert/siphon/pkg/cli"
	"github.com/jlrickert/siphon/pkg/mcp"
	"github.com/jlrickert/siphon/pkg/siphon"
)

// siphonMethodToSurfaces maps exported Siphon method names to their expected
// CLI command and MCP tool name. This is the ground truth for surface coverage.
var siphonMethodToSurfaces = map[string]struct {
	CLI string // CLI command path (e.g., "connections list")
	MCP string // MCP tool name (e.g., "connections_list")
}{
	// Connection management
	"ListConnections":  {CLI: "connections list", MCP: "connections_list"},
	"AddConnection":    {CLI: "connections add", MCP: "connections_add"},
	"RemoveConnection": {CLI: "connections remove", MCP: "connections_remove"},
	"TestConnection":   {CLI: "connections test", MCP: "connections_test"},

	// Backup operations
	"Backup":       {CLI: "backup create", MCP: "backup_create"},
	"BackupInfo":   {CLI: "backup info", MCP: "backup_info"},
	"ListBackups":  {CLI: "backup list", MCP: "backup_list"},
	"VerifyBackup": {CLI: "backup verify", MCP: "backup_verify"},
	"LabelBackup":  {CLI: "backup label", MCP: "backup_label"},

	// Restore
	"Restore": {CLI: "restore", MCP: "restore"},

	// Transfer
	"Transfer": {CLI: "transfer", MCP: "transfer"},

	// SQL
	"ExecuteSQL": {CLI: "sql", MCP: "sql_execute"},

	// Schedules
	"CreateSchedule": {CLI: "schedule create", MCP: "schedule_create"},
	"ListSchedules":  {CLI: "schedule list", MCP: "schedule_list"},
	"RemoveSchedule": {CLI: "schedule remove", MCP: "schedule_remove"},

	// Config
	"Config": {CLI: "config", MCP: "config"},
}

// siphonMethodsExcluded lists Siphon methods that are intentionally excluded
// from surface coverage checks.
var siphonMethodsExcluded = map[string]string{
	// Internal service accessors, not user-facing operations.
}

// TestCoverage_AllSiphonMethodsHaveBothSurfaces uses reflection to enumerate
// exported methods on *siphon.Siphon and verify that each method listed in
// siphonMethodToSurfaces has both a registered CLI command and MCP tool.
func TestCoverage_AllSiphonMethodsHaveBothSurfaces(t *testing.T) {
	t.Parallel()

	// Enumerate exported Siphon methods via reflection.
	siphonType := reflect.TypeOf(&siphon.Siphon{})
	siphonMethods := make(map[string]bool)
	for i := 0; i < siphonType.NumMethod(); i++ {
		m := siphonType.Method(i)
		if m.IsExported() {
			siphonMethods[m.Name] = true
		}
	}

	// Verify every mapped method actually exists on Siphon.
	for method := range siphonMethodToSurfaces {
		require.True(t, siphonMethods[method],
			"siphonMethodToSurfaces references %q but it is not an exported method on *siphon.Siphon", method)
	}

	// Collect MCP tool names from a live server.
	mcpTools := collectMCPToolNames(t)

	// Collect CLI command names from a live command tree.
	cliCommands := collectCLICommandPaths(t)

	// Check each mapped method has both surfaces.
	for method, surfaces := range siphonMethodToSurfaces {
		t.Run("surface/"+method, func(t *testing.T) {
			require.Contains(t, mcpTools, surfaces.MCP,
				"Siphon.%s is mapped to MCP tool %q but tool is not registered", method, surfaces.MCP)

			require.Contains(t, cliCommands, surfaces.CLI,
				"Siphon.%s is mapped to CLI command %q but command is not registered", method, surfaces.CLI)
		})
	}

	// Report any Siphon methods that are neither mapped nor excluded.
	for method := range siphonMethods {
		if _, mapped := siphonMethodToSurfaces[method]; mapped {
			continue
		}
		if _, excluded := siphonMethodsExcluded[method]; excluded {
			continue
		}
		t.Errorf("Siphon.%s is not mapped in siphonMethodToSurfaces and not in siphonMethodsExcluded — add it to one of them", method)
	}
}

// collectMCPToolNames returns the set of registered MCP tool names.
func collectMCPToolNames(t *testing.T) map[string]bool {
	t.Helper()

	sb := sandbox.NewSandbox(t, &sandbox.Options{
		Data: testdata,
		Home: "/home/testuser",
		User: "testuser",
	}, sandbox.WithFixture("testuser", "~"))
	ctx := sb.Context()

	s, err := siphon.New(siphon.SiphonOptions{
		Runtime: sb.Runtime(),
	})
	require.NoError(t, err)

	srv := mcp.NewServer(s, "test")
	serverTransport, clientTransport := sdkmcp.NewInMemoryTransports()

	done := make(chan error, 1)
	go func() {
		done <- srv.Run(ctx, serverTransport)
	}()
	t.Cleanup(func() {
		if err := <-done; err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("MCP server error: %v", err)
		}
	})

	client := sdkmcp.NewClient(&sdkmcp.Implementation{
		Name:    "coverage-test",
		Version: "0.1",
	}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { session.Close() })

	res, err := session.ListTools(ctx, nil)
	require.NoError(t, err)

	tools := make(map[string]bool)
	for _, tool := range res.Tools {
		tools[tool.Name] = true
	}
	return tools
}

// collectCLICommandPaths returns the set of CLI command paths by walking the
// Cobra command tree dynamically.
func collectCLICommandPaths(t *testing.T) map[string]bool {
	t.Helper()

	root := cli.NewRootCmd(&cli.Deps{})

	paths := make(map[string]bool)
	walkCommands(root, "", paths)
	return paths
}

// walkCommands recursively walks a Cobra command tree, collecting command paths
// relative to the root.
func walkCommands(cmd *cobra.Command, prefix string, paths map[string]bool) {
	for _, child := range cmd.Commands() {
		path := child.Name()
		if prefix != "" {
			path = prefix + " " + child.Name()
		}
		paths[path] = true
		walkCommands(child, path, paths)
	}
}
