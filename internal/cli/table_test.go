package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestTable_BasicRendering(t *testing.T) {
	tbl := NewTable("ID", "NAME", "STATUS")
	tbl.AddRow("1", "node-us", "online")
	tbl.AddRow("2", "node-eu", "offline")

	var buf bytes.Buffer
	tbl.Render(&buf)
	output := buf.String()

	// Check header is present.
	if !strings.Contains(output, "ID") {
		t.Error("output missing header 'ID'")
	}
	if !strings.Contains(output, "NAME") {
		t.Error("output missing header 'NAME'")
	}
	if !strings.Contains(output, "STATUS") {
		t.Error("output missing header 'STATUS'")
	}

	// Check data rows are present.
	if !strings.Contains(output, "node-us") {
		t.Error("output missing 'node-us'")
	}
	if !strings.Contains(output, "node-eu") {
		t.Error("output missing 'node-eu'")
	}

	// Check separator line exists.
	lines := strings.Split(output, "\n")
	if len(lines) < 4 {
		t.Fatalf("expected at least 4 lines (header + separator + 2 rows), got %d", len(lines))
	}

	// Separator line should contain dashes.
	separatorLine := lines[1]
	dashCount := strings.Count(separatorLine, "-")
	if dashCount == 0 {
		t.Error("separator line has no dashes")
	}
}

func TestTable_ColumnAlignment(t *testing.T) {
	tbl := NewTable("ID", "NAME")
	tbl.AddRow("1", "short")
	tbl.AddRow("100", "a-much-longer-name")

	var buf bytes.Buffer
	tbl.Render(&buf)
	lines := strings.Split(buf.String(), "\n")

	// All non-empty lines should have their second column starting at the same position.
	// The "NAME" column should start after the widest ID value ("100") + padding.
	if len(lines) < 4 {
		t.Fatalf("expected at least 4 lines, got %d", len(lines))
	}

	// Check that the header "NAME" and data "short" are aligned.
	headerNameIdx := strings.Index(lines[0], "NAME")
	row1NameIdx := strings.Index(lines[2], "short")
	row2NameIdx := strings.Index(lines[3], "a-much-longer-name")

	if headerNameIdx != row1NameIdx || headerNameIdx != row2NameIdx {
		t.Errorf("columns not aligned: header=%d, row1=%d, row2=%d",
			headerNameIdx, row1NameIdx, row2NameIdx)
	}
}

func TestTable_EmptyTable(t *testing.T) {
	tbl := NewTable("A", "B", "C")

	var buf bytes.Buffer
	tbl.Render(&buf)
	output := buf.String()

	// Should have header and separator but no data rows.
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines (header + separator), got %d", len(lines))
	}
}

func TestTable_NoColumns(t *testing.T) {
	tbl := NewTable()

	var buf bytes.Buffer
	tbl.Render(&buf)

	if buf.Len() != 0 {
		t.Errorf("rendering table with no columns should produce empty output, got %q", buf.String())
	}
}

func TestTable_MissingValues(t *testing.T) {
	tbl := NewTable("A", "B", "C")
	tbl.AddRow("1") // only one value, others should be empty

	var buf bytes.Buffer
	tbl.Render(&buf)
	output := buf.String()

	if !strings.Contains(output, "1") {
		t.Error("output missing value '1'")
	}

	// Should still render without panic.
	if tbl.RowCount() != 1 {
		t.Errorf("RowCount() = %d, want 1", tbl.RowCount())
	}
}

func TestTable_ExtraValues(t *testing.T) {
	tbl := NewTable("A", "B")
	tbl.AddRow("1", "2", "3") // extra value "3" should be ignored

	var buf bytes.Buffer
	tbl.Render(&buf)
	output := buf.String()

	if strings.Contains(output, "3") {
		t.Error("extra value '3' should not appear in output")
	}
}

func TestTable_RowCount(t *testing.T) {
	tbl := NewTable("X")
	if tbl.RowCount() != 0 {
		t.Errorf("RowCount() on empty table = %d, want 0", tbl.RowCount())
	}

	tbl.AddRow("a")
	tbl.AddRow("b")
	tbl.AddRow("c")

	if tbl.RowCount() != 3 {
		t.Errorf("RowCount() = %d, want 3", tbl.RowCount())
	}
}

func TestTable_SpecialCharacters(t *testing.T) {
	tbl := NewTable("KEY", "VALUE")
	tbl.AddRow("path", "/var/run/panel.sock")
	tbl.AddRow("url", "http://127.0.0.1:8080")

	var buf bytes.Buffer
	tbl.Render(&buf)
	output := buf.String()

	if !strings.Contains(output, "/var/run/panel.sock") {
		t.Error("output missing path value")
	}
	if !strings.Contains(output, "http://127.0.0.1:8080") {
		t.Error("output missing url value")
	}
}

