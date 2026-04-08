package siphon

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseColonSyntax(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantConn string
		wantDB   string
	}{
		{
			name:     "connection only",
			input:    "myconn",
			wantConn: "myconn",
			wantDB:   "",
		},
		{
			name:     "connection and database",
			input:    "myconn:mydb",
			wantConn: "myconn",
			wantDB:   "mydb",
		},
		{
			name:     "connection with empty database",
			input:    "myconn:",
			wantConn: "myconn",
			wantDB:   "",
		},
		{
			name:     "empty connection with database",
			input:    ":mydb",
			wantConn: "",
			wantDB:   "mydb",
		},
		{
			name:     "empty string",
			input:    "",
			wantConn: "",
			wantDB:   "",
		},
		{
			name:     "multiple colons uses first",
			input:    "conn:db:extra",
			wantConn: "conn",
			wantDB:   "db:extra",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotConn, gotDB := ParseColonSyntax(tc.input)
			require.Equal(t, tc.wantConn, gotConn)
			require.Equal(t, tc.wantDB, gotDB)
		})
	}
}
