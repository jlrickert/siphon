package mcp

import (
	"context"
	"encoding/json"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jlrickert/siphon/pkg/engine"
	"github.com/jlrickert/siphon/pkg/siphon"
)

func registerConnectionsTools(srv *sdkmcp.Server, s *siphon.Siphon) {
	registerConnectionsList(srv, s)
	registerConnectionsAdd(srv, s)
	registerConnectionsRemove(srv, s)
	registerConnectionsTest(srv, s)
}

// --- connections_list ---

type connectionsListInput struct{}

func registerConnectionsList(srv *sdkmcp.Server, s *siphon.Siphon) {
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "connections_list",
		Description: "List all configured database connections",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in connectionsListInput) (*sdkmcp.CallToolResult, any, error) {
		connections, err := s.ListConnections(ctx, &siphon.ListConnectionsOptions{})
		if err != nil {
			return errorResult(err), nil, nil
		}
		data, _ := json.Marshal(connections)
		return textResult(string(data)), nil, nil
	})
}

// --- connections_add ---

type connectionsAddInput struct {
	Name        string        `json:"name" jsonschema:"connection name"`
	Engine      engine.Engine `json:"engine" jsonschema:"database engine (mariadb, postgresql, sqlite)"`
	Host        string        `json:"host,omitempty" jsonschema:"database host"`
	Port        int           `json:"port,omitempty" jsonschema:"database port"`
	User        string        `json:"user,omitempty" jsonschema:"database user"`
	Password    string        `json:"password,omitempty" jsonschema:"database password"`
	PasswordEnv string        `json:"password_env,omitempty" jsonschema:"env var containing password"`
	Database    string        `json:"database,omitempty" jsonschema:"default database"`
	Path        string        `json:"path,omitempty" jsonschema:"SQLite file path"`
}

func registerConnectionsAdd(srv *sdkmcp.Server, s *siphon.Siphon) {
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "connections_add",
		Description: "Add a database connection",
		Annotations: &sdkmcp.ToolAnnotations{
			DestructiveHint: boolPtr(false),
		},
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in connectionsAddInput) (*sdkmcp.CallToolResult, any, error) {
		err := s.AddConnection(ctx, &siphon.AddConnectionOptions{
			Name:        in.Name,
			Engine:      in.Engine,
			Host:        in.Host,
			Port:        in.Port,
			User:        in.User,
			Password:    in.Password,
			PasswordEnv: in.PasswordEnv,
			Database:    in.Database,
			Path:        in.Path,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult("connection added"), nil, nil
	})
}

// --- connections_remove ---

type connectionsRemoveInput struct {
	Name  string `json:"name" jsonschema:"connection name to remove"`
	Force bool   `json:"force,omitempty" jsonschema:"skip confirmation"`
}

func registerConnectionsRemove(srv *sdkmcp.Server, s *siphon.Siphon) {
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "connections_remove",
		Description: "Remove a database connection",
		Annotations: &sdkmcp.ToolAnnotations{
			DestructiveHint: boolPtr(true),
		},
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in connectionsRemoveInput) (*sdkmcp.CallToolResult, any, error) {
		err := s.RemoveConnection(ctx, &siphon.RemoveConnectionOptions{
			Name:  in.Name,
			Force: in.Force,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult("connection removed"), nil, nil
	})
}

// --- connections_test ---

type connectionsTestInput struct {
	Name    string `json:"name" jsonschema:"connection name to test"`
	Timeout string `json:"timeout,omitempty" jsonschema:"timeout duration (e.g. 5s)"`
}

func registerConnectionsTest(srv *sdkmcp.Server, s *siphon.Siphon) {
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "connections_test",
		Description: "Test connectivity to a database connection",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(true),
		},
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in connectionsTestInput) (*sdkmcp.CallToolResult, any, error) {
		var timeout time.Duration
		if in.Timeout != "" {
			var err error
			timeout, err = time.ParseDuration(in.Timeout)
			if err != nil {
				return errorResult(err), nil, nil
			}
		}
		err := s.TestConnection(ctx, &siphon.TestConnectionOptions{
			Name:    in.Name,
			Timeout: timeout,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult("connection OK"), nil, nil
	})
}
