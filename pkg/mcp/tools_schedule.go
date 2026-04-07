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
	Name             string `json:"name" jsonschema:"schedule name"`
	Connection       string `json:"connection" jsonschema:"target connection"`
	Repo             string `json:"repo,omitempty" jsonschema:"backup repository"`
	Database         string `json:"database,omitempty" jsonschema:"target database"`
	BackupType       string `json:"backup_type,omitempty" jsonschema:"backup type (logical, physical, file)"`
	Compress         string `json:"compress,omitempty" jsonschema:"compression (zstd, gzip, none)"`
	Message          string `json:"message,omitempty" jsonschema:"default message for backups"`
	BackupNameFormat string `json:"backup_name_format,omitempty" jsonschema:"name template override"`
	Time             string `json:"time,omitempty" jsonschema:"schedule time HH:MM"`
	Interval         string `json:"interval,omitempty" jsonschema:"interval: daily, hourly, weekly"`
	Backend          string `json:"backend,omitempty" jsonschema:"backend: launchd or cron"`
	KeepCount        int    `json:"keep_count,omitempty" jsonschema:"retention: keep N most recent"`
	KeepAge          string `json:"keep_age,omitempty" jsonschema:"retention: keep backups newer than duration"`
}

func registerScheduleCreate(srv *sdkmcp.Server, s *siphon.Siphon) {
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "schedule_create",
		Description: "Create a backup schedule",
		Annotations: &sdkmcp.ToolAnnotations{
			DestructiveHint: boolPtr(false),
		},
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in scheduleCreateInput) (*sdkmcp.CallToolResult, any, error) {
		opts := &siphon.CreateScheduleOptions{
			Name:             in.Name,
			Connection:       in.Connection,
			Repo:             in.Repo,
			Database:         in.Database,
			BackupType:       in.BackupType,
			Compress:         in.Compress,
			Message:          in.Message,
			BackupNameFormat: in.BackupNameFormat,
			Time:             in.Time,
			Interval:         in.Interval,
			Backend:          in.Backend,
			Surface:          siphon.SurfaceMCP,
		}

		if in.KeepCount > 0 || in.KeepAge != "" {
			rp := &siphon.RetentionPolicy{}
			if in.KeepCount > 0 {
				rp.KeepCount = &in.KeepCount
			}
			if in.KeepAge != "" {
				rp.KeepAge = &in.KeepAge
			}
			opts.Retention = rp
		}

		err := s.CreateSchedule(ctx, opts)
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult("schedule created"), nil, nil
	})
}

// --- schedule_list ---

type scheduleListInput struct{}

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
			Surface: siphon.SurfaceMCP,
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
			Name:    in.Name,
			Surface: siphon.SurfaceMCP,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult("schedule removed"), nil, nil
	})
}
