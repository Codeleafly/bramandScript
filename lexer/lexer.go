package lexer

import (
	"bramandscript/token"
	"strings"
)

type Lexer struct {
	input        string
	position     int  // current byte offset
	readPosition int  // next byte offset
	ch           byte // current byte
	line         int  // line number (1-based)
	col          int  // col number (1-based)
}

func New(input string) *Lexer {
	l := &Lexer{
		input: input,
		line:  1,
		col:   0,
	}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	l.col++
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	l.skipWhitespaceAndComments()

	curLine := l.line
	curCol := l.col

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.EQ, Literal: string(ch) + string(l.ch), Line: curLine, Col: curCol}
		} else if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.ARROW, Literal: string(ch) + string(l.ch), Line: curLine, Col: curCol}
		} else {
			tok = newToken(token.ASSIGN, l.ch, curLine, curCol)
		}
	case '+':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.PLUS_ASSIGN, Literal: string(ch) + string(l.ch), Line: curLine, Col: curCol}
		} else {
			tok = newToken(token.PLUS, l.ch, curLine, curCol)
		}
	case '-':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.MINUS_ASSIGN, Literal: string(ch) + string(l.ch), Line: curLine, Col: curCol}
		} else {
			tok = newToken(token.MINUS, l.ch, curLine, curCol)
		}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.NOT_EQ, Literal: string(ch) + string(l.ch), Line: curLine, Col: curCol}
		} else {
			tok = newToken(token.BANG, l.ch, curLine, curCol)
		}
	case '*':
		if l.peekChar() == '*' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.POWER, Literal: string(ch) + string(l.ch), Line: curLine, Col: curCol}
		} else if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.MUL_ASSIGN, Literal: string(ch) + string(l.ch), Line: curLine, Col: curCol}
		} else {
			tok = newToken(token.ASTERISK, l.ch, curLine, curCol)
		}
	case '/':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.DIV_ASSIGN, Literal: string(ch) + string(l.ch), Line: curLine, Col: curCol}
		} else {
			tok = newToken(token.SLASH, l.ch, curLine, curCol)
		}
	case '%':
		tok = newToken(token.MODULO, l.ch, curLine, curCol)
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.LTE, Literal: string(ch) + string(l.ch), Line: curLine, Col: curCol}
		} else if l.peekChar() == '-' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.SEND_RECV, Literal: string(ch) + string(l.ch), Line: curLine, Col: curCol}
		} else {
			tok = newToken(token.LT, l.ch, curLine, curCol)
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.GTE, Literal: string(ch) + string(l.ch), Line: curLine, Col: curCol}
		} else {
			tok = newToken(token.GT, l.ch, curLine, curCol)
		}
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.AND, Literal: string(ch) + string(l.ch), Line: curLine, Col: curCol}
		} else {
			tok = newToken(token.ILLEGAL, l.ch, curLine, curCol)
		}
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.OR, Literal: string(ch) + string(l.ch), Line: curLine, Col: curCol}
		} else {
			tok = newToken(token.ILLEGAL, l.ch, curLine, curCol)
		}
	case ';':
		tok = newToken(token.SEMICOLON, l.ch, curLine, curCol)
	case ':':
		tok = newToken(token.COLON, l.ch, curLine, curCol)
	case ',':
		tok = newToken(token.COMMA, l.ch, curLine, curCol)
	case '.':
		tok = newToken(token.DOT, l.ch, curLine, curCol)
	case '(':
		tok = newToken(token.LPAREN, l.ch, curLine, curCol)
	case ')':
		tok = newToken(token.RPAREN, l.ch, curLine, curCol)
	case '{':
		tok = newToken(token.LBRACE, l.ch, curLine, curCol)
	case '}':
		tok = newToken(token.RBRACE, l.ch, curLine, curCol)
	case '[':
		tok = newToken(token.LBRACKET, l.ch, curLine, curCol)
	case ']':
		tok = newToken(token.RBRACKET, l.ch, curLine, curCol)
	case '"':
		str, ok := l.readString()
		if !ok {
			return token.Token{Type: token.ILLEGAL, Literal: "unterminated string", Line: curLine, Col: curCol}
		}
		return token.Token{Type: token.STRING, Literal: str, Line: curLine, Col: curCol}
	case 0:
		tok.Literal = ""
		tok.Type = token.EOF
		tok.Line = curLine
		tok.Col = curCol
	default:
		if isLetter(l.ch) {
			lit := l.readIdentifier()
			tokType := token.LookupIdent(lit)
			return token.Token{Type: tokType, Literal: lit, Line: curLine, Col: curCol}
		} else if isDigit(l.ch) {
			lit, isFloat := l.readNumber()
			var tokType token.TokenType = token.INT
			if isFloat {
				tokType = token.FLOAT
			}
			return token.Token{Type: tokType, Literal: lit, Line: curLine, Col: curCol}
		} else {
			tok = newToken(token.ILLEGAL, l.ch, curLine, curCol)
		}
	}

	l.readChar()
	return tok
}

func newToken(tokenType token.TokenType, ch byte, line, col int) token.Token {
	return token.Token{Type: tokenType, Literal: string(ch), Line: line, Col: col}
}

func (l *Lexer) skipWhitespaceAndComments() {
	for {
		if l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
			l.readChar()
			continue
		}
		if l.ch == '\n' {
			l.line++
			l.col = 0
			l.readChar()
			continue
		}
		// Single line comment: // ...
		if l.ch == '/' && l.peekChar() == '/' {
			for l.ch != '\n' && l.ch != 0 {
				l.readChar()
			}
			continue
		}
		// Multi-line comment: /* ... */
		if l.ch == '/' && l.peekChar() == '*' {
			l.readChar()
			l.readChar()
			for {
				if l.ch == 0 {
					break
				}
				if l.ch == '\n' {
					l.line++
					l.col = 0
				}
				if l.ch == '*' && l.peekChar() == '/' {
					l.readChar()
					l.readChar()
					break
				}
				l.readChar()
			}
			continue
		}
		break
	}
}

func (l *Lexer) readIdentifier() string {
	startPos := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[startPos:l.position]
}

func (l *Lexer) readNumber() (string, bool) {
	startPos := l.position
	isFloat := false

	for isDigit(l.ch) {
		l.readChar()
	}

	if l.ch == '.' && isDigit(l.peekChar()) {
		isFloat = true
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	return l.input[startPos:l.position], isFloat
}

func (l *Lexer) readString() (string, bool) {
	var sb strings.Builder
	l.readChar()

	for {
		if l.ch == 0 {
			return "", false
		}
		if l.ch == '"' {
			l.readChar()
			break
		}
		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case 'e':
				sb.WriteByte(0x1b)
			case '0':
				if l.peekChar() == '3' {
					l.readChar()
					if l.peekChar() == '3' {
						l.readChar()
						sb.WriteByte(0x1b)
					} else {
						sb.WriteByte(0)
					}
				} else {
					sb.WriteByte(0)
				}
			case '\\':
				sb.WriteByte('\\')
			case '"':
				sb.WriteByte('"')
			default:
				sb.WriteByte('\\')
				sb.WriteByte(l.ch)
			}
			l.readChar()
			continue
		}
		sb.WriteByte(l.ch)
		l.readChar()
	}

	return sb.String(), true
}

func isLetter(ch byte) bool {
	return ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z') || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}
