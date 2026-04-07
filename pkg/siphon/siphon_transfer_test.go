package siphon

import (
	"context"
	"testing"

	"github.com/jlrickert/cli-toolkit/sandbox"
	"github.com/jlrickert/siphon/pkg/engine"
	"github.com/stretchr/testify/require"
)

// newTransferTestSiphon creates a sandboxed Siphon with a mock factory that
// supports transfer operations. The mock returns configurable table lists and
// records TransferTables calls.
func newTransferTestSiphon(t *testing.T, tables []string, fks []engine.ForeignKey) (*Siphon, *sandbox.Sandbox, *transferRecord) {
	t.Helper()

	sb := sandbox.NewSandbox(t, nil)
	rt := sb.Runtime()

	// Write a config with two connections.
	configDir := "/home/testuser/.config/siphon"
	sb.Mkdir(configDir, true)
	sb.WriteFile(configDir+"/config.yaml", []byte(`
version: "1"
connections:
  source:
    name: source
    engine: mariadb
    host: localhost
    database: srcdb
  target:
    name: target
    engine: mariadb
    host: localhost
    database: tgtdb
table_groups:
  core:
    tables:
      - users
      - orders
  logs:
    pattern: "^log_"
`), 0o644)

	s, err := New(SiphonOptions{Runtime: rt})
	require.NoError(t, err)

	rec := &transferRecord{}

	s.AdaptorFactory = func(cfg *engine.ConnectionConfig) (engine.Adaptor, error) {
		mock := engine.NewMockAdaptor(cfg.Engine)
		mock.Tables = tables
		mock.ForeignKeys = fks
		mock.TransferTablesFn = func(ctx context.Context, opts engine.TransferOptions) error {
			rec.called = true
			rec.opts = opts
			return nil
		}
		return mock, nil
	}

	return s, sb, rec
}

type transferRecord struct {
	called bool
	opts   engine.TransferOptions
}

func TestTransfer_BasicTransfer(t *testing.T) {
	t.Parallel()
	tables := []string{"users", "orders"}
	s, _, rec := newTransferTestSiphon(t, tables, nil)

	err := s.Transfer(context.Background(), &TransferOptions{
		Source: "source:srcdb",
		Target: "target:tgtdb",
	})
	require.NoError(t, err)
	require.True(t, rec.called)
	require.ElementsMatch(t, tables, rec.opts.Tables)
	require.Equal(t, "skip", rec.opts.OnConflict)
}

func TestTransfer_ExplicitTables(t *testing.T) {
	t.Parallel()
	tables := []string{"users", "orders", "products"}
	s, _, rec := newTransferTestSiphon(t, tables, nil)

	err := s.Transfer(context.Background(), &TransferOptions{
		Source: "source:srcdb",
		Target: "target:tgtdb",
		Tables: []string{"users"},
	})
	require.NoError(t, err)
	require.True(t, rec.called)
	require.Equal(t, []string{"users"}, rec.opts.Tables)
}

func TestTransfer_OnConflictOverwrite(t *testing.T) {
	t.Parallel()
	tables := []string{"users"}
	s, _, rec := newTransferTestSiphon(t, tables, nil)

	err := s.Transfer(context.Background(), &TransferOptions{
		Source:     "source:srcdb",
		Target:     "target:tgtdb",
		OnConflict: "overwrite",
	})
	require.NoError(t, err)
	require.Equal(t, "overwrite", rec.opts.OnConflict)
}

