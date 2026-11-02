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
