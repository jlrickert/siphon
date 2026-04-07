package siphon

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// OutputFormat identifies a result output format.
type OutputFormat string

const (
	FormatTable OutputFormat = "table"
	FormatCSV   OutputFormat = "csv"
	FormatJSON  OutputFormat = "json"
)

// Formatter renders a QueryResult to a writer.
type Formatter interface {
	Format(w io.Writer, result *QueryResult) error
}

// NewFormatter returns a Formatter for the given format string.
func NewFormatter(format OutputFormat) (Formatter, error) {
	switch format {
	case FormatTable, "":
		return &tableFormatter{}, nil
	case FormatCSV:
		return &csvFormatter{}, nil
	case FormatJSON:
		return &jsonFormatter{}, nil
	default:
		return nil, fmt.Errorf("unknown output format: %s", format)
	}
}

// --- table ---

type tableFormatter struct{}

func (f *tableFormatter) Format(w io.Writer, result *QueryResult) error {
	if len(result.Columns) == 0 {
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	// Header.
	_, _ = fmt.Fprintln(tw, strings.Join(result.Columns, "\t"))

	// Separator.
	seps := make([]string, len(result.Columns))
	for i, col := range result.Columns {
		seps[i] = strings.Repeat("-", len(col))
	}
	_, _ = fmt.Fprintln(tw, strings.Join(seps, "\t"))

	// Rows.
	for _, row := range result.Rows {
		_, _ = fmt.Fprintln(tw, strings.Join(row, "\t"))
	}

	return tw.Flush()
}

// --- csv ---

type csvFormatter struct{}

func (f *csvFormatter) Format(w io.Writer, result *QueryResult) error {
	cw := csv.NewWriter(w)

	if len(result.Columns) > 0 {
		if err := cw.Write(result.Columns); err != nil {
			return err
		}
	}
	for _, row := range result.Rows {
		if err := cw.Write(row); err != nil {
			return err
		}
	}

	cw.Flush()
	return cw.Error()
}

// --- json ---

type jsonFormatter struct{}

func (f *jsonFormatter) Format(w io.Writer, result *QueryResult) error {
	objects := make([]map[string]string, 0, len(result.Rows))
	for _, row := range result.Rows {
		obj := make(map[string]string, len(result.Columns))
		for i, col := range result.Columns {
			if i < len(row) {
				obj[col] = row[i]
			}
		}
		objects = append(objects, obj)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(objects)
}
