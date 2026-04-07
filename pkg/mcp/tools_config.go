package mcp

import (
	"context"
	"encoding/json"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jlrickert/siphon/pkg/siphon"
)

func registerConfigTools(srv *sdkmcp.Server, s *siphon.Siphon) {
	registerConfig(srv, s)
}

type configInput struct {
	Explain bool `json:"explain,omitempty" jsonschema:"show per-field provenance"`
}

func registerConfig(srv *sdkmcp.Server, s *siphon.Siphon) {
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "config",
		Description: "Show resolved siphon configuration",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in configInput) (*sdkmcp.CallToolResult, any, error) {
		cfg, err := s.Config(ctx, &siphon.ConfigOptions{
			Explain: in.Explain,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}
		data, _ := json.Marshal(cfg)
		return textResult(string(data)), nil, nil
	})
}
