package lexer

import (
	"bramandscript/token"
	"testing"
)

func TestNextToken(t *testing.T) {
	input := `
	// BramandScript Day 1 Lexer
	let x = 100;
	const pi = 3.14159;
	let msg = "Welcome to Brahmand!\n";
	
	/* Multi-line comment */
	if (x >= 50 && pi == 3.14159) {
		x += 5;
		x = 2 ** 3;
	} else {
		let n = null;
	}
	`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.LET, "let"},
		{token.IDENT, "x"},
		{token.ASSIGN, "="},
		{token.INT, "100"},
		{token.SEMICOLON, ";"},

		{token.CONST, "const"},
		{token.IDENT, "pi"},
		{token.ASSIGN, "="},
		{token.FLOAT, "3.14159"},
		{token.SEMICOLON, ";"},

		{token.LET, "let"},
		{token.IDENT, "msg"},
		{token.ASSIGN, "="},
		{token.STRING, "Welcome to Brahmand!\n"},
		{token.SEMICOLON, ";"},

		{token.IF, "if"},
		{token.LPAREN, "("},
		{token.IDENT, "x"},
		{token.GTE, ">="},
		{token.INT, "50"},
		{token.AND, "&&"},
		{token.IDENT, "pi"},
		{token.EQ, "=="},
		{token.FLOAT, "3.14159"},
		{token.RPAREN, ")"},
		{token.LBRACE, "{"},

		{token.IDENT, "x"},
		{token.PLUS_ASSIGN, "+="},
		{token.INT, "5"},
		{token.SEMICOLON, ";"},

		{token.IDENT, "x"},
		{token.ASSIGN, "="},
		{token.INT, "2"},
		{token.POWER, "**"},
		{token.INT, "3"},
		{token.SEMICOLON, ";"},

		{token.RBRACE, "}"},
		{token.ELSE, "else"},
		{token.LBRACE, "{"},

		{token.LET, "let"},
		{token.IDENT, "n"},
		{token.ASSIGN, "="},
		{token.NULL, "null"},
		{token.SEMICOLON, ";"},

		{token.RBRACE, "}"},
		{token.EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("test[%d] - wrong type. expected=%q, got=%q", i, tt.expectedType, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("test[%d] - wrong literal. expected=%q, got=%q", i, tt.expectedLiteral, tok.Literal)
		}
	}
}
