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

func TestNextTokenDay4(t *testing.T) {
	input := `
	spawn worker(10);
	ch <- 42;
	let val = <-ch;
	`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.SPAWN, "spawn"},
		{token.IDENT, "worker"},
		{token.LPAREN, "("},
		{token.INT, "10"},
		{token.RPAREN, ")"},
		{token.SEMICOLON, ";"},

		{token.IDENT, "ch"},
		{token.SEND_RECV, "<-"},
		{token.INT, "42"},
		{token.SEMICOLON, ";"},

		{token.LET, "let"},
		{token.IDENT, "val"},
		{token.ASSIGN, "="},
		{token.SEND_RECV, "<-"},
		{token.IDENT, "ch"},
		{token.SEMICOLON, ";"},

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

func TestUnicodeLexing(t *testing.T) {
	input := `
	// टिप्पणि: देवनागरी चर
	let संदेश = "नमस्ते";
	const संख्या_१ = 100;
	let 変数 = "日本語";
	let α_β = 3.14;
	let café = "coffee";
	let escape_str = "\u0041\u0042\u0928";
	`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.LET, "let"},
		{token.IDENT, "संदेश"},
		{token.ASSIGN, "="},
		{token.STRING, "नमस्ते"},
		{token.SEMICOLON, ";"},

		{token.CONST, "const"},
		{token.IDENT, "संख्या_१"},
		{token.ASSIGN, "="},
		{token.INT, "100"},
		{token.SEMICOLON, ";"},

		{token.LET, "let"},
		{token.IDENT, "変数"},
		{token.ASSIGN, "="},
		{token.STRING, "日本語"},
		{token.SEMICOLON, ";"},

		{token.LET, "let"},
		{token.IDENT, "α_β"},
		{token.ASSIGN, "="},
		{token.FLOAT, "3.14"},
		{token.SEMICOLON, ";"},

		{token.LET, "let"},
		{token.IDENT, "café"},
		{token.ASSIGN, "="},
		{token.STRING, "coffee"},
		{token.SEMICOLON, ";"},

		{token.LET, "let"},
		{token.IDENT, "escape_str"},
		{token.ASSIGN, "="},
		{token.STRING, "ABन"},
		{token.SEMICOLON, ";"},

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

func TestUnicodeColumnTracking(t *testing.T) {
	input := `let संदेश = 42;`
	l := New(input)

	tok1 := l.NextToken() // let
	if tok1.Line != 1 || tok1.Col != 1 {
		t.Fatalf("tok1 'let' expected line 1, col 1, got line %d, col %d", tok1.Line, tok1.Col)
	}

	tok2 := l.NextToken() // संदेश
	if tok2.Line != 1 || tok2.Col != 5 {
		t.Fatalf("tok2 'संदेश' expected line 1, col 5, got line %d, col %d", tok2.Line, tok2.Col)
	}

	tok3 := l.NextToken() // =
	// "let " is 4 runes, "संदेश" is 5 runes (स + ं + द + े + श), " " is 1 rune -> 4 + 5 + 1 = 10 runes. '=' is at col 11
	if tok3.Line != 1 || tok3.Col != 11 {
		t.Fatalf("tok3 '=' expected line 1, col 11, got line %d, col %d", tok3.Line, tok3.Col)
	}

	tok4 := l.NextToken() // 42
	if tok4.Line != 1 || tok4.Col != 13 {
		t.Fatalf("tok4 '42' expected line 1, col 13, got line %d, col %d", tok4.Line, tok4.Col)
	}
}
