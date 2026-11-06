package parser

import (
	"bramandscript/ast"
	"bramandscript/lexer"
	"bramandscript/token"
	"fmt"
	"math/big"
	"strconv"
)

// Precedence levels
const (
	_ int = iota
	LOWEST
	PIPE        // |>
	SEND        // <-
	LOGICAL_OR  // ||
	LOGICAL_AND // &&
	BIT_OR      // |
	BIT_XOR     // ^
	BIT_AND     // &
	EQUALS      // == or !=
	LESSGREATER // > or < or >= or <=
	SHIFT       // << or >>
	SUM         // + or -
	PRODUCT     // * or / or %
	POWER       // **
	PREFIX      // -X or !X
	CALL        // myFunction(X)
	INDEX       // array[index]
)

var precedences = map[token.TokenType]int{
	token.PIPE_OP:     PIPE,
	token.SEND_RECV:   SEND,
	token.OR:          LOGICAL_OR,
	token.AND:         LOGICAL_AND,
	token.BIT_OR:      BIT_OR,
	token.BIT_XOR:     BIT_XOR,
	token.BIT_AND:     BIT_AND,
	token.EQ:          EQUALS,
	token.NOT_EQ:      EQUALS,
	token.LT:          LESSGREATER,
	token.GT:          LESSGREATER,
	token.LTE:         LESSGREATER,
	token.GTE:         LESSGREATER,
	token.SHIFT_LEFT:  SHIFT,
	token.SHIFT_RIGHT: SHIFT,
	token.PLUS:        SUM,
	token.MINUS:     SUM,
	token.SLASH:     PRODUCT,
	token.ASTERISK:  PRODUCT,
	token.MODULO:    PRODUCT,
	token.POWER:     POWER,
	token.LPAREN:    CALL,
	token.LBRACKET:  INDEX,
	token.DOT:       INDEX,
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

type SyntaxError struct {
	Line    int
	Col     int
	Message string
}

type Parser struct {
	l *lexer.Lexer

	curToken  token.Token
	peekToken token.Token

	errors       []string
	SyntaxErrors []SyntaxError

	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn

	destructCounter int
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:            l,
		errors:       []string{},
		SyntaxErrors: []SyntaxError{},
	}

	p.prefixParseFns = make(map[token.TokenType]prefixParseFn)
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.INT, p.parseIntegerLiteral)
	p.registerPrefix(token.FLOAT, p.parseFloatLiteral)
	p.registerPrefix(token.STRING, p.parseStringLiteral)
	p.registerPrefix(token.TRUE, p.parseBooleanLiteral)
	p.registerPrefix(token.FALSE, p.parseBooleanLiteral)
	p.registerPrefix(token.NULL, p.parseNullLiteral)
	p.registerPrefix(token.BANG, p.parsePrefixExpression)
	p.registerPrefix(token.MINUS, p.parsePrefixExpression)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.IF, p.parseIfExpression)
	p.registerPrefix(token.FUNCTION, p.parseFunctionLiteral)
	p.registerPrefix(token.LBRACKET, p.parseArrayLiteral)
	p.registerPrefix(token.LBRACE, p.parseHashLiteral)
	p.registerPrefix(token.MATCH, p.parseMatchExpression)
	p.registerPrefix(token.SPAWN, p.parseSpawnExpression)
	p.registerPrefix(token.SEND_RECV, p.parsePrefixExpression)
	p.registerPrefix(token.BIT_NOT, p.parsePrefixExpression)
	p.registerPrefix(token.YIELD, p.parseYieldExpression)

	p.infixParseFns = make(map[token.TokenType]infixParseFn)
	p.registerInfix(token.PIPE_OP, p.parsePipeExpression)
	p.registerInfix(token.PLUS, p.parseInfixExpression)
	p.registerInfix(token.MINUS, p.parseInfixExpression)
	p.registerInfix(token.SLASH, p.parseInfixExpression)
	p.registerInfix(token.ASTERISK, p.parseInfixExpression)
	p.registerInfix(token.MODULO, p.parseInfixExpression)
	p.registerInfix(token.POWER, p.parseInfixExpression)
	p.registerInfix(token.BIT_AND, p.parseInfixExpression)
	p.registerInfix(token.BIT_OR, p.parseInfixExpression)
	p.registerInfix(token.BIT_XOR, p.parseInfixExpression)
	p.registerInfix(token.SHIFT_LEFT, p.parseInfixExpression)
	p.registerInfix(token.SHIFT_RIGHT, p.parseInfixExpression)
	p.registerInfix(token.EQ, p.parseInfixExpression)
	p.registerInfix(token.NOT_EQ, p.parseInfixExpression)
	p.registerInfix(token.LT, p.parseInfixExpression)
	p.registerInfix(token.GT, p.parseInfixExpression)
	p.registerInfix(token.LTE, p.parseInfixExpression)
	p.registerInfix(token.GTE, p.parseInfixExpression)
	p.registerInfix(token.AND, p.parseInfixExpression)
	p.registerInfix(token.OR, p.parseInfixExpression)
	p.registerInfix(token.SEND_RECV, p.parseInfixExpression)
	p.registerInfix(token.LPAREN, p.parseCallExpression)
	p.registerInfix(token.LBRACKET, p.parseIndexExpression)
	p.registerInfix(token.DOT, p.parseDotExpression)

	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) peekError(t token.TokenType) {
	msg := fmt.Sprintf("line %d, col %d: expected next token to be %s, got %s instead",
		p.peekToken.Line, p.peekToken.Col, t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
	p.SyntaxErrors = append(p.SyntaxErrors, SyntaxError{
		Line:    p.peekToken.Line,
		Col:     p.peekToken.Col,
		Message: fmt.Sprintf("expected '%s', but found '%s'", t, p.peekToken.Literal),
	})
}

func (p *Parser) registerPrefix(tokenType token.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType token.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}

	for !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

func (p *Parser) parseDotExpression(left ast.Expression) ast.Expression {
	dotToken := p.curToken
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	field := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	return &ast.DotExpression{Token: dotToken, Left: left, Field: field}
}

func (p *Parser) parseMatchExpression() ast.Expression {
	expression := &ast.MatchExpression{Token: p.curToken}

	p.nextToken()
	expression.Subject = p.parseExpression(LOWEST)

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	p.nextToken()

	hasDefault := false

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		clause := &ast.MatchClause{Values: []ast.Expression{}}

		if p.curTokenIs(token.DEFAULT) {
			if hasDefault {
				p.errors = append(p.errors, fmt.Sprintf("multiple default clauses in match expression at line %d, col %d", p.curToken.Line, p.curToken.Col))
			}
			hasDefault = true
			if !p.expectPeek(token.ARROW) {
				return nil
			}
			if !p.expectPeek(token.LBRACE) {
				return nil
			}
			clause.Body = p.parseBlockStatement()
			expression.Clauses = append(expression.Clauses, clause)
			p.nextToken()
			continue
		}

		if p.curTokenIs(token.CASE) {
			p.nextToken()
		}

		val := p.parseExpression(LOWEST)
		clause.Values = append(clause.Values, val)

		for p.peekTokenIs(token.COMMA) {
			p.nextToken()
			if p.peekTokenIs(token.ARROW) {
				p.errors = append(p.errors, fmt.Sprintf("trailing comma without expression in match clause at line %d, col %d", p.curToken.Line, p.curToken.Col))
				break
			}
			p.nextToken()
			clause.Values = append(clause.Values, p.parseExpression(LOWEST))
		}

		if !p.expectPeek(token.ARROW) {
			return nil
		}

		if !p.expectPeek(token.LBRACE) {
			return nil
		}

		clause.Body = p.parseBlockStatement()
		expression.Clauses = append(expression.Clauses, clause)
		p.nextToken()
	}

	return expression
}

