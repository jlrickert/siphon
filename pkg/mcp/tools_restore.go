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
	BackupID   string `json:"backup_id" jsonschema:"backup identifier"`
	Path       string `json:"path,omitempty" jsonschema:"backup path"`
	Connection string `json:"connection,omitempty" jsonschema:"target connection"`
	Database   string `json:"database,omitempty" jsonschema:"target database"`
	Force      bool   `json:"force,omitempty" jsonschema:"allow destructive restore"`
	Confirm    bool   `json:"confirm,omitempty" jsonschema:"confirm destructive operation"`
}

func registerRestore(srv *sdkmcp.Server, s *siphon.Siphon) {
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "restore",
		Description: "Restore a database from a backup",
		Annotations: &sdkmcp.ToolAnnotations{
			DestructiveHint: boolPtr(true),
		},
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in restoreInput) (*sdkmcp.CallToolResult, any, error) {
		err := s.Restore(ctx, &siphon.RestoreOptions{
			BackupID:   in.BackupID,
			Path:       in.Path,
			Connection: in.Connection,
			Database:   in.Database,
			Force:      in.Force,
			Confirm:    in.Confirm,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult("restore completed"), nil, nil
	})
}
