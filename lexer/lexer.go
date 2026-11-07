package lexer

import (
	"bramandscript/token"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Lexer struct {
	input                  string
	position               int  // current byte offset of ch
	readPosition           int  // next byte offset
	ch                     rune // current rune (0 for EOF)
	chWidth                int  // byte width of current rune
	line                   int  // line number (1-based)
	col                    int  // col number (1-based, rune-based)
	hasUnterminatedComment bool
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
		l.chWidth = 0
		l.position = l.readPosition
	} else {
		r, width := utf8.DecodeRuneInString(l.input[l.readPosition:])
		l.ch = r
		l.chWidth = width
		l.position = l.readPosition
		l.readPosition += width
		l.col++
	}
}

func (l *Lexer) peekChar() rune {
	if l.readPosition >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.readPosition:])
	return r
}

func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	l.skipWhitespaceAndComments()

	curLine := l.line
	curCol := l.col

	if l.hasUnterminatedComment {
		l.hasUnterminatedComment = false
		return token.Token{Type: token.ILLEGAL, Literal: "unterminated block comment", Line: curLine, Col: curCol}
	}

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
		} else if l.peekChar() == '<' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.SHIFT_LEFT, Literal: string(ch) + string(l.ch), Line: curLine, Col: curCol}
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
		} else if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.SHIFT_RIGHT, Literal: string(ch) + string(l.ch), Line: curLine, Col: curCol}
		} else {
			tok = newToken(token.GT, l.ch, curLine, curCol)
		}
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.AND, Literal: string(ch) + string(l.ch), Line: curLine, Col: curCol}
		} else {
			tok = newToken(token.BIT_AND, l.ch, curLine, curCol)
		}
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.OR, Literal: string(ch) + string(l.ch), Line: curLine, Col: curCol}
		} else if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.PIPE_OP, Literal: string(ch) + string(l.ch), Line: curLine, Col: curCol}
		} else {
			tok = newToken(token.BIT_OR, l.ch, curLine, curCol)
		}
	case '^':
		tok = newToken(token.BIT_XOR, l.ch, curLine, curCol)
	case '~':
		tok = newToken(token.BIT_NOT, l.ch, curLine, curCol)
	case ';':
		tok = newToken(token.SEMICOLON, l.ch, curLine, curCol)
	case ':':
		tok = newToken(token.COLON, l.ch, curLine, curCol)
	case ',':
		tok = newToken(token.COMMA, l.ch, curLine, curCol)
	case '.':
		if l.peekChar() == '.' {
			nextPos := l.readPosition + 1
			if nextPos < len(l.input) && l.input[nextPos] == '.' {
				l.readChar()
				l.readChar()
				tok = token.Token{Type: token.ELLIPSIS, Literal: "...", Line: curLine, Col: curCol}
			} else {
				tok = newToken(token.DOT, l.ch, curLine, curCol)
			}
		} else {
			tok = newToken(token.DOT, l.ch, curLine, curCol)
		}
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
	case '`':
		str, ok := l.readFormatString('`')
		if !ok {
			return token.Token{Type: token.ILLEGAL, Literal: "unterminated template string", Line: curLine, Col: curCol}
		}
		return token.Token{Type: token.FORMAT_STRING, Literal: str, Line: curLine, Col: curCol}
	case '?':
		if l.peekChar() == '?' {
			ch := l.ch
			l.readChar()
			if l.peekChar() == '=' {
				l.readChar()
				tok = token.Token{Type: token.COALESCE_ASSIGN, Literal: "??=", Line: curLine, Col: curCol}
			} else {
				tok = token.Token{Type: token.COALESCE, Literal: string(ch) + string(l.ch), Line: curLine, Col: curCol}
			}
		} else if l.peekChar() == '.' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.QUESTION_DOT, Literal: string(ch) + string(l.ch), Line: curLine, Col: curCol}
		} else {
			tok = newToken(token.QUESTION, l.ch, curLine, curCol)
		}
	case 0:
		tok.Literal = ""
		tok.Type = token.EOF
		tok.Line = curLine
		tok.Col = curCol
	default:
		if (l.ch == 'f' || l.ch == 'F') && l.peekChar() == '"' {
			l.readChar() // consume 'f' or 'F'
			str, ok := l.readFormatString('"')
			if !ok {
				return token.Token{Type: token.ILLEGAL, Literal: "unterminated format string", Line: curLine, Col: curCol}
			}
			return token.Token{Type: token.FORMAT_STRING, Literal: str, Line: curLine, Col: curCol}
		}
		if isIdentStart(l.ch) {
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

func newToken(tokenType token.TokenType, ch rune, line, col int) token.Token {
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
					l.hasUnterminatedComment = true
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
	for isIdentPart(l.ch) {
		l.readChar()
	}
	return l.input[startPos:l.position]
}

func (l *Lexer) readNumber() (string, bool) {
	startPos := l.position
	isFloat := false

	if l.ch == '0' {
		next := l.peekChar()
		if next == 'x' || next == 'X' {
			l.readChar() // consume '0'
			l.readChar() // consume 'x'/'X'
			for isHexDigit(l.ch) {
				l.readChar()
			}
			return l.input[startPos:l.position], false
		} else if next == 'b' || next == 'B' {
			l.readChar() // consume '0'
			l.readChar() // consume 'b'/'B'
			for l.ch == '0' || l.ch == '1' {
				l.readChar()
			}
			return l.input[startPos:l.position], false
		} else if next == 'o' || next == 'O' {
			l.readChar() // consume '0'
			l.readChar() // consume 'o'/'O'
			for l.ch >= '0' && l.ch <= '7' {
				l.readChar()
			}
			return l.input[startPos:l.position], false
		}
	}

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

	if l.ch == 'e' || l.ch == 'E' {
		next := l.peekChar()
		hasExp := false
		if isDigit(next) {
			hasExp = true
		} else if next == '+' || next == '-' {
			if l.readPosition+1 < len(l.input) {
				afterSign, _ := utf8.DecodeRuneInString(l.input[l.readPosition+1:])
				if isDigit(afterSign) {
					hasExp = true
				}
			}
		}
		if hasExp {
			isFloat = true
			l.readChar() // consume 'e' or 'E'
			if l.ch == '+' || l.ch == '-' {
				l.readChar() // consume '+' or '-'
			}
			for isDigit(l.ch) {
				l.readChar()
			}
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
		if l.ch == '\n' {
			l.line++
			l.col = 0
		}
		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				sb.WriteRune('\n')
			case 't':
				sb.WriteRune('\t')
			case 'r':
				sb.WriteRune('\r')
			case 'e':
				sb.WriteRune(0x1b)
			case '0':
				if l.peekChar() == '3' {
					secondPos := l.readPosition + 1
					if secondPos < len(l.input) {
						rSecond, _ := utf8.DecodeRuneInString(l.input[secondPos:])
						if rSecond == '3' {
							l.readChar()
							l.readChar()
							sb.WriteRune(0x1b)
							break
						}
					}
				}
				sb.WriteRune(0)
			case 'x':
				hexStr := ""
				for i := 0; i < 2; i++ {
					p := l.peekChar()
					if (p >= '0' && p <= '9') || (p >= 'a' && p <= 'f') || (p >= 'A' && p <= 'F') {
						l.readChar()
						hexStr += string(l.ch)
					} else {
						break
					}
				}
				if len(hexStr) == 2 {
					val, err := strconv.ParseInt(hexStr, 16, 32)
					if err == nil {
						sb.WriteRune(rune(val))
					} else {
						sb.WriteString("\\x" + hexStr)
					}
				} else {
					sb.WriteString("\\x" + hexStr)
				}
			case 'u':
				hexStr := ""
				for i := 0; i < 4; i++ {
					p := l.peekChar()
					if (p >= '0' && p <= '9') || (p >= 'a' && p <= 'f') || (p >= 'A' && p <= 'F') {
						l.readChar()
						hexStr += string(l.ch)
					} else {
						break
					}
				}
				if len(hexStr) == 4 {
					val, err := strconv.ParseInt(hexStr, 16, 32)
					if err == nil {
						sb.WriteRune(rune(val))
					} else {
						sb.WriteString("\\u" + hexStr)
					}
				} else {
					sb.WriteString("\\u" + hexStr)
				}
			case '\\':
				sb.WriteRune('\\')
			case '"':
				sb.WriteRune('"')
			default:
				sb.WriteRune('\\')
				sb.WriteRune(l.ch)
			}
			l.readChar()
			continue
		}
		sb.WriteRune(l.ch)
		l.readChar()
	}

	return sb.String(), true
}