func (p *Parser) parseSpawnExpression() ast.Expression {
	expr := &ast.SpawnExpression{Token: p.curToken}
	p.nextToken()
	expr.Call = p.parseExpression(PREFIX)
	return expr
}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()
		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) noPrefixParseFnError(t token.TokenType) {
	msg := fmt.Sprintf("line %d, col %d: no prefix parse function for %s found",
		p.curToken.Line, p.curToken.Col, t)
	p.errors = append(p.errors, msg)
	p.SyntaxErrors = append(p.SyntaxErrors, SyntaxError{
		Line:    p.curToken.Line,
		Col:     p.curToken.Col,
		Message: fmt.Sprintf("unexpected token '%s'", p.curToken.Literal),
	})
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.curToken}
	val, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err == nil {
		lit.Value = val
		return lit
	}

	// Promote large integers to arbitrary-precision BigInt automatically
	if bi, ok := new(big.Int).SetString(p.curToken.Literal, 0); ok {
		lit.BigValue = bi
		return lit
	}

	msg := fmt.Sprintf("could not parse %q as integer", p.curToken.Literal)
	p.errors = append(p.errors, msg)
	p.SyntaxErrors = append(p.SyntaxErrors, SyntaxError{
		Line:    p.curToken.Line,
		Col:     p.curToken.Col,
		Message: msg,
	})
	return nil
}

