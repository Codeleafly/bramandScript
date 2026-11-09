package vm

import (
	"path/filepath"
	"testing"

	"bramandscript/compiler"
	"bramandscript/lexer"
	"bramandscript/object"
	"bramandscript/parser"
)

func TestTimeTravelExecutionRecorderAndReplay(t *testing.T) {
	source := `
let x = 10;
let y = 20;
let z = x + y;
`
	l := lexer.New(source)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.SyntaxErrors) > 0 {
		t.Fatalf("syntax errors: %v", p.SyntaxErrors)
	}

	comp := compiler.New()
	if err := comp.Compile(prog); err != nil {
		t.Fatalf("compile error: %s", err)
	}

	machine := New(comp.Bytecode())
	trace, err := machine.RunRecorded("test_script.bmd")
	if err != nil {
		t.Fatalf("runtime error: %s", err)
	}

	if trace == nil || trace.TotalSteps == 0 {
		t.Fatalf("expected non-empty trace, got: %v", trace)
	}

	tmpDir := t.TempDir()
	tracePath := filepath.Join(tmpDir, "trace.json")
	if err := trace.Save(tracePath); err != nil {
		t.Fatalf("failed to save trace: %s", err)
	}

	loadedTrace, err := LoadTrace(tracePath)
	if err != nil {
		t.Fatalf("failed to load trace: %s", err)
	}

	if loadedTrace.TotalSteps != trace.TotalSteps {
		t.Errorf("expected total steps %d, got %d", trace.TotalSteps, loadedTrace.TotalSteps)
	}

	debugger := NewReplayDebugger(loadedTrace)
	step1 := debugger.Current()
	if step1 == nil || step1.Step != 1 {
		t.Fatalf("expected step 1 at start, got %v", step1)
	}

	// Step forward
	step2 := debugger.Step()
	if step2.Step != 2 {
		t.Errorf("expected step 2, got %d", step2.Step)
	}

	// Time-travel backwards!
	backStep := debugger.StepBack()
	if backStep.Step != 1 {
		t.Errorf("expected step 1 after step back, got %d", backStep.Step)
	}

	// Goto
	gotoStep := debugger.Goto(5)
	if gotoStep.Step != 5 {
		t.Errorf("expected step 5 after goto, got %d", gotoStep.Step)
	}

	// Inspect history
	history := debugger.InspectHistory(5)
	if len(history) == 0 {
		t.Errorf("expected history checkpoints, got empty")
	}
}

func TestComptimeMetaprogramming(t *testing.T) {
	source := `
let x = comptime { 5 * 10 + 2 };
`
	l := lexer.New(source)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.SyntaxErrors) > 0 {
		t.Fatalf("syntax errors: %v", p.SyntaxErrors)
	}

	comp := compiler.New()
	if err := comp.Compile(prog); err != nil {
		t.Fatalf("compile error: %s", err)
	}

	// Verify constant was inlined at compile time
	constants := comp.Bytecode().Constants
	foundInlined := false
	for _, c := range constants {
		if intObj, ok := c.(*object.Integer); ok && intObj.Value == 52 {
			foundInlined = true
			break
		}
	}

	if !foundInlined {
		t.Fatalf("expected inlined constant 52 in constants pool, constants were: %v", constants)
	}
}
