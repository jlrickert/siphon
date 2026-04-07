package mcp

import (
	"context"
	"encoding/json"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jlrickert/siphon/pkg/siphon"
)

func registerSQLTools(srv *sdkmcp.Server, s *siphon.Siphon) {
	registerSQLExecute(srv, s)
}

type sqlExecuteInput struct {
	Connection string `json:"connection,omitempty" jsonschema:"target connection"`
	Database   string `json:"database,omitempty" jsonschema:"target database"`
	Query      string `json:"query" jsonschema:"SQL query to execute"`
	Format     string `json:"format,omitempty" jsonschema:"output format (table, json, csv)"`
}

func registerSQLExecute(srv *sdkmcp.Server, s *siphon.Siphon) {
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "sql_execute",
		Description: "Execute a SQL query against a database connection",
		Annotations: &sdkmcp.ToolAnnotations{
			DestructiveHint: boolPtr(true),
			OpenWorldHint:   boolPtr(true),
		},
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in sqlExecuteInput) (*sdkmcp.CallToolResult, any, error) {
		result, err := s.ExecuteSQL(ctx, &siphon.ExecuteSQLOptions{
			Connection: in.Connection,
			Database:   in.Database,
			Query:      in.Query,
			Format:     in.Format,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}
		data, _ := json.Marshal(result)
		return textResult(string(data)), nil, nil
	})
}
