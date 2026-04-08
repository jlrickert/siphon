package siphon

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jlrickert/siphon/pkg/engine"
)

// backupImpl implements the Backup service method. It replaces the
// ErrNotImplemented stub.
func (s *Siphon) backupImpl(ctx context.Context, opts *BackupOptions) (*BackupDescriptor, error) {
	cfg, err := s.ConfigService.Config(true)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	// 1. Resolve connection.
	cc, ok := cfg.Connections[opts.Connection]
	if !ok {
		return nil, ErrConnectionNotFound
	}

	// 2. Create adaptor via factory.
	adaptor, err := s.createAdaptor(cc)
	if err != nil {
		return nil, fmt.Errorf("creating adaptor: %w", err)
	}
	defer adaptor.Close()

	// 3. Connect.
	if err := adaptor.Connect(ctx); err != nil {
		return nil, fmt.Errorf("connecting: %w", err)
	}

	// 4. Determine backup type.
	backupType := opts.BackupType
	if backupType == "" {
		backupType = defaultBackupType(adaptor)
	}

	// 5. Check capability.
	if err := checkBackupCapability(adaptor, backupType); err != nil {
		return nil, err
	}

	// 6. Resolve backup name.
	now := s.Runtime.Clock().Now()
	nameData := NewBackupNameData(
		opts.Connection,
		opts.Database,
		string(adaptor.Engine()),
		backupType,
		now,
	)
	tmplStr := DefaultBackupNameTemplate
	if cfg.BackupNameFormat != nil && *cfg.BackupNameFormat != "" {
		tmplStr = *cfg.BackupNameFormat
	}
	if opts.Name != "" {
		// Explicit name overrides template.
		tmplStr = opts.Name
	}
	backupName, err := ResolveBackupName(tmplStr, nameData)
	if err != nil {
		return nil, fmt.Errorf("resolving backup name: %w", err)
	}

	// 7. Create target directory.
	var backupDir string
	if opts.Repo != "" {
		repoPath, err := ResolveRepoPath(cfg, "@"+opts.Repo, s.home())
		if err != nil {
			return nil, fmt.Errorf("resolving repo: %w", err)
		}
		backupDir = filepath.Join(repoPath, backupName)
	} else {
		// Default: current working directory based.
		wd, err := s.Runtime.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting working directory: %w", err)
		}
		backupDir = filepath.Join(wd, backupName)
	}

	if err := s.Runtime.Mkdir(backupDir, 0o755, true); err != nil {
		return nil, fmt.Errorf("creating backup directory: %w", err)
	}

	// 8. Execute backup through capability interface.
	var files []BackupFileEntry
	database := opts.Database
	if database == "" && cc.Database != nil {
		database = *cc.Database
	}

	switch backupType {
	case "physical":
		pb, _ := engine.HasPhysicalBackup(adaptor)
		if err := pb.PhysicalBackup(ctx, engine.PhysicalBackupOptions{
			TargetDir: backupDir,
		}); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBackupFailed, err)
		}
		// Walk the backup directory to collect files.
		files, err = collectBackupFiles(s, backupDir)
		if err != nil {
			return nil, fmt.Errorf("collecting backup files: %w", err)
		}

	case "logical":
		lb, _ := engine.HasLogicalBackup(adaptor)
		dumpFile := filepath.Join(backupDir, "dump.sql")
		var buf bytes.Buffer
		if err := lb.Dump(ctx, engine.DumpOptions{
			Database: database,
			Output:   &buf,
		}); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBackupFailed, err)
		}

		dumpData := buf.Bytes()

		// Apply compression if requested.
		compress := opts.Compress
		if compress == "" {
			compress = "none"
		}
		comp, err := NewCompressor(compress)
		if err != nil {
			return nil, fmt.Errorf("creating compressor: %w", err)
		}
		if compress != "none" && compress != "" {
			dumpFile = dumpFile + comp.Extension()
			var compressed bytes.Buffer
			if err := comp.Compress(&compressed, bytes.NewReader(dumpData)); err != nil {
				return nil, fmt.Errorf("compressing dump: %w", err)
			}
			dumpData = compressed.Bytes()
		}

		if err := s.Runtime.WriteFile(dumpFile, dumpData, 0o644); err != nil {
			return nil, fmt.Errorf("writing dump file: %w", err)
		}

		checksum := sha256Hex(dumpData)
		files = []BackupFileEntry{{
			Path:     filepath.Base(dumpFile),
			Size:     int64(len(dumpData)),
			Checksum: checksum,
		}}

	case "file":
		fb, _ := engine.HasFileBackup(adaptor)
		srcPath := ""
		if cc.Path != nil {
			srcPath = *cc.Path
		} else if cc.Database != nil {
			srcPath = *cc.Database
		}
		destFile := filepath.Join(backupDir, filepath.Base(srcPath))
		if err := fb.CopyBackup(ctx, engine.CopyBackupOptions{
			SourcePath: srcPath,
			DestPath:   destFile,
		}); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBackupFailed, err)
		}

		fileData, err := s.Runtime.ReadFile(destFile)
		if err != nil {
			return nil, fmt.Errorf("reading backup file: %w", err)
		}
		checksum := sha256Hex(fileData)
		files = []BackupFileEntry{{
			Path:     filepath.Base(destFile),
			Size:     int64(len(fileData)),
			Checksum: checksum,
		}}
	}

	// 9. Calculate total size.
	var totalSize int64
	for _, f := range files {
		totalSize += f.Size
	}

	// 10. Compute overall checksum (of all file checksums concatenated).
	overallChecksum := computeOverallChecksum(files)

	// 11. Write manifest.
	compress := opts.Compress
	if compress == "" {
		compress = "none"
	}
	manifest := &BackupManifest{
		Version:    "1",
		Engine:     string(adaptor.Engine()),
		Type:       backupType,
		Connection: opts.Connection,
		Database:   database,
		Timestamp:  now,
		Checksum:   overallChecksum,
		Compress:   compress,
		Label:      opts.Label,
		Message:    opts.Message,
		Files:      files,
		Size:       totalSize,
	}

	manifestPath := filepath.Join(backupDir, ManifestFilename)
	if err := WriteManifest(s.Runtime, manifestPath, manifest); err != nil {
		return nil, fmt.Errorf("writing manifest: %w", err)
	}

	// 12. Build and return descriptor.
	desc := manifestToDescriptor(manifest, backupDir)
	return desc, nil
}

