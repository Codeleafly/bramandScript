package token

type TokenType string

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Col     int
}

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// Identifiers & Literals
	IDENT  = "IDENT"
	INT    = "INT"
	FLOAT  = "FLOAT"
	STRING = "STRING"

	// Arithmetic Operators
	ASSIGN   = "="
	PLUS     = "+"
	MINUS    = "-"
	ASTERISK = "*"
	SLASH    = "/"
	MODULO   = "%"
	POWER    = "**"

	// Compound Assignment
	PLUS_ASSIGN  = "+="
	MINUS_ASSIGN = "-="
	MUL_ASSIGN   = "*="
	DIV_ASSIGN   = "/="

	// Logical Operators
	BANG = "!"
	AND  = "&&"
	OR   = "||"

	// Comparison Operators
	EQ     = "=="
	NOT_EQ = "!="
	LT     = "<"
	GT     = ">"
	LTE    = "<="
	GTE    = ">="

	// Delimiters
	COMMA     = ","
	SEMICOLON = ";"
	COLON     = ":"

	LPAREN   = "("
	RPAREN   = ")"
	LBRACE   = "{"
	RBRACE   = "}"
	LBRACKET = "["
	RBRACKET = "]"

	// Keywords
	FUNCTION = "FUNCTION"
	LET      = "LET"
	CONST    = "CONST"
	TRUE     = "TRUE"
	FALSE    = "FALSE"
	IF       = "IF"
	ELSE     = "ELSE"
	RETURN   = "RETURN"
	WHILE    = "WHILE"
	FOR      = "FOR"
	IN       = "IN"
	NULL     = "NULL"
	BREAK    = "BREAK"
	CONTINUE = "CONTINUE"
)

var keywords = map[string]TokenType{
	"fn":       FUNCTION,
	"let":      LET,
	"const":    CONST,
	"true":     TRUE,
	"false":    FALSE,
	"if":       IF,
	"else":     ELSE,
	"return":   RETURN,
	"while":    WHILE,
	"for":      FOR,
	"in":       IN,
	"null":     NULL,
	"break":    BREAK,
	"continue": CONTINUE,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
