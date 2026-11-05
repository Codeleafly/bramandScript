package compiler

import (
	"testing"

	"bramandscript/code"
	"bramandscript/lexer"
	"bramandscript/parser"
)

func TestConstantFolding(t *testing.T) {
	input := "let x = 2 * 3 + 4 * 5;"
	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()

	c := NewOptimizingCompiler()
	err := c.Compile(prog)
	if err != nil {
		t.Fatalf("compiler error: %v", err)
	}

	bc := c.Bytecode()
	// Folded: 2*3 = 6, 4*5 = 20, 6+20 = 26
	// Single constant 26 instead of 4 constants and 3 arithmetic opcodes!
	if len(bc.Constants) != 1 {
		t.Fatalf("expected 1 constant, got %d", len(bc.Constants))
	}
}

func TestConstantFoldingPower(t *testing.T) {
	input := "let x = 2 ** 10;"
	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()

	c := NewOptimizingCompiler()
	err := c.Compile(prog)
	if err != nil {
		t.Fatalf("compiler error: %v", err)
	}

	bc := c.Bytecode()
	if len(bc.Constants) != 1 {
		t.Fatalf("expected 1 constant, got %d", len(bc.Constants))
	}
}

func TestInstructionReduction(t *testing.T) {
	input := "let a = 10 + 20 + 30 + 40;"
	l1 := lexer.New(input)
	p1 := parser.New(l1)
	prog1 := p1.ParseProgram()

	// Unoptimized
	c1 := New()
	_ = c1.Compile(prog1)
	bc1 := c1.Bytecode()

	// Optimized
	l2 := lexer.New(input)
	p2 := parser.New(l2)
	prog2 := p2.ParseProgram()

	c2 := NewOptimizingCompiler()
	_ = c2.Compile(prog2)
	bc2 := c2.Bytecode()

	if len(bc2.Instructions) >= len(bc1.Instructions) {
		t.Fatalf("expected optimized bytecode to be smaller: unoptimized=%d, optimized=%d",
			len(bc1.Instructions), len(bc2.Instructions))
	}
}

func TestPeepholeOptimizer(t *testing.T) {
	// Instructions with OpDup followed by OpPop
	ins := code.Instructions{}
	ins = append(ins, byte(code.OpConstant), 0, 0)
	ins = append(ins, byte(code.OpDup))
	ins = append(ins, byte(code.OpPop))

	opt := OptimizeBytecode(ins)
	expectedLen := 3 // only OpConstant remains
	if len(opt) != expectedLen {
		t.Fatalf("expected %d instructions after peephole, got %d", expectedLen, len(opt))
	}
}
