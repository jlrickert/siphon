package mcp

import (
	"context"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jlrickert/siphon/pkg/siphon"
)

func registerRestoreTools(srv *sdkmcp.Server, s *siphon.Siphon) {
	registerRestore(srv, s)
}

type restoreInput struct {
	Connection string   `json:"connection" jsonschema:"target connection name"`
	Path       string   `json:"path" jsonschema:"backup path or @repo/name"`
	BackupID   string   `json:"backup_id,omitempty" jsonschema:"backup identifier (alias for path)"`
	Force      bool     `json:"force,omitempty" jsonschema:"required for destructive restore"`
	Database   string   `json:"database,omitempty" jsonschema:"override target database name"`
	Tables     []string `json:"tables,omitempty" jsonschema:"specific tables for partial restore"`
	Type       string   `json:"type,omitempty" jsonschema:"explicit backup type (physical, logical, file)"`
	Confirm    bool     `json:"confirm,omitempty" jsonschema:"confirm destructive operation (required for MCP)"`
}

func registerRestore(srv *sdkmcp.Server, s *siphon.Siphon) {
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "restore",
		Description: "Restore a database from a backup. Destructive operation requiring force and confirm flags.",
		Annotations: &sdkmcp.ToolAnnotations{
			DestructiveHint: boolPtr(true),
		},
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in restoreInput) (*sdkmcp.CallToolResult, any, error) {
		err := s.Restore(ctx, &siphon.RestoreOptions{
			Connection: in.Connection,
			Path:       in.Path,
			BackupID:   in.BackupID,
			Force:      in.Force,
			Database:   in.Database,
			Tables:     in.Tables,
			Type:       in.Type,
			Surface:    siphon.SurfaceMCP,
			Confirm:    in.Confirm,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult("restore completed"), nil, nil
	})
}
