package siphon

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jlrickert/siphon/pkg/engine"
)

// restoreImpl implements the Restore service method.
func (s *Siphon) restoreImpl(ctx context.Context, opts *RestoreOptions) error {
	cfg, err := s.ConfigService.Config(true)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// 1. Resolve backup path.
	backupPath := opts.Path
	if backupPath == "" {
		backupPath = opts.BackupID
	}
	if backupPath == "" {
		return fmt.Errorf("backup path is required")
	}

	if strings.HasPrefix(backupPath, "@") {
		resolved, err := ResolveRepoPath(cfg, backupPath)
		if err != nil {
			return fmt.Errorf("resolving path: %w", err)
		}
		backupPath = resolved
	}

	// 2. Resolve connection.
	connName := opts.Connection
	if connName == "" {
		if cfg.DefaultConnection != nil {
			connName = *cfg.DefaultConnection
		}
	}
	if connName == "" {
		return fmt.Errorf("connection name is required")
	}

	cc, ok := cfg.Connections[connName]
	if !ok {
		return ErrConnectionNotFound
	}

	// 3. Check policy -- restore is destructive.
	surface := opts.Surface
	if surface == "" {
		surface = SurfaceCLI
	}
	action := ResolvePolicy(cfg.Policies, connName, surface)
	switch action {
	case PolicyDeny:
		return ErrOperationDenied
	case PolicyReadonly:
		return ErrOperationDenied
	case PolicyConfirm:
		if surface == SurfaceMCP && !opts.Confirm {
			return ErrConfirmationRequired
		}
	case PolicyAllow:
		// proceed
	}

	// Also check the per-connection allow_restore policy.
	if connPolicy, ok := cfg.Policies[connName]; ok && connPolicy != nil {
		if connPolicy.AllowRestore != nil && !*connPolicy.AllowRestore {
			return ErrOperationDenied
		}
	}

	// 4. Enforce --force flag.
	if !opts.Force {
		return ErrForceRequired
	}

	// 5. Read manifest from backup path.
	manifestPath := filepath.Join(backupPath, ManifestFilename)
	manifest, err := ReadManifest(s.Runtime, manifestPath)

	var backupType string
	if err != nil {
		// No manifest found.
		if opts.Type == "" {
			return fmt.Errorf("%w: %v", ErrManifestNotFound, err)
		}
		// Use explicit --type override.
		backupType = opts.Type
	} else {
		// Manifest found -- discover type from it.
		backupType = manifest.Type
		if opts.Type != "" {
			// Validate explicit type against manifest if both present.
			if opts.Type != manifest.Type {
				return fmt.Errorf("%w: backup is %q but --type is %q",
					ErrBackupTypeMismatch, manifest.Type, opts.Type)
			}
		}
	}

	// 6. Create adaptor via factory.
	factory := s.AdaptorFactory
	if factory == nil {
		factory = engine.NewAdaptor
	}
	adaptor, err := factory(cc)
	if err != nil {
		return fmt.Errorf("creating adaptor: %w", err)
	}
	defer adaptor.Close()

	// 7. Connect.
	if err := adaptor.Connect(ctx); err != nil {
		return fmt.Errorf("connecting: %w", err)
	}

	// 8. Validate backup type matches engine capability.
	if err := checkRestoreCapability(adaptor, backupType); err != nil {
		return err
	}

	// 9. Reject partial restore (--tables) of physical backup.
	if backupType == "physical" && len(opts.Tables) > 0 {
		return fmt.Errorf("%w: partial restore (--tables) is not supported for physical backups",
			ErrUnsupportedRestoreTarget)
	}

	// 10. Determine target database.
	database := opts.Database
	if database == "" && manifest != nil {
		database = manifest.Database
	}
	if database == "" && cc.Database != nil {
		database = *cc.Database
	}

	// 11. Handle decompression and execute restore.
	switch backupType {
	case "physical":
		pb, _ := engine.HasPhysicalBackup(adaptor)

		// Prepare the backup.
		if err := pb.Prepare(ctx, engine.PrepareOptions{
			TargetDir: backupPath,
		}); err != nil {
			return fmt.Errorf("%w: prepare: %v", ErrRestoreFailed, err)
		}

		// Copy back to data directory.
		dataDir := ""
		if cc.Path != nil {
			dataDir = *cc.Path
		}
		if err := pb.CopyBack(ctx, engine.CopyBackOptions{
			SourceDir: backupPath,
			DataDir:   dataDir,
		}); err != nil {
			return fmt.Errorf("%w: copy-back: %v", ErrRestoreFailed, err)
		}

	case "logical":
		lb, _ := engine.HasLogicalBackup(adaptor)

		// Find the dump file.
		dumpFile, err := findDumpFile(s, backupPath, manifest)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrRestoreFailed, err)
		}

		// Read dump data.
		dumpData, err := s.Runtime.ReadFile(dumpFile)
		if err != nil {
			return fmt.Errorf("%w: reading dump file: %v", ErrRestoreFailed, err)
		}

		// Decompress if needed.
		compress := "none"
		if manifest != nil {
			compress = manifest.Compress
		}
		if compress != "" && compress != "none" {
			comp, err := NewCompressor(compress)
			if err != nil {
				return fmt.Errorf("creating decompressor: %w", err)
			}
			var decompressed bytes.Buffer
			if err := comp.Decompress(&decompressed, bytes.NewReader(dumpData)); err != nil {
				return fmt.Errorf("%w: decompressing dump: %v", ErrRestoreFailed, err)
			}
			dumpData = decompressed.Bytes()
		}

		// Load the dump.
		loadOpts := engine.LoadDumpOptions{
			Database: database,
			Input:    bytes.NewReader(dumpData),
		}
		if err := lb.LoadDump(ctx, loadOpts); err != nil {
			return fmt.Errorf("%w: %v", ErrRestoreFailed, err)
		}

	case "file":
		fb, _ := engine.HasFileBackup(adaptor)

		// Find the backup file.
		srcFile, err := findBackupFile(s, backupPath, manifest)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrRestoreFailed, err)
		}

		// Determine destination path.
		destPath := ""
		if cc.Path != nil {
			destPath = *cc.Path
		} else if cc.Database != nil {
			destPath = *cc.Database
		}
		if destPath == "" {
			return fmt.Errorf("%w: no destination path for file restore", ErrRestoreFailed)
		}

		if err := fb.CopyRestore(ctx, engine.CopyRestoreOptions{
			SourcePath: srcFile,
			DestPath:   destPath,
			Force:      opts.Force,
		}); err != nil {
			return fmt.Errorf("%w: %v", ErrRestoreFailed, err)
		}

	default:
		return fmt.Errorf("unknown backup type: %s", backupType)
	}

	return nil
}

