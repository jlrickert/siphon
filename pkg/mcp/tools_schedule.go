package mcp

import (
	"context"
	"encoding/json"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jlrickert/siphon/pkg/siphon"
)

func registerScheduleTools(srv *sdkmcp.Server, s *siphon.Siphon) {
	registerScheduleCreate(srv, s)
	registerScheduleList(srv, s)
	registerScheduleRemove(srv, s)
}

// --- schedule_create ---

type scheduleCreateInput struct {
	Name       string `json:"name" jsonschema:"schedule name"`
	Connection string `json:"connection" jsonschema:"target connection"`
	Database   string `json:"database,omitempty" jsonschema:"target database"`
	Cron       string `json:"cron" jsonschema:"cron expression"`
	Repo       string `json:"repo,omitempty" jsonschema:"backup repository"`
	BackupType string `json:"backup_type,omitempty" jsonschema:"backup type (logical, physical, file)"`
	Retain     int    `json:"retain,omitempty" jsonschema:"number of backups to retain"`
}

func registerScheduleCreate(srv *sdkmcp.Server, s *siphon.Siphon) {
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "schedule_create",
		Description: "Create a backup schedule",
		Annotations: &sdkmcp.ToolAnnotations{
			DestructiveHint: boolPtr(false),
		},
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in scheduleCreateInput) (*sdkmcp.CallToolResult, any, error) {
		err := s.CreateSchedule(ctx, &siphon.CreateScheduleOptions{
			Name:       in.Name,
			Connection: in.Connection,
			Database:   in.Database,
			Cron:       in.Cron,
			Repo:       in.Repo,
			BackupType: in.BackupType,
			Retain:     in.Retain,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult("schedule created"), nil, nil
	})
}

// --- schedule_list ---

type scheduleListInput struct {
	Connection string `json:"connection,omitempty" jsonschema:"filter by connection"`
}

func registerScheduleList(srv *sdkmcp.Server, s *siphon.Siphon) {
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "schedule_list",
		Description: "List backup schedules",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in scheduleListInput) (*sdkmcp.CallToolResult, any, error) {
		schedules, err := s.ListSchedules(ctx, &siphon.ListSchedulesOptions{
			Connection: in.Connection,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}
		data, _ := json.Marshal(schedules)
		return textResult(string(data)), nil, nil
	})
}

// --- schedule_remove ---

type scheduleRemoveInput struct {
	Name string `json:"name" jsonschema:"schedule name to remove"`
}

func registerScheduleRemove(srv *sdkmcp.Server, s *siphon.Siphon) {
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "schedule_remove",
		Description: "Remove a backup schedule",
		Annotations: &sdkmcp.ToolAnnotations{
			DestructiveHint: boolPtr(true),
		},
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in scheduleRemoveInput) (*sdkmcp.CallToolResult, any, error) {
		err := s.RemoveSchedule(ctx, &siphon.RemoveScheduleOptions{
			Name: in.Name,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult("schedule removed"), nil, nil
	})
}
