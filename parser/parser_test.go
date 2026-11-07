package parser

import (
	"bramandscript/ast"
	"bramandscript/lexer"
	"testing"
)

func TestLetStatements(t *testing.T) {
	input := `
	let x = 5;
	let y = 10.5;
	let foobar = "universe";
	`

	l := lexer.New(input)
	p := New(l)

	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 3 {
		t.Fatalf("program.Statements does not contain 3 statements. got=%d", len(program.Statements))
	}

	tests := []struct {
		expectedIdentifier string
	}{
		{"x"},
		{"y"},
		{"foobar"},
	}

	for i, tt := range tests {
		stmt := program.Statements[i]
		letStmt, ok := stmt.(*ast.LetStatement)
		if !ok {
			t.Fatalf("stmt is not *ast.LetStatement. got=%T", stmt)
		}
		if letStmt.Name.Value != tt.expectedIdentifier {
			t.Fatalf("letStmt.Name.Value not %s. got=%s", tt.expectedIdentifier, letStmt.Name.Value)
		}
	}
}

func TestOperatorPrecedenceParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"-a * b",
			"((-a) * b)",
		},
		{
			"!-a",
			"(!(-a))",
		},
		{
			"a + b + c",
			"((a + b) + c)",
		},
		{
			"a * b * c",
			"((a * b) * c)",
		},
		{
			"a * b / c",
			"((a * b) / c)",
		},
		{
			"a + b * c + d / e - f",
			"(((a + (b * c)) + (d / e)) - f)",
		},
		{
			"3 + 4; -5 * 5",
			"(3 + 4)((-5) * 5)",
		},
		{
			"5 > 4 == 3 < 4",
			"((5 > 4) == (3 < 4))",
		},
		{
			"5 < 4 != 3 > 4",
			"((5 < 4) != (3 > 4))",
		},
		{
			"3 + 4 * 5 == 3 * 1 + 4 * 5",
			"((3 + (4 * 5)) == ((3 * 1) + (4 * 5)))",
		},
		{
			"true == false",
			"(true == false)",
		},
		{
			"a && b || c",
			"((a && b) || c)",
		},
		{
			"1 + (2 + 3) + 4",
			"((1 + (2 + 3)) + 4)",
		},
		{
			"!(true == true)",
			"(!(true == true))",
		},
		{
			"a + add(b * c) + d",
			"((a + add((b * c))) + d)",
		},
		{
			"add(a, b, 1, 2 * 3, 4 + 5, add(6, 7 * 8))",
			"add(a, b, 1, (2 * 3), (4 + 5), add(6, (7 * 8)))",
		},
		{
			"a * [1, 2, 3, 4][b * c] * d",
			"((a * ([1, 2, 3, 4][(b * c)])) * d)",
		},
		{
			"ch <- 42",
			"(ch <- 42)",
		},
		{
			"<-ch",
			"(<-ch)",
		},
		{
			"ch <- 1 + 2",
			"(ch <- (1 + 2))",
		},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		actual := program.String()
		if actual != tt.expected {
			t.Errorf("expected=%q, got=%q", tt.expected, actual)
		}
	}
}

func checkParserErrors(t *testing.T, p *Parser) {
	errors := p.Errors()
	if len(errors) == 0 {
		return
	}

	t.Errorf("parser has %d errors", len(errors))
	for _, msg := range errors {
		t.Errorf("parser error: %q", msg)
	}
	t.FailNow()
}

func TestSpawnParsing(t *testing.T) {
	input := "spawn worker(10);"
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not *ast.ExpressionStatement. got=%T", program.Statements[0])
	}

	spawnExp, ok := stmt.Expression.(*ast.SpawnExpression)
	if !ok {
		t.Fatalf("stmt.Expression is not *ast.SpawnExpression. got=%T", stmt.Expression)
	}

	callExp, ok := spawnExp.Call.(*ast.CallExpression)
	if !ok {
		t.Fatalf("spawnExp.Call is not *ast.CallExpression. got=%T", spawnExp.Call)
	}

	if callExp.Function.String() != "worker" {
		t.Errorf("callExp.Function is not 'worker'. got=%s", callExp.Function.String())
	}
	if len(callExp.Arguments) != 1 {
		t.Fatalf("callExp.Arguments does not contain 1 argument. got=%d", len(callExp.Arguments))
	}
}

