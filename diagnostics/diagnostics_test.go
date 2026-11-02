package diagnostics

import (
	"strings"
	"testing"
)

func TestDiagnosticFormatContextLines(t *testing.T) {
	source := `let a = 1
let b = 2
let c = 3
let d = 4
let e = 5
let target = undefined_var + 10
let f = 6
let g = 7
let h = 8
let i = 9
let j = 10`

	d := Diagnostic{
		Filename: "test.bmd",
		Source:   source,
		Line:     6,
		Col:      14,
		Kind:     "CompileError",
		Message:  "undefined variable 'undefined_var'",
		Hint:     GenerateHint("undefined variable 'undefined_var'"),
	}

	formatted := d.Format()

	// Verify error header
	if !strings.Contains(formatted, "error[CompileError]") {
		t.Errorf("expected error header, got: %s", formatted)
	}

	// Verify location pointer
	if !strings.Contains(formatted, "test.bmd:6:14") || !strings.Contains(formatted, "-->") {
		t.Errorf("expected location link, got: %s", formatted)
	}

	// Verify 5 lines before (lines 1 to 5)
	for _, expectedLine := range []string{"let a = 1", "let b = 2", "let c = 3", "let d = 4", "let e = 5"} {
		if !strings.Contains(formatted, expectedLine) {
			t.Errorf("missing context before: %s in %s", expectedLine, formatted)
		}
	}

	// Verify error line and pointer
	if !strings.Contains(formatted, "let target = undefined_var + 10") {
		t.Errorf("missing error line text in: %s", formatted)
	}
	if !strings.Contains(formatted, "^") {
		t.Errorf("missing caret pointer in: %s", formatted)
	}

	// Verify 5 lines after (lines 7 to 11)
	for _, expectedLine := range []string{"let f = 6", "let g = 7", "let h = 8", "let i = 9", "let j = 10"} {
		if !strings.Contains(formatted, expectedLine) {
			t.Errorf("missing context after: %s in %s", expectedLine, formatted)
		}
	}

	// Verify intelligent hint
	if !strings.Contains(formatted, "[Hint:") {
		t.Errorf("missing hint in: %s", formatted)
	}
}
