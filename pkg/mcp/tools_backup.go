package mcp

import (
	"context"
	"encoding/json"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jlrickert/siphon/pkg/siphon"
)

func registerBackupTools(srv *sdkmcp.Server, s *siphon.Siphon) {
	registerBackupCreate(srv, s)
	registerBackupInfo(srv, s)
	registerBackupList(srv, s)
	registerBackupVerify(srv, s)
	registerBackupLabel(srv, s)
}

// --- backup_create ---

type backupCreateInput struct {
	Connection string   `json:"connection" jsonschema:"connection name"`
	Database   string   `json:"database,omitempty" jsonschema:"database name"`
	BackupType string   `json:"backup_type,omitempty" jsonschema:"backup type (logical, physical, file)"`
	Compress   string   `json:"compress,omitempty" jsonschema:"compression (zstd, gzip, none)"`
	Name       string   `json:"name,omitempty" jsonschema:"explicit backup name"`
	Repo       string   `json:"repo,omitempty" jsonschema:"backup repository"`
	Label      string   `json:"label,omitempty" jsonschema:"backup label"`
	Message    string   `json:"message,omitempty" jsonschema:"backup message"`
	Tables     []string `json:"tables,omitempty" jsonschema:"specific tables to back up"`
	Force      bool     `json:"force,omitempty" jsonschema:"overwrite existing backup"`
}

func registerBackupCreate(srv *sdkmcp.Server, s *siphon.Siphon) {
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "backup_create",
		Description: "Create a database backup",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint: false,
		},
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in backupCreateInput) (*sdkmcp.CallToolResult, any, error) {
		desc, err := s.Backup(ctx, &siphon.BackupOptions{
			Connection: in.Connection,
			Database:   in.Database,
			BackupType: in.BackupType,
			Compress:   in.Compress,
			Name:       in.Name,
			Repo:       in.Repo,
			Label:      in.Label,
			Message:    in.Message,
			Tables:     in.Tables,
			Force:      in.Force,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}
		data, _ := json.Marshal(desc)
		return textResult(string(data)), nil, nil
	})
}

// --- backup_info ---

type backupInfoInput struct {
	BackupID string `json:"backup_id,omitempty" jsonschema:"backup identifier"`
	Path     string `json:"path,omitempty" jsonschema:"backup path"`
}

func registerBackupInfo(srv *sdkmcp.Server, s *siphon.Siphon) {
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "backup_info",
		Description: "Show backup metadata",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in backupInfoInput) (*sdkmcp.CallToolResult, any, error) {
		desc, err := s.BackupInfo(ctx, &siphon.BackupInfoOptions{
			BackupID: in.BackupID,
			Path:     in.Path,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}
		data, _ := json.Marshal(desc)
		return textResult(string(data)), nil, nil
	})
}

// --- backup_list ---

type backupListInput struct {
	Connection string `json:"connection,omitempty" jsonschema:"filter by connection"`
	Database   string `json:"database,omitempty" jsonschema:"filter by database"`
	Repo       string `json:"repo,omitempty" jsonschema:"filter by repository"`
	Limit      int    `json:"limit,omitempty" jsonschema:"maximum results"`
}

func registerBackupList(srv *sdkmcp.Server, s *siphon.Siphon) {
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "backup_list",
		Description: "List available backups",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in backupListInput) (*sdkmcp.CallToolResult, any, error) {
		backups, err := s.ListBackups(ctx, &siphon.ListBackupsOptions{
			Connection: in.Connection,
			Database:   in.Database,
			Repo:       in.Repo,
			Limit:      in.Limit,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}
		data, _ := json.Marshal(backups)
		return textResult(string(data)), nil, nil
	})
}

// --- backup_verify ---

type backupVerifyInput struct {
	BackupID string `json:"backup_id,omitempty" jsonschema:"backup identifier"`
	Path     string `json:"path,omitempty" jsonschema:"backup path"`
}

func registerBackupVerify(srv *sdkmcp.Server, s *siphon.Siphon) {
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "backup_verify",
		Description: "Verify backup integrity",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in backupVerifyInput) (*sdkmcp.CallToolResult, any, error) {
		err := s.VerifyBackup(ctx, &siphon.VerifyBackupOptions{
			BackupID: in.BackupID,
			Path:     in.Path,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult("backup verified"), nil, nil
	})
}

// --- backup_label ---

type backupLabelInput struct {
	BackupID string `json:"backup_id" jsonschema:"backup identifier"`
	Label    string `json:"label" jsonschema:"label to assign"`
}

func registerBackupLabel(srv *sdkmcp.Server, s *siphon.Siphon) {
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "backup_label",
		Description: "Set or update a backup label",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint: false,
		},
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in backupLabelInput) (*sdkmcp.CallToolResult, any, error) {
		err := s.LabelBackup(ctx, &siphon.LabelBackupOptions{
			BackupID: in.BackupID,
			Label:    in.Label,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult("label updated"), nil, nil
	})
}
