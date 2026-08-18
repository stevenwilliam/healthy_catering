package app

import (
	"bytes"
	"strings"
	"testing"
)

// The delimiter is a pipe, and it is a real CSV rather than a join.
func TestWriteCSVUsesPipeAndQuotesProperly(t *testing.T) {
	var buf bytes.Buffer
	err := WriteCSV(&buf,
		[]string{"name", "address", "note"},
		[][]string{
			// The case the pipe exists for: commas in real Indonesian data.
			{"Ayam Bakar", "Jl. Kebon Sirih No. 10, Menteng, Jakarta", "pagar abu-abu"},
			// The case the pipe creates: a value containing the delimiter.
			{"Sup | Bening", `he said "hi"`, "line one\nline two"},
		})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	if !strings.HasPrefix(out, "name|address|note\n") {
		t.Errorf("header not pipe-separated:\n%s", out)
	}
	// The address keeps its commas as DATA, in one field.
	if !strings.Contains(out, "Jl. Kebon Sirih No. 10, Menteng, Jakarta") {
		t.Errorf("the address was split or mangled:\n%s", out)
	}
	// A value containing the delimiter must be quoted, or the file is corrupt.
	if !strings.Contains(out, `"Sup | Bening"`) {
		t.Errorf("a value containing a pipe was not quoted:\n%s", out)
	}
	if !strings.Contains(out, `"he said ""hi"""`) {
		t.Errorf("embedded quotes not escaped:\n%s", out)
	}
}

// A CSV is an executable document in Excel; the guard must survive the
// delimiter change.
func TestWriteCSVStillBlocksFormulaInjection(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteCSV(&buf, []string{"h"}, [][]string{
		{"=cmd|' /C calc'!A0"}, {"+1+1"}, {"-2"}, {"@SUM(A1)"},
	}); err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n")[1:] {
		if !strings.HasPrefix(strings.TrimPrefix(line, `"`), "'") {
			t.Errorf("cell not neutralised: %s", line)
		}
	}
}