func (l *Lexer) readFormatString(quote rune) (string, bool) {
	var sb strings.Builder
	l.readChar() // consume opening quote

	braceDepth := 0
	inString := false
	var strQuote rune

	for {
		if l.ch == 0 {
			return "", false
		}
		if !inString && braceDepth == 0 && l.ch == quote {
			l.readChar() // consume closing quote
			break
		}
		if l.ch == '\n' {
			l.line++
			l.col = 0
		}
		if l.ch == '\\' {
			sb.WriteRune(l.ch)
			l.readChar()
			if l.ch == 0 {
				return "", false
			}
			sb.WriteRune(l.ch)
			l.readChar()
			continue
		}

		if inString {
			if l.ch == strQuote {
				inString = false
			}
			sb.WriteRune(l.ch)
			l.readChar()
			continue
		}

		if braceDepth > 0 {
			if l.ch == '"' || l.ch == '\'' {
				inString = true
				strQuote = l.ch
				sb.WriteRune(l.ch)
				l.readChar()
				continue
			}
			if l.ch == '{' {
				braceDepth++
			} else if l.ch == '}' {
				braceDepth--
			}
			sb.WriteRune(l.ch)
			l.readChar()
			continue
		}

		// At top-level format string (outside embedded expressions)
		if l.ch == '{' && l.peekChar() == '{' {
			sb.WriteString("{{")
			l.readChar()
			l.readChar()
			continue
		}
		if l.ch == '}' && l.peekChar() == '}' {
			sb.WriteString("}}")
			l.readChar()
			l.readChar()
			continue
		}
		if l.ch == '{' {
			braceDepth++
			sb.WriteRune(l.ch)
			l.readChar()
			continue
		}

		sb.WriteRune(l.ch)
		l.readChar()
	}

	return sb.String(), true
}

func isIdentStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func isIdentPart(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsMark(r) || unicode.IsDigit(r) || r == '_'
}

func isLetter(r rune) bool {
	return isIdentStart(r)
}

func isDigit(r rune) bool {
	return '0' <= r && r <= '9'
}

func isHexDigit(r rune) bool {
	return ('0' <= r && r <= '9') || ('a' <= r && r <= 'f') || ('A' <= r && r <= 'F')
}