func TestTransfer_OnConflictInvalid(t *testing.T) {
	t.Parallel()
	tables := []string{"users"}
	s, _, _ := newTransferTestSiphon(t, tables, nil)

	err := s.Transfer(context.Background(), &TransferOptions{
		Source:     "source:srcdb",
		Target:     "target:tgtdb",
		OnConflict: "invalid",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown on-conflict strategy")
}

func TestTransfer_ColonSyntaxParsing(t *testing.T) {
	t.Parallel()
	tables := []string{"users"}
	s, _, rec := newTransferTestSiphon(t, tables, nil)

	err := s.Transfer(context.Background(), &TransferOptions{
		Source: "source:customdb",
		Target: "target:otherdb",
	})
	require.NoError(t, err)
	require.Equal(t, "customdb", rec.opts.Source.Database)
	require.Equal(t, "otherdb", rec.opts.Destination.Database)
}

func TestTransfer_ColonSyntaxDefaultDB(t *testing.T) {
	t.Parallel()
	tables := []string{"users"}
	s, _, rec := newTransferTestSiphon(t, tables, nil)

	// No database in colon syntax -- should use connection default.
	err := s.Transfer(context.Background(), &TransferOptions{
		Source: "source",
		Target: "target",
	})
	require.NoError(t, err)
	require.Equal(t, "srcdb", rec.opts.Source.Database)
	require.Equal(t, "tgtdb", rec.opts.Destination.Database)
}

func TestTransfer_TableGroupSelection(t *testing.T) {
	t.Parallel()
	tables := []string{"users", "orders", "products", "log_access", "log_error"}
	s, _, rec := newTransferTestSiphon(t, tables, nil)

	err := s.Transfer(context.Background(), &TransferOptions{
		Source: "source:srcdb",
		Target: "target:tgtdb",
		Groups: []string{"core"},
	})
	require.NoError(t, err)
	require.True(t, rec.called)
	require.ElementsMatch(t, []string{"users", "orders"}, rec.opts.Tables)
}

func TestTransfer_FKOrdering(t *testing.T) {
	t.Parallel()
	tables := []string{"users", "orders", "order_items"}
	fks := []engine.ForeignKey{
		{Table: "orders", ReferencedTable: "users"},
		{Table: "order_items", ReferencedTable: "orders"},
	}
	s, _, rec := newTransferTestSiphon(t, tables, fks)

	err := s.Transfer(context.Background(), &TransferOptions{
		Source: "source:srcdb",
		Target: "target:tgtdb",
	})
	require.NoError(t, err)

	// Verify ordering: users before orders, orders before order_items.
	indexOf := make(map[string]int)
	for i, table := range rec.opts.Tables {
		indexOf[table] = i
	}
	require.Less(t, indexOf["users"], indexOf["orders"])
	require.Less(t, indexOf["orders"], indexOf["order_items"])
}

func TestTransfer_SourceConnectionNotFound(t *testing.T) {
	t.Parallel()
	s, _, _ := newTransferTestSiphon(t, nil, nil)

	err := s.Transfer(context.Background(), &TransferOptions{
		Source: "nonexistent:db",
		Target: "target:tgtdb",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrConnectionNotFound)
}

func TestTransfer_TargetConnectionNotFound(t *testing.T) {
	t.Parallel()
	s, _, _ := newTransferTestSiphon(t, nil, nil)

	err := s.Transfer(context.Background(), &TransferOptions{
		Source: "source:srcdb",
		Target: "nonexistent:db",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrConnectionNotFound)
}

func TestTransfer_EmptySource(t *testing.T) {
	t.Parallel()
	s, _, _ := newTransferTestSiphon(t, nil, nil)

	err := s.Transfer(context.Background(), &TransferOptions{
		Source: "",
		Target: "target:tgtdb",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "source connection is required")
}

func TestTransfer_PolicyDenyTarget(t *testing.T) {
	t.Parallel()
	tables := []string{"users"}
	s, sb, _ := newTransferTestSiphon(t, tables, nil)

	// Overwrite config with a policy that denies the target.
	configDir := "/home/testuser/.config/siphon"
	sb.WriteFile(configDir+"/config.yaml", []byte(`
version: "1"
connections:
  source:
    name: source
    engine: mariadb
    host: localhost
    database: srcdb
  target:
    name: target
    engine: mariadb
    host: localhost
    database: tgtdb
policies:
  target:
    cli: deny
`), 0o644)

	// Reset config cache to pick up the new config.
	s.ConfigService.ResetCache()

	err := s.Transfer(context.Background(), &TransferOptions{
		Source:  "source:srcdb",
		Target:  "target:tgtdb",
		Surface: SurfaceCLI,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrOperationDenied)
}

func TestTransfer_MCPRequiresConfirm(t *testing.T) {
	t.Parallel()
	tables := []string{"users"}
	s, sb, _ := newTransferTestSiphon(t, tables, nil)

	// Config with confirm policy for MCP surface on target.
	configDir := "/home/testuser/.config/siphon"
	sb.WriteFile(configDir+"/config.yaml", []byte(`
version: "1"
connections:
  source:
    name: source
    engine: mariadb
    host: localhost
    database: srcdb
  target:
    name: target
    engine: mariadb
    host: localhost
    database: tgtdb
policies:
  target:
    mcp: confirm
`), 0o644)
	s.ConfigService.ResetCache()

	err := s.Transfer(context.Background(), &TransferOptions{
		Source:  "source:srcdb",
		Target:  "target:tgtdb",
		Surface: SurfaceMCP,
		Confirm: false,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrConfirmationRequired)

	// With confirm = true, should succeed.
	err = s.Transfer(context.Background(), &TransferOptions{
		Source:  "source:srcdb",
		Target:  "target:tgtdb",
		Surface: SurfaceMCP,
		Confirm: true,
	})
	require.NoError(t, err)
}