// backupInfoImpl implements BackupInfo.
func (s *Siphon) backupInfoImpl(ctx context.Context, opts *BackupInfoOptions) (*BackupDescriptor, error) {
	cfg, err := s.ConfigService.Config(true)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	backupPath := opts.Path
	if backupPath == "" {
		backupPath = opts.BackupID
	}

	// Resolve @repo syntax.
	if strings.HasPrefix(backupPath, "@") {
		resolved, err := ResolveRepoPath(cfg, backupPath, s.home())
		if err != nil {
			return nil, fmt.Errorf("resolving path: %w", err)
		}
		backupPath = resolved
	}

	manifestPath := filepath.Join(backupPath, ManifestFilename)
	manifest, err := ReadManifest(s.Runtime, manifestPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrManifestNotFound, err)
	}

	return manifestToDescriptor(manifest, backupPath), nil
}

// listBackupsImpl implements ListBackups.
func (s *Siphon) listBackupsImpl(ctx context.Context, opts *ListBackupsOptions) ([]BackupDescriptor, error) {
	cfg, err := s.ConfigService.Config(true)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	// Determine base directory to search.
	searchDir := "."
	if opts.Repo != "" {
		resolved, err := ResolveRepoPath(cfg, "@"+opts.Repo, s.home())
		if err != nil {
			return nil, fmt.Errorf("resolving repo: %w", err)
		}
		searchDir = resolved
	}

	var descriptors []BackupDescriptor

	// Walk directory looking for backup.meta.yaml files in immediate
	// subdirectories (backup directories are one level deep).
	entries, err := s.Runtime.ReadDir(searchDir)
	if err != nil {
		if os.IsNotExist(err) {
			return descriptors, nil
		}
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(searchDir, entry.Name(), ManifestFilename)
		manifest, err := ReadManifest(s.Runtime, manifestPath)
		if err != nil {
			continue // skip directories without manifests
		}

		// Apply filters.
		if opts.Connection != "" && manifest.Connection != opts.Connection {
			continue
		}
		if opts.Database != "" && manifest.Database != opts.Database {
			continue
		}

		backupDir := filepath.Join(searchDir, entry.Name())
		desc := manifestToDescriptor(manifest, backupDir)
		descriptors = append(descriptors, *desc)
	}

	// Sort by timestamp descending (newest first).
	sort.Slice(descriptors, func(i, j int) bool {
		return descriptors[i].Timestamp.After(descriptors[j].Timestamp)
	})

	// Apply limit.
	if opts.Limit > 0 && len(descriptors) > opts.Limit {
		descriptors = descriptors[:opts.Limit]
	}

	return descriptors, nil
}

