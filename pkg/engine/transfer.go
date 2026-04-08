package engine

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// transferTablesSQL implements row-level data transfer between two *sql.DB
// connections. The dialect parameter controls conflict-handling syntax.
func transferTablesSQL(ctx context.Context, srcDB, dstDB *sql.DB, opts TransferOptions, dialect string) error {
	for _, table := range opts.Tables {
		if err := transferOneTable(ctx, srcDB, dstDB, table, opts.OnConflict, dialect); err != nil {
			return fmt.Errorf("%w: table %s: %v", ErrTransferFailed, table, err)
		}
	}
	return nil
}

func transferOneTable(ctx context.Context, srcDB, dstDB *sql.DB, table, onConflict, dialect string) error {
	// Read all rows from source.
	rows, err := srcDB.QueryContext(ctx, "SELECT * FROM "+table)
	if err != nil {
		return fmt.Errorf("reading source: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("reading columns: %v", err)
	}
	if len(columns) == 0 {
		return nil
	}

	// Build INSERT statement with conflict handling.
	insertSQL := buildInsertSQL(table, columns, onConflict, dialect)

	// Prepare the insert statement for batch efficiency.
	stmt, err := dstDB.PrepareContext(ctx, insertSQL)
	if err != nil {
		return fmt.Errorf("preparing insert: %v", err)
	}
	defer stmt.Close()

	// Scan and insert row by row.
	values := make([]any, len(columns))
	ptrs := make([]any, len(columns))
	for i := range values {
		ptrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return fmt.Errorf("scanning row: %v", err)
		}
		if _, err := stmt.ExecContext(ctx, values...); err != nil {
			return fmt.Errorf("inserting row: %v", err)
		}
	}
	return rows.Err()
}

func buildInsertSQL(table string, columns []string, onConflict, dialect string) string {
	colList := strings.Join(columns, ", ")
	placeholders := make([]string, len(columns))

	switch dialect {
	case "postgresql":
		for i := range placeholders {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
		}
	default: // sqlite, mysql/mariadb
		for i := range placeholders {
			placeholders[i] = "?"
		}
	}
	valList := strings.Join(placeholders, ", ")

	switch dialect {
	case "sqlite":
		switch onConflict {
		case "overwrite":
			return fmt.Sprintf("INSERT OR REPLACE INTO %s (%s) VALUES (%s)", table, colList, valList)
		case "skip":
			return fmt.Sprintf("INSERT OR IGNORE INTO %s (%s) VALUES (%s)", table, colList, valList)
		default:
			return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, colList, valList)
		}
	case "mysql", "mariadb":
		switch onConflict {
		case "overwrite":
			return fmt.Sprintf("REPLACE INTO %s (%s) VALUES (%s)", table, colList, valList)
		case "skip":
			return fmt.Sprintf("INSERT IGNORE INTO %s (%s) VALUES (%s)", table, colList, valList)
		default:
			return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, colList, valList)
		}
	case "postgresql":
		base := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, colList, valList)
		switch onConflict {
		case "skip":
			return base + " ON CONFLICT DO NOTHING"
		case "overwrite":
			// Requires knowledge of the primary key, which we don't have here.
			// Fall back to DO NOTHING for safety.
			return base + " ON CONFLICT DO NOTHING"
		default:
			return base
		}
	default:
		return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, colList, valList)
	}
}
