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

func TestDiagnosticOperatorSpansAndHints(t *testing.T) {
	// Operator span length test
	tests := []struct {
		line   string
		offset int
		want   int
	}{
		{"let a ??= 10", 6, 3}, // "??=" at index 6 -> length 3
		{"let a == 10", 6, 2},  // "==" at index 6 -> length 2
		{"let a != 10", 6, 2},  // "!=" at index 6 -> length 2
		{"let a ?. b", 6, 2},   // "?." at index 6 -> length 2
		{"let a |> b", 6, 2},   // "|>" at index 6 -> length 2
		{"let my_var = 1", 4, 6}, // identifier "my_var" -> length 6
	}

	for _, tc := range tests {
		got := getWordLength(tc.line, tc.offset)
		if got != tc.want {
			t.Errorf("getWordLength(%q, %d) = %d, want %d", tc.line, tc.offset, got, tc.want)
		}
	}

	// Hint tests
	permHint := GenerateHint("PermissionDenied: exec is not permitted")
	if !strings.Contains(permHint, "--allow-run") {
		t.Errorf("expected permission hint to mention --allow-run, got: %s", permHint)
	}

	nullHint := GenerateHint("calling non-function and non-builtin: NULL")
	if !strings.Contains(nullHint, "optional chaining") {
		t.Errorf("expected null call hint to mention optional chaining, got: %s", nullHint)
	}
}