// verifyBackupImpl implements VerifyBackup.
func (s *Siphon) verifyBackupImpl(ctx context.Context, opts *VerifyBackupOptions) error {
	cfg, err := s.ConfigService.Config(true)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	backupPath := opts.Path
	if backupPath == "" {
		backupPath = opts.BackupID
	}

	if strings.HasPrefix(backupPath, "@") {
		resolved, err := ResolveRepoPath(cfg, backupPath, s.home())
		if err != nil {
			return fmt.Errorf("resolving path: %w", err)
		}
		backupPath = resolved
	}

	manifestPath := filepath.Join(backupPath, ManifestFilename)
	manifest, err := ReadManifest(s.Runtime, manifestPath)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrManifestNotFound, err)
	}

	// Verify each file exists and checksum matches.
	for _, f := range manifest.Files {
		filePath := filepath.Join(backupPath, f.Path)
		data, err := s.Runtime.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("backup file missing: %s: %w", f.Path, err)
		}
		actual := sha256Hex(data)
		if actual != f.Checksum {
			return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", f.Path, f.Checksum, actual)
		}
		if int64(len(data)) != f.Size {
			return fmt.Errorf("size mismatch for %s: expected %d, got %d", f.Path, f.Size, len(data))
		}
	}

	return nil
}

// labelBackupImpl implements LabelBackup.
func (s *Siphon) labelBackupImpl(ctx context.Context, opts *LabelBackupOptions) error {
	cfg, err := s.ConfigService.Config(true)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	backupPath := opts.BackupID
	if strings.HasPrefix(backupPath, "@") {
		resolved, err := ResolveRepoPath(cfg, backupPath, s.home())
		if err != nil {
			return fmt.Errorf("resolving path: %w", err)
		}
		backupPath = resolved
	}

	manifestPath := filepath.Join(backupPath, ManifestFilename)
	manifest, err := ReadManifest(s.Runtime, manifestPath)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrManifestNotFound, err)
	}

	manifest.Label = opts.Label

	if err := WriteManifest(s.Runtime, manifestPath, manifest); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	return nil
}

// --- helpers ---

// defaultBackupType returns the default backup type for an engine.
func defaultBackupType(a engine.Adaptor) string {
	switch a.Engine() {
	case engine.EngineSQLite:
		return "file"
	default:
		return "logical"
	}
}

// checkBackupCapability validates that the adaptor supports the requested
// backup type.
func checkBackupCapability(a engine.Adaptor, backupType string) error {
	switch backupType {
	case "physical":
		if _, ok := engine.HasPhysicalBackup(a); !ok {
			return fmt.Errorf("%w: engine %s does not support physical backup", ErrUnsupportedCapability, a.Engine())
		}
	case "logical":
		if _, ok := engine.HasLogicalBackup(a); !ok {
			return fmt.Errorf("%w: engine %s does not support logical backup", ErrUnsupportedCapability, a.Engine())
		}
	case "file":
		if _, ok := engine.HasFileBackup(a); !ok {
			return fmt.Errorf("%w: engine %s does not support file backup", ErrUnsupportedCapability, a.Engine())
		}
	default:
		return fmt.Errorf("unknown backup type: %s", backupType)
	}
	return nil
}

// sha256Hex returns the hex-encoded SHA-256 hash of data.
func sha256Hex(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// computeOverallChecksum computes a checksum from the concatenation of all
// file checksums.
func computeOverallChecksum(files []BackupFileEntry) string {
	h := sha256.New()
	for _, f := range files {
		h.Write([]byte(f.Checksum))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// collectBackupFiles reads a directory and returns BackupFileEntry values for
// all regular files. It uses the runtime for reading file data.
func collectBackupFiles(s *Siphon, dir string) ([]BackupFileEntry, error) {
	dirEntries, err := s.Runtime.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var entries []BackupFileEntry
	for _, d := range dirEntries {
		if d.IsDir() {
			continue
		}
		// Skip the manifest file itself.
		if d.Name() == ManifestFilename {
			continue
		}
		filePath := filepath.Join(dir, d.Name())
		data, err := s.Runtime.ReadFile(filePath)
		if err != nil {
			continue
		}
		info, err := d.Info()
		if err != nil {
			continue
		}
		entries = append(entries, BackupFileEntry{
			Path:     d.Name(),
			Size:     info.Size(),
			Checksum: sha256Hex(data),
		})
	}
	return entries, nil
}

// manifestToDescriptor converts a BackupManifest into a BackupDescriptor.
func manifestToDescriptor(m *BackupManifest, backupDir string) *BackupDescriptor {
	var filePaths []string
	for _, f := range m.Files {
		filePaths = append(filePaths, filepath.Join(backupDir, f.Path))
	}
	return &BackupDescriptor{
		ID:         filepath.Base(backupDir),
		Engine:     engine.Engine(m.Engine),
		BackupType: m.Type,
		Connection: m.Connection,
		Database:   m.Database,
		Timestamp:  m.Timestamp,
		Size:       m.Size,
		Checksum:   m.Checksum,
		Label:      m.Label,
		FilePaths:  filePaths,
	}
}
