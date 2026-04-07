package siphon

import (
	"context"
	"fmt"

	"github.com/jlrickert/siphon/pkg/engine"
)

// transferImpl implements the Transfer service method.
func (s *Siphon) transferImpl(ctx context.Context, opts *TransferOptions) error {
	cfg, err := s.ConfigService.Config(true)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// 1. Parse colon syntax for source and target.
	srcConn, srcDB := ParseColonSyntax(opts.Source)
	tgtConn, tgtDB := ParseColonSyntax(opts.Target)

	if srcConn == "" {
		return fmt.Errorf("source connection is required")
	}
	if tgtConn == "" {
		return fmt.Errorf("target connection is required")
	}

	// 2. Resolve both connections from config.
	srcCfg, ok := cfg.Connections[srcConn]
	if !ok {
		return fmt.Errorf("source %w: %s", ErrConnectionNotFound, srcConn)
	}
	tgtCfg, ok := cfg.Connections[tgtConn]
	if !ok {
		return fmt.Errorf("target %w: %s", ErrConnectionNotFound, tgtConn)
	}

	// Fill in default database from connection config if not specified.
	if srcDB == "" && srcCfg.Database != nil {
		srcDB = *srcCfg.Database
	}
	if tgtDB == "" && tgtCfg.Database != nil {
		tgtDB = *tgtCfg.Database
	}

	// 3. Check policy for source (read) and target (write).
	surface := opts.Surface
	if surface == "" {
		surface = SurfaceCLI
	}

	// Source: at minimum readonly access is needed.
	srcAction := ResolvePolicy(cfg.Policies, srcConn, surface)
	if srcAction == PolicyDeny {
		return fmt.Errorf("source: %w", ErrOperationDenied)
	}

	// Target: write access is needed; readonly is insufficient.
	tgtAction := ResolvePolicy(cfg.Policies, tgtConn, surface)
	switch tgtAction {
	case PolicyDeny:
		return fmt.Errorf("target: %w", ErrOperationDenied)
	case PolicyReadonly:
		return fmt.Errorf("target: %w", ErrOperationDenied)
	case PolicyConfirm:
		if surface == SurfaceMCP && !opts.Confirm {
			return ErrConfirmationRequired
		}
	case PolicyAllow:
		// proceed
	}

	// Check per-connection allow_transfer policy.
	if tgtPolicy, ok := cfg.Policies[tgtConn]; ok && tgtPolicy != nil {
		if tgtPolicy.AllowTransfer != nil && !*tgtPolicy.AllowTransfer {
			return fmt.Errorf("target: %w", ErrOperationDenied)
		}
	}

	// 4. Create adaptors for both, connect.
	factory := s.AdaptorFactory
	if factory == nil {
		factory = engine.NewAdaptor
	}

	srcAdaptor, err := factory(srcCfg)
	if err != nil {
		return fmt.Errorf("creating source adaptor: %w", err)
	}
	defer srcAdaptor.Close()

	tgtAdaptor, err := factory(tgtCfg)
	if err != nil {
		return fmt.Errorf("creating target adaptor: %w", err)
	}
	defer tgtAdaptor.Close()

	if err := srcAdaptor.Connect(ctx); err != nil {
		return fmt.Errorf("connecting to source: %w", err)
	}
	if err := tgtAdaptor.Connect(ctx); err != nil {
		return fmt.Errorf("connecting to target: %w", err)
	}

	// 5. Check source implements TransferAdaptor for table discovery.
	srcTransfer, ok := engine.HasTransfer(srcAdaptor)
	if !ok {
		return fmt.Errorf("source %w: engine %s does not support transfer",
			ErrUnsupportedCapability, srcAdaptor.Engine())
	}

	tgtTransfer, ok := engine.HasTransfer(tgtAdaptor)
	if !ok {
		return fmt.Errorf("target %w: engine %s does not support transfer",
			ErrUnsupportedCapability, tgtAdaptor.Engine())
	}

	// 6. Discover tables from source.
	allTables, err := srcTransfer.ListTables(ctx, srcDB)
	if err != nil {
		return fmt.Errorf("listing source tables: %w", err)
	}

	// 7. Resolve table selection (groups, excludes, explicit tables, all-tables).
	tableSelection := TableSelectionOptions{
		AllTables:     opts.AllTables,
		Tables:        opts.Tables,
		Groups:        opts.Groups,
		ExcludeGroups: opts.ExcludeGroups,
	}
	selectedTables, err := ResolveTableSelection(tableSelection, cfg.TableGroups, allTables)
	if err != nil {
		return fmt.Errorf("resolving table selection: %w", err)
	}

	if len(selectedTables) == 0 {
		return fmt.Errorf("%w: no tables selected for transfer", ErrTransferFailed)
	}

	// 8. Get FK dependencies, topological sort.
	fks, err := srcTransfer.GetForeignKeys(ctx, srcDB)
	if err != nil {
		return fmt.Errorf("getting foreign keys: %w", err)
	}

	// Build dependency list only for selected tables.
	selectedSet := make(map[string]bool)
	for _, t := range selectedTables {
		selectedSet[t] = true
	}

	depMap := make(map[string][]string)
	for _, t := range selectedTables {
		depMap[t] = nil // ensure all selected tables appear
	}
	for _, fk := range fks {
		if selectedSet[fk.Table] && selectedSet[fk.ReferencedTable] {
			depMap[fk.Table] = append(depMap[fk.Table], fk.ReferencedTable)
		}
	}

	var tableDeps []TableDependency
	for table, deps := range depMap {
		tableDeps = append(tableDeps, TableDependency{
			Table:     table,
			DependsOn: deps,
		})
	}

	orderedTables, err := TopoSort(tableDeps)
	if err != nil {
		return fmt.Errorf("sorting tables: %w", err)
	}

	// 9. Resolve on-conflict strategy.
	onConflict := opts.OnConflict
	if onConflict == "" {
		onConflict = "skip"
	}
	switch onConflict {
	case "skip", "overwrite", "merge":
		// valid
	default:
		return fmt.Errorf("%w: unknown on-conflict strategy: %s", ErrTransferFailed, onConflict)
	}

	// 10. Execute transfer via target adaptor.
	// The engine-level TransferTables handles schema and data transfer.
	transferOpts := engine.TransferOptions{
		Source: engine.DatabaseTarget{
			Connection: srcConn,
			Database:   srcDB,
		},
		Destination: engine.DatabaseTarget{
			Connection: tgtConn,
			Database:   tgtDB,
		},
		Tables:     orderedTables,
		OnConflict: onConflict,
	}

	if err := tgtTransfer.TransferTables(ctx, transferOpts); err != nil {
		return fmt.Errorf("%w: %v", ErrTransferFailed, err)
	}

	return nil
}
