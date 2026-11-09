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
	IDENT         = "IDENT"
	INT           = "INT"
	FLOAT         = "FLOAT"
	STRING        = "STRING"
	FORMAT_STRING = "FORMAT_STRING"

	// Arithmetic Operators
	ASSIGN   = "="
	PLUS     = "+"
	MINUS    = "-"
	ASTERISK = "*"
	SLASH    = "/"
	MODULO   = "%"
	POWER    = "**"

	// Compound Assignment
	PLUS_ASSIGN     = "+="
	MINUS_ASSIGN    = "-="
	MUL_ASSIGN      = "*="
	DIV_ASSIGN      = "/="
	COALESCE_ASSIGN = "??="

	// Logical & Conditional Operators
	BANG         = "!"
	AND          = "&&"
	OR           = "||"
	QUESTION     = "?"
	COALESCE     = "??"
	QUESTION_DOT = "?."

	// Bitwise Operators
	BIT_AND     = "&"
	BIT_OR      = "|"
	BIT_XOR     = "^"
	BIT_NOT     = "~"
	SHIFT_LEFT  = "<<"
	SHIFT_RIGHT = ">>"

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
	DOT       = "."
	ARROW     = "=>"
	SEND_RECV = "<-"
	PIPE_OP   = "|>"
	ELLIPSIS  = "..."

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
	MATCH    = "MATCH"
	CASE     = "CASE"
	DEFAULT  = "DEFAULT"
	IMPORT   = "IMPORT"
	STRUCT   = "STRUCT"
	AS       = "AS"
	SPAWN    = "SPAWN"
	ENUM     = "ENUM"
	EMBED    = "EMBED"
	YIELD    = "YIELD"
	TRY      = "TRY"
	CATCH    = "CATCH"
	FINALLY  = "FINALLY"
	THROW    = "THROW"
	DEFER    = "DEFER"
	ACTOR    = "ACTOR"
	ASYNC    = "ASYNC"
	AWAIT    = "AWAIT"
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
	"match":    MATCH,
	"case":     CASE,
	"default":  DEFAULT,
	"import":   IMPORT,
	"struct":   STRUCT,
	"as":       AS,
	"spawn":    SPAWN,
	"enum":     ENUM,
	"embed":    EMBED,
	"yield":    YIELD,
	"try":      TRY,
	"catch":    CATCH,
	"finally":  FINALLY,
	"throw":    THROW,
	"defer":    DEFER,
	"actor":    ACTOR,
	"async":    ASYNC,
	"await":    AWAIT,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

func IsKeyword(tok TokenType) bool {
	for _, kw := range keywords {
		if kw == tok {
			return true
		}
	}
	return false
}
