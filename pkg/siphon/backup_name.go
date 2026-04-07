package siphon

import (
	"bytes"
	"fmt"
	"text/template"
	"time"
)

// DefaultBackupNameTemplate is the default Go template used to generate
// backup directory names when no custom template is configured.
const DefaultBackupNameTemplate = "{{.Connection}}-{{.Date}}-{{.Time}}"

// BackupNameData provides the variables available to backup name templates.
type BackupNameData struct {
	Connection string
	Database   string
	Timestamp  time.Time
	Date       string // YYYY-MM-DD
	Time       string // HH-MM-SS
	Engine     string
	Type       string // "logical", "physical", "file"
}

// NewBackupNameData builds a BackupNameData from the given parameters,
// populating derived fields (Date, Time) from the timestamp.
func NewBackupNameData(connection, database, engineStr, backupType string, ts time.Time) BackupNameData {
	return BackupNameData{
		Connection: connection,
		Database:   database,
		Timestamp:  ts,
		Date:       ts.Format("2006-01-02"),
		Time:       ts.Format("15-04-05"),
		Engine:     engineStr,
		Type:       backupType,
	}
}

// ResolveBackupName evaluates a Go text/template string against the given
// BackupNameData and returns the resulting backup directory name.
func ResolveBackupName(tmpl string, data BackupNameData) (string, error) {
	if tmpl == "" {
		tmpl = DefaultBackupNameTemplate
	}
	t, err := template.New("backup_name").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("parsing backup name template: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing backup name template: %w", err)
	}
	result := buf.String()
	if result == "" {
		return "", fmt.Errorf("backup name template produced empty result")
	}
	return result, nil
}
