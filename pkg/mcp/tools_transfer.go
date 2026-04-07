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
	Source        string   `json:"source" jsonschema:"source CONN:DATABASE"`
	Target        string   `json:"target" jsonschema:"target CONN:DATABASE"`
	Tables        []string `json:"tables,omitempty" jsonschema:"explicit table list"`
	Groups        []string `json:"groups,omitempty" jsonschema:"named table groups to include"`
	ExcludeGroups []string `json:"exclude_groups,omitempty" jsonschema:"named table groups to exclude"`
	AllTables     bool     `json:"all_tables,omitempty" jsonschema:"transfer all tables"`
	OnConflict    string   `json:"on_conflict,omitempty" jsonschema:"conflict strategy: skip, overwrite, merge"`
	Confirm       bool     `json:"confirm,omitempty" jsonschema:"confirm destructive operation"`
}

func registerTransfer(srv *sdkmcp.Server, s *siphon.Siphon) {
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "transfer",
		Description: "Transfer tables between databases using CONN:DATABASE syntax. Tables are transferred in FK dependency order.",
		Annotations: &sdkmcp.ToolAnnotations{
			DestructiveHint: boolPtr(true),
		},
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in transferInput) (*sdkmcp.CallToolResult, any, error) {
		err := s.Transfer(ctx, &siphon.TransferOptions{
			Source:        in.Source,
			Target:        in.Target,
			Tables:        in.Tables,
			Groups:        in.Groups,
			ExcludeGroups: in.ExcludeGroups,
			AllTables:     in.AllTables,
			OnConflict:    in.OnConflict,
			Surface:       siphon.SurfaceMCP,
			Confirm:       in.Confirm,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult("transfer completed"), nil, nil
	})
}