// checkRestoreCapability validates that the adaptor supports the requested
// restore type.
func checkRestoreCapability(a engine.Adaptor, backupType string) error {
	switch backupType {
	case "physical":
		if _, ok := engine.HasPhysicalBackup(a); !ok {
			return fmt.Errorf("%w: engine %s does not support physical restore",
				ErrUnsupportedRestoreTarget, a.Engine())
		}
	case "logical":
		if _, ok := engine.HasLogicalBackup(a); !ok {
			return fmt.Errorf("%w: engine %s does not support logical restore",
				ErrUnsupportedRestoreTarget, a.Engine())
		}
	case "file":
		if _, ok := engine.HasFileBackup(a); !ok {
			return fmt.Errorf("%w: engine %s does not support file restore",
				ErrUnsupportedRestoreTarget, a.Engine())
		}
	default:
		return fmt.Errorf("unknown backup type: %s", backupType)
	}
	return nil
}

// findDumpFile locates the SQL dump file within a backup directory.
// If a manifest is available, it uses the first file entry. Otherwise,
// it looks for common dump filenames.
func findDumpFile(s *Siphon, backupDir string, manifest *BackupManifest) (string, error) {
	if manifest != nil && len(manifest.Files) > 0 {
		return filepath.Join(backupDir, manifest.Files[0].Path), nil
	}

	// Fallback: look for common dump filenames.
	candidates := []string{"dump.sql", "dump.sql.zst", "dump.sql.gz"}
	for _, name := range candidates {
		path := filepath.Join(backupDir, name)
		if _, err := s.Runtime.ReadFile(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no dump file found in %s", backupDir)
}

// findBackupFile locates the backup data file within a backup directory
// for file-type backups (e.g., SQLite).
func findBackupFile(s *Siphon, backupDir string, manifest *BackupManifest) (string, error) {
	if manifest != nil && len(manifest.Files) > 0 {
		return filepath.Join(backupDir, manifest.Files[0].Path), nil
	}

	// Fallback: find the first non-manifest file.
	entries, err := s.Runtime.ReadDir(backupDir)
	if err != nil {
		return "", fmt.Errorf("reading backup directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == ManifestFilename {
			continue
		}
		return filepath.Join(backupDir, entry.Name()), nil
	}

	return "", fmt.Errorf("no backup file found in %s", backupDir)
}
