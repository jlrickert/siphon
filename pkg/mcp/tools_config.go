package mcp

import (
	"context"
	"encoding/json"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jlrickert/siphon/pkg/siphon"
)

func registerConfigTools(srv *sdkmcp.Server, s *siphon.Siphon) {
	registerConfig(srv, s)
	registerConfigInit(srv, s)
	registerConfigTemplate(srv, s)
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

type configInitInput struct {
	Scope string `json:"scope" jsonschema:"config scope (user, project, local)"`
	Force bool   `json:"force,omitempty" jsonschema:"overwrite if file exists"`
}

func registerConfigInit(srv *sdkmcp.Server, s *siphon.Siphon) {
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "config_init",
		Description: "Create a config file at the specified scope",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:  false,
			OpenWorldHint: boolPtr(false),
		},
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in configInitInput) (*sdkmcp.CallToolResult, any, error) {
		result, err := s.ConfigInit(ctx, &siphon.ConfigInitOptions{
			Scope: siphon.ConfigScope(in.Scope),
			Force: in.Force,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}
		data, _ := json.Marshal(result)
		return textResult(string(data)), nil, nil
	})
}

type configTemplateInput struct{}

func registerConfigTemplate(srv *sdkmcp.Server, s *siphon.Siphon) {
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "config_template",
		Description: "Return an annotated YAML config template",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in configTemplateInput) (*sdkmcp.CallToolResult, any, error) {
		tmpl, err := s.ConfigTemplate(ctx, &siphon.ConfigTemplateOptions{})
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult(tmpl), nil, nil
	})
}
