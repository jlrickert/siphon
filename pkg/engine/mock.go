package engine

import (
	"context"
	"database/sql"
	"database/sql/driver"
)

// Compile-time interface checks.
var (
	_ Adaptor               = (*MockAdaptor)(nil)
	_ LogicalBackupAdaptor  = (*MockAdaptor)(nil)
	_ PhysicalBackupAdaptor = (*MockAdaptor)(nil)
	_ FileBackupAdaptor     = (*MockAdaptor)(nil)
	_ TransferAdaptor       = (*MockAdaptor)(nil)
	_ QueryAdaptor          = (*MockAdaptor)(nil)
	_ RawDBAccessor         = (*MockAdaptor)(nil)
)

// MockAdaptor implements all engine capability interfaces for testing.
type MockAdaptor struct {
	EngineType Engine
	Connected  bool
	Databases  []string

	// Hook functions for custom behavior in tests. When nil, the default
	// behavior returns nil error.
	ConnectFn        func(ctx context.Context) error
	PingFn           func(ctx context.Context) error
	CloseFn          func() error
	ListDatabasesFn  func(ctx context.Context) ([]string, error)
	DumpFn           func(ctx context.Context, opts DumpOptions) error
	LoadDumpFn       func(ctx context.Context, opts LoadDumpOptions) error
	PhysicalBackupFn func(ctx context.Context, opts PhysicalBackupOptions) error
	PrepareFn        func(ctx context.Context, opts PrepareOptions) error
	CopyBackFn       func(ctx context.Context, opts CopyBackOptions) error
	CopyBackupFn     func(ctx context.Context, opts CopyBackupOptions) error
	CopyRestoreFn    func(ctx context.Context, opts CopyRestoreOptions) error
	ListTablesFn     func(ctx context.Context, database string) ([]string, error)
	GetForeignKeysFn func(ctx context.Context, database string) ([]ForeignKey, error)
	TransferTablesFn func(ctx context.Context, opts TransferOptions) error

	// Tables holds the configurable table list returned by ListTables.
	Tables []string

	// ForeignKeys holds the configurable FK list returned by GetForeignKeys.
	ForeignKeys []ForeignKey
	ExecuteFn   func(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryFn     func(ctx context.Context, query string, args ...any) (*QueryResult, error)
	RawDBFn     func() *sql.DB
	DB          *sql.DB // configurable raw DB for transfer tests
}

// NewMockAdaptor returns a MockAdaptor with the given engine type.
func NewMockAdaptor(engine Engine) *MockAdaptor {
	return &MockAdaptor{
		EngineType: engine,
		Databases:  []string{"testdb"},
	}
}

func (m *MockAdaptor) Connect(ctx context.Context) error {
	if m.ConnectFn != nil {
		return m.ConnectFn(ctx)
	}
	m.Connected = true
	return nil
}

func (m *MockAdaptor) Ping(ctx context.Context) error {
	if m.PingFn != nil {
		return m.PingFn(ctx)
	}
	return nil
}

func (m *MockAdaptor) Close() error {
	if m.CloseFn != nil {
		return m.CloseFn()
	}
	m.Connected = false
	return nil
}

func (m *MockAdaptor) ListDatabases(ctx context.Context) ([]string, error) {
	if m.ListDatabasesFn != nil {
		return m.ListDatabasesFn(ctx)
	}
	return m.Databases, nil
}

func (m *MockAdaptor) Engine() Engine {
	return m.EngineType
}

func (m *MockAdaptor) Dump(ctx context.Context, opts DumpOptions) error {
	if m.DumpFn != nil {
		return m.DumpFn(ctx, opts)
	}
	return nil
}

func (m *MockAdaptor) LoadDump(ctx context.Context, opts LoadDumpOptions) error {
	if m.LoadDumpFn != nil {
		return m.LoadDumpFn(ctx, opts)
	}
	return nil
}

func (m *MockAdaptor) PhysicalBackup(ctx context.Context, opts PhysicalBackupOptions) error {
	if m.PhysicalBackupFn != nil {
		return m.PhysicalBackupFn(ctx, opts)
	}
	return nil
}

func (m *MockAdaptor) Prepare(ctx context.Context, opts PrepareOptions) error {
	if m.PrepareFn != nil {
		return m.PrepareFn(ctx, opts)
	}
	return nil
}

func (m *MockAdaptor) CopyBack(ctx context.Context, opts CopyBackOptions) error {
	if m.CopyBackFn != nil {
		return m.CopyBackFn(ctx, opts)
	}
	return nil
}

func (m *MockAdaptor) CopyBackup(ctx context.Context, opts CopyBackupOptions) error {
	if m.CopyBackupFn != nil {
		return m.CopyBackupFn(ctx, opts)
	}
	return nil
}

func (m *MockAdaptor) CopyRestore(ctx context.Context, opts CopyRestoreOptions) error {
	if m.CopyRestoreFn != nil {
		return m.CopyRestoreFn(ctx, opts)
	}
	return nil
}

func (m *MockAdaptor) ListTables(ctx context.Context, database string) ([]string, error) {
	if m.ListTablesFn != nil {
		return m.ListTablesFn(ctx, database)
	}
	return m.Tables, nil
}

func (m *MockAdaptor) GetForeignKeys(ctx context.Context, database string) ([]ForeignKey, error) {
	if m.GetForeignKeysFn != nil {
		return m.GetForeignKeysFn(ctx, database)
	}
	return m.ForeignKeys, nil
}

func (m *MockAdaptor) TransferTables(ctx context.Context, opts TransferOptions) error {
	if m.TransferTablesFn != nil {
		return m.TransferTablesFn(ctx, opts)
	}
	return nil
}

func (m *MockAdaptor) Execute(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if m.ExecuteFn != nil {
		return m.ExecuteFn(ctx, query, args...)
	}
	return &mockResult{}, nil
}

func (m *MockAdaptor) Query(ctx context.Context, query string, args ...any) (*QueryResult, error) {
	if m.QueryFn != nil {
		return m.QueryFn(ctx, query, args...)
	}
	return &QueryResult{}, nil
}

func (m *MockAdaptor) RawDB() *sql.DB {
	if m.RawDBFn != nil {
		return m.RawDBFn()
	}
	return m.DB
}

// mockResult implements sql.Result for the mock adaptor.
type mockResult struct{}

func (r *mockResult) LastInsertId() (int64, error) { return 0, nil }
func (r *mockResult) RowsAffected() (int64, error) { return 0, nil }

// Compile-time check that mockResult satisfies driver.Result too.
var _ driver.Result = (*mockResult)(nil)
