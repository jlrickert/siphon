package siphon

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsWriteQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		sql     string
		isWrite bool
	}{
		{"SELECT is read", "SELECT * FROM users", false},
		{"SHOW is read", "SHOW TABLES", false},
		{"DESCRIBE is read", "DESCRIBE users", false},
		{"EXPLAIN is read", "EXPLAIN SELECT * FROM users", false},
		{"WITH is read", "WITH cte AS (SELECT 1) SELECT * FROM cte", false},

		{"INSERT is write", "INSERT INTO users (name) VALUES ('a')", true},
		{"UPDATE is write", "UPDATE users SET name = 'b'", true},
		{"DELETE is write", "DELETE FROM users WHERE id = 1", true},
		{"DROP is write", "DROP TABLE users", true},
		{"ALTER is write", "ALTER TABLE users ADD COLUMN age INT", true},
		{"CREATE is write", "CREATE TABLE users (id INT)", true},
		{"TRUNCATE is write", "TRUNCATE TABLE users", true},
		{"REPLACE is write", "REPLACE INTO users VALUES (1, 'a')", true},

		{"case insensitive select", "select * from users", false},
		{"case insensitive insert", "insert into users values (1)", true},
		{"mixed case", "Insert INTO users VALUES (1)", true},

		{"leading whitespace", "   SELECT 1", false},
		{"leading newlines", "\n\nSELECT 1", false},
		{"leading tabs", "\t\tDELETE FROM users", true},

		{"single-line comment before SELECT", "-- comment\nSELECT 1", false},
		{"single-line comment before INSERT", "-- comment\nINSERT INTO t VALUES (1)", true},
		{"block comment before SELECT", "/* comment */ SELECT 1", false},
		{"block comment before DELETE", "/* multi\nline\ncomment */\nDELETE FROM t", true},

		{"empty string", "", false},
		{"only whitespace", "   \n\t  ", false},
		{"only comment", "-- just a comment", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := IsWriteQuery(tc.sql)
			require.Equal(t, tc.isWrite, got, "IsWriteQuery(%q)", tc.sql)
		})
	}
}
