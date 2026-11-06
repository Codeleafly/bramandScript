package repl

import (
	"bytes"
	"strings"
	"testing"

	"bramandscript/diagnostics"
	"bramandscript/object"
)

func TestFormatReplValue(t *testing.T) {
	tests := []struct {
		name     string
		input    object.Object
		expected string
	}{
		{
			name:     "Integer",
			input:    &object.Integer{Value: 42},
			expected: diagnostics.ColorYellow + "42" + diagnostics.ColorReset,
		},
		{
			name:     "Float",
			input:    &object.Float{Value: 3.14},
			expected: diagnostics.ColorYellow + "3.14" + diagnostics.ColorReset,
		},
		{
			name:     "String",
			input:    &object.String{Value: "hello"},
			expected: diagnostics.ColorGreen + "\"hello\"" + diagnostics.ColorReset,
		},
		{
			name:     "Boolean",
			input:    &object.Boolean{Value: true},
			expected: diagnostics.ColorMagenta + "true" + diagnostics.ColorReset,
		},
		{
			name:     "Null",
			input:    &object.Null{},
			expected: diagnostics.ColorDim + "null" + diagnostics.ColorReset,
		},
		{
			name:     "Error",
			input:    &object.Error{Message: "boom"},
			expected: diagnostics.ColorRed + "Runtime Error: boom" + diagnostics.ColorReset,
		},
		{
			name:  "Array",
			input: &object.Array{Elements: []object.Object{&object.Integer{Value: 1}, &object.String{Value: "a"}}},
			expected: "[" + diagnostics.ColorYellow + "1" + diagnostics.ColorReset + ", " +
				diagnostics.ColorGreen + "\"a\"" + diagnostics.ColorReset + "]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatReplValue(tt.input)
			if got != tt.expected {
				t.Errorf("FormatReplValue(%v) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestReplExecutionNewlines(t *testing.T) {
	// Test expression evaluation
	var out bytes.Buffer
	in := strings.NewReader("5\n")
	Start(in, &out)

	outStr := out.String()
	if !strings.Contains(outStr, diagnostics.ColorYellow+"5"+diagnostics.ColorReset) {
		t.Errorf("Expected yellow 5 in REPL output, got: %s", outStr)
	}

	// Test print output followed by prompt with newline
	var out2 bytes.Buffer
	in2 := strings.NewReader("print(42)\n")
	Start(in2, &out2)

	out2Str := out2.String()
	// Should contain "42\n✦ bmd ❯ " not "42✦ bmd ❯ "
	if strings.Contains(out2Str, "42✦ bmd ❯ ") {
		t.Errorf("Glued prompt detected in: %s", out2Str)
	}
	if !strings.Contains(out2Str, "42\n") {
		t.Errorf("Expected clean newline after 42 in: %s", out2Str)
	}
}
