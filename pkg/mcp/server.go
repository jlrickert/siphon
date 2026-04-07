package mcp

import (
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jlrickert/siphon/pkg/siphon"
)

// ServerOptions holds configuration for creating an MCP server.
type ServerOptions struct {
	LicenseText string
}

// NewServer builds an MCP server with all registered tools.
func NewServer(s *siphon.Siphon, version string, opts ...ServerOptions) *sdkmcp.Server {
	srv := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    "siphon",
		Version: version,
	}, nil)

	registerConnectionsTools(srv, s)
	registerBackupTools(srv, s)
	registerRestoreTools(srv, s)
	registerTransferTools(srv, s)
	registerSQLTools(srv, s)
	registerScheduleTools(srv, s)
	registerConfigTools(srv, s)

	return srv
}

// textResult wraps a string in a CallToolResult.
func textResult(text string) *sdkmcp.CallToolResult {
	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{
			&sdkmcp.TextContent{Text: text},
		},
	}
}

// errorResult returns a CallToolResult with IsError set.
func errorResult(err error) *sdkmcp.CallToolResult {
	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{
			&sdkmcp.TextContent{Text: err.Error()},
		},
		IsError: true,
	}
}

// boolPtr returns a pointer to b.
func boolPtr(b bool) *bool { return &b }
