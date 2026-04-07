// Package parity_test provides table-driven tests that verify CLI commands and
// MCP tools produce equivalent results when calling the same underlying Siphon
// API methods against the same sandboxed state.
//
// The tests exercise both surfaces through their real execution paths:
//   - CLI: pkg/cli.Run -> Cobra -> pkg/siphon.Siphon
//   - MCP: in-memory MCP client -> pkg/mcp.Server -> pkg/siphon.Siphon
//
// Both surfaces share the same sandbox, runtime, and Siphon instance per test case.
package parity_test

import (
	"context"
	"embed"
	"errors"
	"strings"
	"testing"

	"github.com/jlrickert/cli-toolkit/sandbox"
	"github.com/jlrickert/cli-toolkit/toolkit"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/jlrickert/siphon/pkg/cli"
	"github.com/jlrickert/siphon/pkg/mcp"
	"github.com/jlrickert/siphon/pkg/siphon"
)

//go:embed all:data/**
var testdata embed.FS

// --- test harness ---

// parityEnv holds a sandboxed environment with both a CLI runner and an MCP
// session ready to use against the same siphon state.
type parityEnv struct {
	t       *testing.T
	sb      *sandbox.Sandbox
	siphon  *siphon.Siphon
	session *sdkmcp.ClientSession
	ctx     context.Context
}

// newParityEnv creates a shared test environment.
func newParityEnv(t *testing.T) *parityEnv {
	t.Helper()

	sb := sandbox.NewSandbox(t, &sandbox.Options{
		Data: testdata,
		Home: "/home/testuser",
		User: "testuser",
	}, sandbox.WithFixture("testuser", "~"))

	rt := sb.Runtime()
	ctx := sb.Context()

	s, err := siphon.New(siphon.SiphonOptions{
		Runtime: rt,
	})
	require.NoError(t, err)

	// Set up MCP server with in-memory transport.
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
		Name:    "parity-test",
		Version: "0.1",
	}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		session.Close()
	})

	return &parityEnv{
		t:       t,
		sb:      sb,
		siphon:  s,
		session: session,
		ctx:     ctx,
	}
}

// runCLI executes a CLI command and returns stdout as a string.
func (e *parityEnv) runCLI(args ...string) (string, error) {
	e.t.Helper()
	proc := sandbox.NewProcess(func(ctx context.Context, rt *toolkit.Runtime) (int, error) {
		return cli.Run(ctx, rt, args)
	}, false)
	result := proc.Run(e.ctx, e.sb.Runtime())
	if result.Err != nil {
		return strings.TrimSpace(string(result.Stdout)), result.Err
	}
	return strings.TrimSpace(string(result.Stdout)), nil
}

// runMCP calls an MCP tool and returns the text content as a string.
func (e *parityEnv) runMCP(toolName string, args map[string]any) (string, error) {
	e.t.Helper()
	res, err := e.session.CallTool(e.ctx, &sdkmcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	})
	if err != nil {
		return "", err
	}
	text := extractText(e.t, res)
	if res.IsError {
		return text, &mcpError{msg: text}
	}
	return strings.TrimSpace(text), nil
}

type mcpError struct {
	msg string
}

func (e *mcpError) Error() string { return e.msg }

func extractText(t *testing.T, res *sdkmcp.CallToolResult) string {
	t.Helper()
	var parts []string
	for _, c := range res.Content {
		if tc, ok := c.(*sdkmcp.TextContent); ok {
			parts = append(parts, tc.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// --- ParityTestCase ---

// ParityTestCase defines a single parity test: given the same state, CLI
// and MCP should produce equivalent results.
type ParityTestCase struct {
	Name string

	// CLI invocation.
	CLIArgs []string

	// MCP invocation.
	MCPTool  string
	MCPInput map[string]any

	// Compare defines how to compare CLI and MCP output.
	Compare func(t *testing.T, cliOut, mcpOut string)

	SkipCLI bool
	SkipMCP bool

	WantErr         bool
	WantErrContains string
}

func runParityTests(t *testing.T, cases []ParityTestCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			env := newParityEnv(t)

			var cliOut string
			var cliErr error
			if !tc.SkipCLI {
				cliOut, cliErr = env.runCLI(tc.CLIArgs...)
			}

			var mcpOut string
			var mcpErr error
			if !tc.SkipMCP {
				mcpOut, mcpErr = env.runMCP(tc.MCPTool, tc.MCPInput)
			}

			if tc.WantErr {
				if !tc.SkipCLI {
					require.Error(t, cliErr, "CLI should have returned an error")
				}
				if !tc.SkipMCP {
					require.Error(t, mcpErr, "MCP should have returned an error")
				}
				if tc.WantErrContains != "" {
					if !tc.SkipCLI {
						require.Contains(t, cliErr.Error(), tc.WantErrContains)
					}
					if !tc.SkipMCP {
						require.Contains(t, mcpErr.Error(), tc.WantErrContains)
					}
				}
				return
			}

			if !tc.SkipCLI {
				require.NoError(t, cliErr, "CLI failed: %s", cliOut)
			}
			if !tc.SkipMCP {
				require.NoError(t, mcpErr, "MCP failed: %s", mcpOut)
			}

			if !tc.SkipCLI && !tc.SkipMCP && tc.Compare != nil {
				tc.Compare(t, cliOut, mcpOut)
			}
		})
	}
}