func (p *Parser) parseFloatLiteral() ast.Expression {
	lit := &ast.FloatLiteral{Token: p.curToken}
	val, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as float", p.curToken.Literal)
		p.errors = append(p.errors, msg)
		p.SyntaxErrors = append(p.SyntaxErrors, SyntaxError{
			Line:    p.curToken.Line,
			Col:     p.curToken.Col,
			Message: msg,
		})
		return nil
	}
	lit.Value = val
	return lit
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseBooleanLiteral() ast.Expression {
	return &ast.BooleanLiteral{Token: p.curToken, Value: p.curTokenIs(token.TRUE)}
}

func (p *Parser) parseNullLiteral() ast.Expression {
	return &ast.NullLiteral{Token: p.curToken}
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	expression := &ast.PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}
	p.nextToken()
	expression.Right = p.parseExpression(PREFIX)
	return expression
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	if left == nil {
		return nil
	}
	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}
	precedence := p.curPrecedence()
	if p.curToken.Type == token.POWER {
		precedence--
	}
	p.nextToken()
	expression.Right = p.parseExpression(precedence)
	if expression.Right == nil {
		return nil
	}
	return expression
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()
	exp := p.parseExpression(LOWEST)
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	return exp
}

func (p *Parser) parseIfExpression() ast.Expression {
	expression := &ast.IfExpression{Token: p.curToken}

	hasParen := false
	if p.peekTokenIs(token.LPAREN) {
		hasParen = true
		p.nextToken()
	}

	p.nextToken()
	expression.Condition = p.parseExpression(LOWEST)

	if hasParen {
		if !p.expectPeek(token.RPAREN) {
			if p.peekTokenIs(token.LBRACE) {
				p.nextToken()
				_ = p.parseBlockStatement()
			}
			return nil
		}
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	expression.Consequence = p.parseBlockStatement()

	if p.peekTokenIs(token.ELSE) {
		p.nextToken()
		if p.peekTokenIs(token.IF) {
			p.nextToken()
			elseIf := p.parseIfExpression()
			stmt := &ast.ExpressionStatement{Token: p.curToken, Expression: elseIf}
			expression.Alternative = &ast.BlockStatement{
				Token:      p.curToken,
				Statements: []ast.Statement{stmt},
			}
		} else {
			if !p.expectPeek(token.LBRACE) {
				return nil
			}
			expression.Alternative = p.parseBlockStatement()
		}
	}

	return expression
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.curToken}
	block.Statements = []ast.Statement{}

	p.nextToken()

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
}

func (p *Parser) parseFunctionLiteral() ast.Expression {
	lit := &ast.FunctionLiteral{Token: p.curToken}

	if p.peekTokenIs(token.IDENT) {
		p.nextToken()
		lit.Name = p.curToken.Literal
	}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	params, destructStmts := p.parseFunctionParameters()
	lit.Parameters = params

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	lit.Body = p.parseBlockStatement()
	if lit.Body != nil && len(destructStmts) > 0 {
		lit.Body.Statements = append(destructStmts, lit.Body.Statements...)
	}

	return lit
}

