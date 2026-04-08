package engine

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// transferTablesSQL implements row-level data transfer between two *sql.DB
// connections. The dialect parameter controls conflict-handling and identifier
// quoting syntax.
func transferTablesSQL(ctx context.Context, srcDB, dstDB *sql.DB, opts TransferOptions, dialect string) error {
	for _, table := range opts.Tables {
		if err := transferOneTable(ctx, srcDB, dstDB, table, opts.OnConflict, dialect); err != nil {
			return fmt.Errorf("%w: table %s: %v", ErrTransferFailed, table, err)
		}
	}
	return nil
}

func transferOneTable(ctx context.Context, srcDB, dstDB *sql.DB, table, onConflict, dialect string) error {
	// Read all rows from source. Table names come from ListTables (catalog
	// data), so they are valid unquoted identifiers in the source engine.
	// We don't quote here because we don't know the source dialect.
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
	insertSQL, err := buildInsertSQL(table, columns, onConflict, dialect)
	if err != nil {
		return err
	}

	// Wrap inserts in a transaction for atomic per-table transfer.
	tx, err := dstDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("starting transaction: %v", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, insertSQL)
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
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating rows: %v", err)
	}

	return tx.Commit()
}

func buildInsertSQL(table string, columns []string, onConflict, dialect string) (string, error) {
	quotedTable := quoteIdent(table, dialect)
	quotedCols := make([]string, len(columns))
	for i, c := range columns {
		quotedCols[i] = quoteIdent(c, dialect)
	}
	colList := strings.Join(quotedCols, ", ")

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
			return fmt.Sprintf("INSERT OR REPLACE INTO %s (%s) VALUES (%s)", quotedTable, colList, valList), nil
		case "skip":
			return fmt.Sprintf("INSERT OR IGNORE INTO %s (%s) VALUES (%s)", quotedTable, colList, valList), nil
		default:
			return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quotedTable, colList, valList), nil
		}
	case "mysql", "mariadb":
		switch onConflict {
		case "overwrite":
			return fmt.Sprintf("REPLACE INTO %s (%s) VALUES (%s)", quotedTable, colList, valList), nil
		case "skip":
			return fmt.Sprintf("INSERT IGNORE INTO %s (%s) VALUES (%s)", quotedTable, colList, valList), nil
		default:
			return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quotedTable, colList, valList), nil
		}
	case "postgresql":
		base := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quotedTable, colList, valList)
		switch onConflict {
		case "skip":
			return base + " ON CONFLICT DO NOTHING", nil
		case "overwrite":
			return "", fmt.Errorf("%w: PostgreSQL does not support --on-conflict=overwrite without primary key knowledge", ErrTransferFailed)
		default:
			return base, nil
		}
	default:
		return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quotedTable, colList, valList), nil
	}
}

// quoteIdent quotes a SQL identifier using dialect-appropriate quoting to
// prevent SQL injection via table or column names.
func quoteIdent(name, dialect string) string {
	switch dialect {
	case "mysql", "mariadb":
		// Backtick quoting; escape embedded backticks by doubling.
		return "`" + strings.ReplaceAll(name, "`", "``") + "`"
	default: // postgresql, sqlite, and others use double-quote quoting.
		return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
	}
}