func TestTable_SingleColumn(t *testing.T) {
	tbl := NewTable("NAME")
	tbl.AddRow("alpha")
	tbl.AddRow("beta")
	tbl.AddRow("gamma")

	var buf bytes.Buffer
	tbl.Render(&buf)
	output := buf.String()

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	// header + separator + 3 data rows = 5 lines.
	if len(lines) != 5 {
		t.Errorf("expected 5 lines, got %d", len(lines))
	}

	if !strings.Contains(lines[0], "NAME") {
		t.Error("header missing 'NAME'")
	}
	if !strings.Contains(lines[2], "alpha") {
		t.Error("first row missing 'alpha'")
	}
}

func TestTable_ManyColumns(t *testing.T) {
	tbl := NewTable("ID", "NAME", "IP", "SCORE", "STATUS", "REGION", "LAST_SEEN")
	tbl.AddRow("1", "node-us-east", "10.0.0.1", "0.95", "online", "us-east-1", "2024-01-01T00:00:00Z")
	tbl.AddRow("2", "node-eu-west", "10.0.0.2", "0.72", "stale", "eu-west-1", "2024-01-01T00:05:00Z")

	var buf bytes.Buffer
	tbl.Render(&buf)
	output := buf.String()

	// All headers should appear on the first line.
	headerLine := strings.Split(output, "\n")[0]
	for _, col := range []string{"ID", "NAME", "IP", "SCORE", "STATUS", "REGION", "LAST_SEEN"} {
		if !strings.Contains(headerLine, col) {
			t.Errorf("header missing column %q", col)
		}
	}

	// Both rows should be present.
	if !strings.Contains(output, "node-us-east") {
		t.Error("output missing 'node-us-east'")
	}
	if !strings.Contains(output, "node-eu-west") {
		t.Error("output missing 'node-eu-west'")
	}
}

func TestTable_LongValuesAffectWidth(t *testing.T) {
	tbl := NewTable("ID", "DESCRIPTION")
	tbl.AddRow("1", "short")
	tbl.AddRow("2", "a very long description that exceeds the header width significantly")

	var buf bytes.Buffer
	tbl.Render(&buf)
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")

	// The separator line should be at least as wide as the longest value.
	separatorLine := lines[1]
	// Separator has dashes for each column width. The DESCRIPTION column dashes
	// should be at least as long as the longest value.
	dashSegments := strings.Fields(separatorLine)
	if len(dashSegments) < 2 {
		t.Fatal("expected at least 2 dash segments in separator")
	}
	descDashes := dashSegments[1]
	longVal := "a very long description that exceeds the header width significantly"
	if len(descDashes) < len(longVal) {
		t.Errorf("separator dashes (%d) shorter than longest value (%d)", len(descDashes), len(longVal))
	}
}

func TestTable_MultiRowLineCount(t *testing.T) {
	tests := []struct {
		name      string
		rows      int
		wantLines int
	}{
		{name: "no rows", rows: 0, wantLines: 2},    // header + separator
		{name: "one row", rows: 1, wantLines: 3},    // header + separator + 1
		{name: "five rows", rows: 5, wantLines: 7},  // header + separator + 5
		{name: "ten rows", rows: 10, wantLines: 12}, // header + separator + 10
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tbl := NewTable("X", "Y")
			for i := 0; i < tt.rows; i++ {
				tbl.AddRow("a", "b")
			}

			var buf bytes.Buffer
			tbl.Render(&buf)
			lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")

			if len(lines) != tt.wantLines {
				t.Errorf("got %d lines, want %d", len(lines), tt.wantLines)
			}
		})
	}
}

func TestTable_PaddingBetweenColumns(t *testing.T) {
	tbl := NewTable("A", "B")
	tbl.AddRow("xx", "yy")

	var buf bytes.Buffer
	tbl.Render(&buf)
	lines := strings.Split(buf.String(), "\n")

	// Header line: "A" padded to width 2 (matches "xx"), then 2 spaces padding, then "B".
	// Expected: "A   B" (width 2 for A + 2 padding + B).
	headerLine := lines[0]
	bIdx := strings.Index(headerLine, "B")
	if bIdx < 0 {
		t.Fatal("header missing 'B'")
	}

	// B should start after A's column width (2) + padding (2) = position 4.
	if bIdx != 4 {
		t.Errorf("column B starts at position %d, expected 4", bIdx)
	}
}
