package mcp

import (
	"context"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jlrickert/siphon/pkg/siphon"
)

func registerTransferTools(srv *sdkmcp.Server, s *siphon.Siphon) {
	registerTransfer(srv, s)
}

type transferInput struct {
	Source      string   `json:"source" jsonschema:"source CONN:DATABASE"`
	Destination string   `json:"destination" jsonschema:"destination CONN:DATABASE"`
	Tables      []string `json:"tables,omitempty" jsonschema:"tables to transfer (default: all)"`
	Truncate    bool     `json:"truncate,omitempty" jsonschema:"truncate destination tables first"`
	Force       bool     `json:"force,omitempty" jsonschema:"allow destructive transfer"`
	Confirm     bool     `json:"confirm,omitempty" jsonschema:"confirm destructive operation"`
}

func registerTransfer(srv *sdkmcp.Server, s *siphon.Siphon) {
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "transfer",
		Description: "Transfer tables between databases",
		Annotations: &sdkmcp.ToolAnnotations{
			DestructiveHint: boolPtr(true),
		},
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in transferInput) (*sdkmcp.CallToolResult, any, error) {
		err := s.Transfer(ctx, &siphon.TransferOptions{
			Source:      in.Source,
			Destination: in.Destination,
			Tables:      in.Tables,
			Truncate:    in.Truncate,
			Force:       in.Force,
			Confirm:     in.Confirm,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult("transfer completed"), nil, nil
	})
}