func (p *Parser) parseArrayPattern() *ast.ArrayPattern {
	pattern := &ast.ArrayPattern{Token: p.curToken, Elements: []ast.Expression{}}

	if p.peekTokenIs(token.RBRACKET) {
		p.nextToken()
		return pattern
	}

	for !p.curTokenIs(token.RBRACKET) && !p.curTokenIs(token.EOF) {
		p.nextToken()
		if p.curTokenIs(token.ELLIPSIS) {
			ellipsisTok := p.curToken
			if !p.expectPeek(token.IDENT) {
				return nil
			}
			target := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
			pattern.Elements = append(pattern.Elements, &ast.RestPattern{Token: ellipsisTok, Target: target})
		} else if p.curTokenIs(token.IDENT) {
			ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
			pattern.Elements = append(pattern.Elements, ident)
		} else if p.curTokenIs(token.RBRACKET) {
			break
		}

		if p.peekTokenIs(token.COMMA) {
			p.nextToken()
		} else if p.peekTokenIs(token.RBRACKET) {
			p.nextToken()
			break
		}
	}

	return pattern
}

func (p *Parser) parseHashPattern() *ast.HashPattern {
	pattern := &ast.HashPattern{Token: p.curToken, Keys: []*ast.Identifier{}}

	if p.peekTokenIs(token.RBRACE) {
		p.nextToken()
		return pattern
	}

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		p.nextToken()
		if p.curTokenIs(token.IDENT) {
			ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
			pattern.Keys = append(pattern.Keys, ident)
		} else if p.curTokenIs(token.RBRACE) {
			break
		}

		if p.peekTokenIs(token.COMMA) {
			p.nextToken()
		} else if p.peekTokenIs(token.RBRACE) {
			p.nextToken()
			break
		}
	}

	return pattern
}

func (p *Parser) parseFunctionParameters() ([]*ast.Identifier, []ast.Statement) {
	identifiers := []*ast.Identifier{}
	destructStmts := []ast.Statement{}
	seen := make(map[string]bool)

	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return identifiers, destructStmts
	}

	parseOneParam := func() {
		if p.peekTokenIs(token.LBRACE) {
			p.nextToken()
			pattern := p.parseHashPattern()
			p.destructCounter++
			synthName := fmt.Sprintf("__param_destruct_%d", p.destructCounter)
			ident := &ast.Identifier{Token: token.Token{Type: token.IDENT, Literal: synthName}, Value: synthName}
			identifiers = append(identifiers, ident)
			destructStmts = append(destructStmts, &ast.LetStatement{
				Token:   token.Token{Type: token.LET, Literal: "let"},
				Pattern: pattern,
				Value:   ident,
			})
		} else if p.peekTokenIs(token.LBRACKET) {
			p.nextToken()
			pattern := p.parseArrayPattern()
			p.destructCounter++
			synthName := fmt.Sprintf("__param_destruct_%d", p.destructCounter)
			ident := &ast.Identifier{Token: token.Token{Type: token.IDENT, Literal: synthName}, Value: synthName}
			identifiers = append(identifiers, ident)
			destructStmts = append(destructStmts, &ast.LetStatement{
				Token:   token.Token{Type: token.LET, Literal: "let"},
				Pattern: pattern,
				Value:   ident,
			})
		} else {
			p.nextToken()
			ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
			if seen[ident.Value] {
				p.errors = append(p.errors, fmt.Sprintf("duplicate parameter '%s' in function definition at line %d, col %d", ident.Value, p.curToken.Line, p.curToken.Col))
			}
			seen[ident.Value] = true
			identifiers = append(identifiers, ident)
		}
	}

	parseOneParam()

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		parseOneParam()
	}

	if !p.expectPeek(token.RPAREN) {
		return nil, nil
	}

	return identifiers, destructStmts
}

func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	exp := &ast.CallExpression{Token: p.curToken, Function: function}
	exp.Arguments = p.parseExpressionList(token.RPAREN)
	return exp
}

func (p *Parser) parseExpressionList(end token.TokenType) []ast.Expression {
	list := []ast.Expression{}

	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}

	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(end) {
		return nil
	}

	return list
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Token: p.curToken}
	array.Elements = p.parseExpressionList(token.RBRACKET)
	return array
}

