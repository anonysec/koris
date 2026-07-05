package cli

import (
	"fmt"
	"io"
	"strings"
)

// Table provides formatted columnar output for CLI commands.
// It auto-calculates column widths based on content and renders
// aligned output with padding.
type Table struct {
	columns []string
	rows    [][]string
	padding int
}

// NewTable creates a Table with the given column headers.
func NewTable(columns ...string) *Table {
	return &Table{
		columns: columns,
		padding: 2,
	}
}

// AddRow appends a row of values. The number of values should match
// the number of columns. Extra values are ignored; missing values
// are treated as empty strings.
func (t *Table) AddRow(values ...string) {
	row := make([]string, len(t.columns))
	for i := range row {
		if i < len(values) {
			row[i] = values[i]
		}
	}
	t.rows = append(t.rows, row)
}

// Render writes the formatted table to the given writer.
// Columns are left-aligned with padding between them.
func (t *Table) Render(w io.Writer) {
	if len(t.columns) == 0 {
		return
	}

	widths := t.computeWidths()

	// Render header.
	t.renderRow(w, t.columns, widths)

	// Render separator.
	for i, width := range widths {
		if i > 0 {
			fmt.Fprint(w, strings.Repeat(" ", t.padding))
		}
		fmt.Fprint(w, strings.Repeat("-", width))
	}
	fmt.Fprintln(w)

	// Render data rows.
	for _, row := range t.rows {
		t.renderRow(w, row, widths)
	}
}

// RowCount returns the number of data rows (excluding header).
func (t *Table) RowCount() int {
	return len(t.rows)
}

// computeWidths determines the maximum width for each column
// based on headers and row data.
func (t *Table) computeWidths() []int {
	widths := make([]int, len(t.columns))

	// Start with header widths.
	for i, col := range t.columns {
		widths[i] = len(col)
	}

	// Check row data.
	for _, row := range t.rows {
		for i, val := range row {
			if i < len(widths) && len(val) > widths[i] {
				widths[i] = len(val)
			}
		}
	}

	return widths
}

// renderRow writes a single row with proper column alignment.
func (t *Table) renderRow(w io.Writer, values []string, widths []int) {
	for i, val := range values {
		if i > 0 {
			fmt.Fprint(w, strings.Repeat(" ", t.padding))
		}
		if i < len(widths)-1 {
			// Left-align with padding to column width.
			fmt.Fprintf(w, "%-*s", widths[i], val)
		} else {
			// Last column doesn't need trailing spaces.
			fmt.Fprint(w, val)
		}
	}
	fmt.Fprintln(w)
}
