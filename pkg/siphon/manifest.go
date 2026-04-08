package siphon

import (
	"fmt"
	"time"

	"github.com/jlrickert/cli-toolkit/toolkit"
	"gopkg.in/yaml.v3"
)

// ManifestFilename is the conventional name for backup manifest files.
const ManifestFilename = "backup.meta.yaml"

// BackupManifest holds the complete metadata for a single backup, written as
// a YAML file alongside the backup data.
type BackupManifest struct {
	Version    string            `yaml:"version"`
	Engine     string            `yaml:"engine"`
	Type       string            `yaml:"type"` // "physical", "logical", "file"
	Connection string            `yaml:"connection"`
	Database   string            `yaml:"database,omitempty"`
	Timestamp  time.Time         `yaml:"timestamp"`
	Checksum   string            `yaml:"checksum,omitempty"` // SHA-256 of backup data
	Compress   string            `yaml:"compress,omitempty"` // "zstd", "gzip", "none"
	Label      string            `yaml:"label,omitempty"`
	Message    string            `yaml:"message,omitempty"`
	Files      []BackupFileEntry `yaml:"files"`
	LSN        string            `yaml:"lsn,omitempty"` // For incremental mariabackup
	Size       int64             `yaml:"size"`
}

// BackupFileEntry describes a single file within a backup.
type BackupFileEntry struct {
	Path     string `yaml:"path"`
	Size     int64  `yaml:"size"`
	Checksum string `yaml:"checksum"`
}

// WriteManifest serializes a BackupManifest to a YAML file at path using the
// provided runtime for file I/O.
func WriteManifest(rt *toolkit.Runtime, path string, m *BackupManifest) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}
	if err := rt.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing manifest to %s: %w", path, err)
	}
	return nil
}

// ReadManifest deserializes a BackupManifest from a YAML file at path using
// the provided runtime for file I/O.
func ReadManifest(rt *toolkit.Runtime, path string) (*BackupManifest, error) {
	data, err := rt.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest from %s: %w", path, err)
	}
	var m BackupManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest from %s: %w", path, err)
	}
	return &m, nil
}