func TestSliceExpressionsWithStep(t *testing.T) {
	tests := []struct {
		input       string
		hasStart    bool
		hasEnd      bool
		hasStep     bool
	}{
		{"arr[:]", false, false, false},
		{"arr[::]", false, false, false},
		{"arr[::2]", false, false, true},
		{"arr[1::2]", true, false, true},
		{"arr[:4:2]", false, true, true},
		{"arr[1:4:2]", true, true, true},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("[%s] expected 1 statement, got=%d", tt.input, len(program.Statements))
		}
		stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
		if !ok {
			t.Fatalf("[%s] stmt is not ExpressionStatement", tt.input)
		}
		sliceExp, ok := stmt.Expression.(*ast.SliceExpression)
		if !ok {
			t.Fatalf("[%s] expr is not SliceExpression", tt.input)
		}

		if (sliceExp.Start != nil) != tt.hasStart {
			t.Errorf("[%s] hasStart mismatch: want=%v got=%v", tt.input, tt.hasStart, sliceExp.Start != nil)
		}
		if (sliceExp.End != nil) != tt.hasEnd {
			t.Errorf("[%s] hasEnd mismatch: want=%v got=%v", tt.input, tt.hasEnd, sliceExp.End != nil)
		}
		if (sliceExp.Step != nil) != tt.hasStep {
			t.Errorf("[%s] hasStep mismatch: want=%v got=%v", tt.input, tt.hasStep, sliceExp.Step != nil)
		}
	}
}

func TestTryCatchParsing(t *testing.T) {
	input := `
	try {
		let a = 10 / 0;
	} catch err {
		print(err);
	} finally {
		print("done");
	}
	`
	l := lexer.New(input)
	p := New(l)
	prog := p.ParseProgram()
	checkParserErrors(t, p)

	if len(prog.Statements) != 1 {
		t.Fatalf("expected 1 statement, got=%d", len(prog.Statements))
	}
	tc, ok := prog.Statements[0].(*ast.TryCatchStatement)
	if !ok {
		t.Fatalf("expected *ast.TryCatchStatement, got=%T", prog.Statements[0])
	}
	if tc.CatchVar == nil || tc.CatchVar.Value != "err" {
		t.Fatalf("expected catch var 'err', got=%v", tc.CatchVar)
	}
	if tc.Block == nil || len(tc.Block.Statements) != 1 {
		t.Fatalf("expected 1 statement in try block, got=%v", tc.Block)
	}
	if tc.CatchBlock == nil || len(tc.CatchBlock.Statements) != 1 {
		t.Fatalf("expected 1 statement in catch block, got=%v", tc.CatchBlock)
	}
	if tc.FinallyBlock == nil || len(tc.FinallyBlock.Statements) != 1 {
		t.Fatalf("expected 1 statement in finally block, got=%v", tc.FinallyBlock)
	}
}

