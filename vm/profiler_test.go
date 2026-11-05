package vm

import (
	"bramandscript/compiler"
	"bramandscript/lexer"
	"bramandscript/parser"
	"strings"
	"testing"
)

func TestProfilerAndDisasm(t *testing.T) {
	input := `
	let sum = 0
	let i = 0
	while i < 10 {
		sum += i
		i += 1
	}
	sum
	`

	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.SyntaxErrors) > 0 {
		t.Fatalf("parser errors: %v", p.SyntaxErrors)
	}

	comp := compiler.New()
	err := comp.Compile(prog)
	if err != nil {
		t.Fatalf("compiler error: %v", err)
	}

	bytecode := comp.Bytecode()
	disasm := compiler.FormatBytecode(bytecode)
	if !strings.Contains(disasm, "OpConstant") || !strings.Contains(disasm, "Constant Pool") {
		t.Errorf("FormatBytecode failed: %s", disasm)
	}

	machine := New(bytecode)
	report, err := machine.RunProfiled()
	if err != nil {
		t.Fatalf("VM RunProfiled failed: %v", err)
	}

	if report.TotalInstructions == 0 {
		t.Errorf("Expected instructions to be > 0, got %d", report.TotalInstructions)
	}

	repStr := report.String()
	if !strings.Contains(repStr, "Execution Profiler") || !strings.Contains(repStr, "OpAdd") {
		t.Errorf("Profile report missing expected content: %s", repStr)
	}
}