func (p *Parser) parseHashLiteral() ast.Expression {
	hash := &ast.HashLiteral{Token: p.curToken}
	hash.Pairs = make(map[ast.Expression]ast.Expression)

	for !p.peekTokenIs(token.RBRACE) {
		p.nextToken()
		key := p.parseExpression(LOWEST)

		if !p.expectPeek(token.COLON) {
			return nil
		}

		p.nextToken()
		val := p.parseExpression(LOWEST)

		hash.Pairs[key] = val

		if !p.peekTokenIs(token.RBRACE) && !p.expectPeek(token.COMMA) {
			return nil
		}
	}

	if !p.expectPeek(token.RBRACE) {
		return nil
	}

	return hash
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	tok := p.curToken // '['
	p.nextToken()

	// Case 1: Start is omitted, starts with ':' (e.g. [:], [::], [::step], [:end], [:end:step])
	if p.curTokenIs(token.COLON) {
		sliceExp := &ast.SliceExpression{Token: tok, Left: left, Start: nil}
		p.nextToken() // consume first ':'

		// Check if second ':' comes immediately: [::] or [::step]
		if p.curTokenIs(token.COLON) {
			p.nextToken() // consume second ':'
			if !p.curTokenIs(token.RBRACKET) {
				sliceExp.Step = p.parseExpression(LOWEST)
				if !p.expectPeek(token.RBRACKET) {
					return nil
				}
			}
			return sliceExp
		}

		if p.curTokenIs(token.RBRACKET) {
			return sliceExp
		}

		sliceExp.End = p.parseExpression(LOWEST)
		if p.peekTokenIs(token.COLON) {
			p.nextToken() // curToken is now ':'
			p.nextToken() // consume second ':'
			if !p.curTokenIs(token.RBRACKET) {
				sliceExp.Step = p.parseExpression(LOWEST)
				if !p.expectPeek(token.RBRACKET) {
					return nil
				}
			}
			return sliceExp
		}

		if !p.expectPeek(token.RBRACKET) {
			return nil
		}
		return sliceExp
	}

	startOrIndex := p.parseExpression(LOWEST)

	// Case 2: Start is provided (or it's an IndexExpression)
	if p.peekTokenIs(token.COLON) {
		p.nextToken() // curToken is now ':'
		sliceExp := &ast.SliceExpression{Token: tok, Left: left, Start: startOrIndex}
		p.nextToken() // consume first ':'

		// Check if second ':' comes immediately: [start::] or [start::step]
		if p.curTokenIs(token.COLON) {
			p.nextToken() // consume second ':'
			if !p.curTokenIs(token.RBRACKET) {
				sliceExp.Step = p.parseExpression(LOWEST)
				if !p.expectPeek(token.RBRACKET) {
					return nil
				}
			}
			return sliceExp
		}

		if p.curTokenIs(token.RBRACKET) {
			return sliceExp
		}

		sliceExp.End = p.parseExpression(LOWEST)
		if p.peekTokenIs(token.COLON) {
			p.nextToken() // curToken is now ':'
			p.nextToken() // consume second ':'
			if !p.curTokenIs(token.RBRACKET) {
				sliceExp.Step = p.parseExpression(LOWEST)
				if !p.expectPeek(token.RBRACKET) {
					return nil
				}
			}
			return sliceExp
		}

		if !p.expectPeek(token.RBRACKET) {
			return nil
		}
		return sliceExp
	}

	exp := &ast.IndexExpression{Token: tok, Left: left, Index: startOrIndex}
	if !p.expectPeek(token.RBRACKET) {
		return nil
	}

	return exp
}

func (p *Parser) parsePipeExpression(left ast.Expression) ast.Expression {
	exp := &ast.PipeExpression{
		Token: p.curToken,
		Left:  left,
	}
	precedence := p.curPrecedence()
	p.nextToken()
	exp.Right = p.parseExpression(precedence)
	return exp
}

func (p *Parser) parseYieldExpression() ast.Expression {
	exp := &ast.YieldExpression{Token: p.curToken}
	if p.peekTokenIs(token.SEMICOLON) || p.peekTokenIs(token.RBRACE) || p.peekTokenIs(token.COMMA) {
		return exp
	}
	p.nextToken()
	exp.Value = p.parseExpression(LOWEST)
	return exp
}
