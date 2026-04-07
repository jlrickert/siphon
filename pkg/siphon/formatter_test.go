package siphon

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTableFormatter_MultipleColumns(t *testing.T) {
	t.Parallel()
	result := &QueryResult{
		Columns: []string{"id", "name", "email"},
		Rows: [][]string{
			{"1", "Alice", "alice@example.com"},
			{"2", "Bob", "bob@example.com"},
		},
	}

	var buf bytes.Buffer
	f, err := NewFormatter(FormatTable)
	require.NoError(t, err)
	require.NoError(t, f.Format(&buf, result))

	out := buf.String()
	require.Contains(t, out, "id")
	require.Contains(t, out, "name")
	require.Contains(t, out, "email")
	require.Contains(t, out, "Alice")
	require.Contains(t, out, "bob@example.com")
	// Separator line should exist.
	require.Contains(t, out, "--")
}

func TestCSVFormatter(t *testing.T) {
	t.Parallel()
	result := &QueryResult{
		Columns: []string{"id", "name"},
		Rows: [][]string{
			{"1", "Alice"},
			{"2", "Bob"},
		},
	}

	var buf bytes.Buffer
	f, err := NewFormatter(FormatCSV)
	require.NoError(t, err)
	require.NoError(t, f.Format(&buf, result))

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 3) // header + 2 rows
	require.Equal(t, "id,name", lines[0])
	require.Equal(t, "1,Alice", lines[1])
	require.Equal(t, "2,Bob", lines[2])
}

func TestJSONFormatter(t *testing.T) {
	t.Parallel()
	result := &QueryResult{
		Columns: []string{"id", "name"},
		Rows: [][]string{
			{"1", "Alice"},
			{"2", "Bob"},
		},
	}

	var buf bytes.Buffer
	f, err := NewFormatter(FormatJSON)
	require.NoError(t, err)
	require.NoError(t, f.Format(&buf, result))

	var objects []map[string]string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &objects))
	require.Len(t, objects, 2)
	require.Equal(t, "1", objects[0]["id"])
	require.Equal(t, "Alice", objects[0]["name"])
	require.Equal(t, "2", objects[1]["id"])
	require.Equal(t, "Bob", objects[1]["name"])
}

func TestFormatter_EmptyResult(t *testing.T) {
	t.Parallel()
	result := &QueryResult{
		Columns: []string{"id", "name"},
		Rows:    nil,
	}

	for _, format := range []OutputFormat{FormatTable, FormatCSV, FormatJSON} {
		t.Run(string(format), func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			f, err := NewFormatter(format)
			require.NoError(t, err)
			require.NoError(t, f.Format(&buf, result))
			// Should not panic or error, even with no rows.
			require.NotEmpty(t, buf.String())
		})
	}
}

func TestFormatter_SingleRow(t *testing.T) {
	t.Parallel()
	result := &QueryResult{
		Columns: []string{"count"},
		Rows:    [][]string{{"42"}},
	}

	var buf bytes.Buffer
	f, err := NewFormatter(FormatTable)
	require.NoError(t, err)
	require.NoError(t, f.Format(&buf, result))
	require.Contains(t, buf.String(), "42")
}

func TestFormatter_UnknownFormat(t *testing.T) {
	t.Parallel()
	_, err := NewFormatter("xml")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown output format")
}

func TestFormatter_EmptyFormatDefaultsToTable(t *testing.T) {
	t.Parallel()
	f, err := NewFormatter("")
	require.NoError(t, err)
	require.NotNil(t, f)
}

func TestTableFormatter_NoColumns(t *testing.T) {
	t.Parallel()
	result := &QueryResult{}

	var buf bytes.Buffer
	f, err := NewFormatter(FormatTable)
	require.NoError(t, err)
	require.NoError(t, f.Format(&buf, result))
	require.Empty(t, buf.String())
}