func TestEnterpriseSyntaxParsing(t *testing.T) {
	// 1. Ternary expression
	inputTernary := `let res = active ? 100 : 200;`
	p := New(lexer.New(inputTernary))
	prog := p.ParseProgram()
	checkParserErrors(t, p)
	stmt := prog.Statements[0].(*ast.LetStatement)
	tern, ok := stmt.Value.(*ast.TernaryExpression)
	if !ok {
		t.Fatalf("expected *ast.TernaryExpression, got=%T", stmt.Value)
	}
	if tern.Condition.String() != "active" {
		t.Fatalf("expected condition 'active', got=%s", tern.Condition.String())
	}

	// 2. Null-coalescing and nullish assignment
	inputCoalesce := `
	let val = x ?? 42;
	x ??= 10;
	`
	p = New(lexer.New(inputCoalesce))
	prog = p.ParseProgram()
	checkParserErrors(t, p)
	stmt1 := prog.Statements[0].(*ast.LetStatement)
	if _, ok := stmt1.Value.(*ast.NullCoalesceExpression); !ok {
		t.Fatalf("expected *ast.NullCoalesceExpression, got=%T", stmt1.Value)
	}
	stmt2 := prog.Statements[1].(*ast.ReassignStatement)
	if stmt2.Operator != "??=" {
		t.Fatalf("expected operator '??=', got=%s", stmt2.Operator)
	}

	// 3. Optional chaining
	inputOpt := `
	let a = user?.profile?.name;
	let b = matrix?.[row];
	let c = service?.start(1, 2);
	`
	p = New(lexer.New(inputOpt))
	prog = p.ParseProgram()
	checkParserErrors(t, p)
	optStmt1 := prog.Statements[0].(*ast.LetStatement)
	if _, ok := optStmt1.Value.(*ast.OptionalDotExpression); !ok {
		t.Fatalf("expected *ast.OptionalDotExpression, got=%T", optStmt1.Value)
	}
	optStmt2 := prog.Statements[1].(*ast.LetStatement)
	if _, ok := optStmt2.Value.(*ast.OptionalIndexExpression); !ok {
		t.Fatalf("expected *ast.OptionalIndexExpression, got=%T", optStmt2.Value)
	}

	// 4. String interpolation
	inputInterp := `let msg = f"Hello, {name}! Count: {count + 1}.";`
	p = New(lexer.New(inputInterp))
	prog = p.ParseProgram()
	checkParserErrors(t, p)
	interpStmt := prog.Statements[0].(*ast.LetStatement)
	interp, ok := interpStmt.Value.(*ast.InterpolatedStringLiteral)
	if !ok {
		t.Fatalf("expected *ast.InterpolatedStringLiteral, got=%T", interpStmt.Value)
	}
	if len(interp.Parts) != 5 {
		t.Fatalf("expected 5 parts, got=%d", len(interp.Parts))
	}
}

func TestNestedTernaryConsequenceParsing(t *testing.T) {
	// Nested ternary in consequence: a ? b ? c : d : e
	input := `let res = a ? b ? 1 : 2 : 3;`
	p := New(lexer.New(input))
	prog := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := prog.Statements[0].(*ast.LetStatement)
	outerTern, ok := stmt.Value.(*ast.TernaryExpression)
	if !ok {
		t.Fatalf("expected outer *ast.TernaryExpression, got=%T", stmt.Value)
	}

	innerTern, ok := outerTern.Consequence.(*ast.TernaryExpression)
	if !ok {
		t.Fatalf("expected inner *ast.TernaryExpression in consequence, got=%T", outerTern.Consequence)
	}

	if innerTern.Condition.String() != "b" {
		t.Errorf("expected inner condition 'b', got=%s", innerTern.Condition.String())
	}
	if outerTern.Alternative.String() != "3" {
		t.Errorf("expected outer alternative '3', got=%s", outerTern.Alternative.String())
	}
}

func TestInterpolatedStringEscapeSequences(t *testing.T) {
	input := `let msg = f"Line1\nLine2\tTab\x41\u0042\{escaped\}";`
	p := New(lexer.New(input))
	prog := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := prog.Statements[0].(*ast.LetStatement)
	interp, ok := stmt.Value.(*ast.InterpolatedStringLiteral)
	if !ok {
		t.Fatalf("expected *ast.InterpolatedStringLiteral, got=%T", stmt.Value)
	}

	if len(interp.Parts) != 1 {
		t.Fatalf("expected 1 part (literal with escaped braces), got=%d", len(interp.Parts))
	}

	strLit := interp.Parts[0].(*ast.StringLiteral)
	expected := "Line1\nLine2\tTabAB{escaped}"
	if strLit.Value != expected {
		t.Errorf("expected unescaped string %q, got %q", expected, strLit.Value)
	}
}
