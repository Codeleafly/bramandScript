package parser

import (
	"fmt"
	"bramandscript/ast"
	"bramandscript/token"
)

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.IDENT:
		if p.peekTokenIs(token.ASSIGN) || p.peekTokenIs(token.PLUS_ASSIGN) ||
			p.peekTokenIs(token.MINUS_ASSIGN) || p.peekTokenIs(token.MUL_ASSIGN) ||
			p.peekTokenIs(token.DIV_ASSIGN) || p.peekTokenIs(token.COALESCE_ASSIGN) {
			return p.parseReassignStatement()
		}
		return p.parseExpressionStatement()
	case token.LET:
		return p.parseLetStatement()
	case token.CONST:
		return p.parseConstStatement()
	case token.RETURN:
		return p.parseReturnStatement()
	case token.WHILE:
		return p.parseWhileStatement()
	case token.FOR:
		return p.parseForInStatement()
	case token.IMPORT:
		return p.parseImportStatement()
	case token.STRUCT:
		return p.parseStructStatement()
	case token.ENUM:
		return p.parseEnumStatement()
	case token.BREAK:
		return p.parseBreakStatement()
	case token.CONTINUE:
		return p.parseContinueStatement()
	case token.TRY:
		return p.parseTryCatchStatement()
	case token.THROW:
		return p.parseThrowStatement()
	case token.DEFER:
		return p.parseDeferStatement()
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseReassignStatement() *ast.ReassignStatement {
	stmt := &ast.ReassignStatement{
		Name: &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal},
	}

	p.nextToken()
	stmt.Token = p.curToken
	stmt.Operator = p.curToken.Literal

	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseWhileStatement() *ast.WhileStatement {
	stmt := &ast.WhileStatement{Token: p.curToken}

	hasParen := false
	if p.peekTokenIs(token.LPAREN) {
		hasParen = true
		p.nextToken()
	}

	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

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

	stmt.Body = p.parseBlockStatement()
	return stmt
}

func (p *Parser) parseBreakStatement() *ast.BreakStatement {
	stmt := &ast.BreakStatement{Token: p.curToken}
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseContinueStatement() *ast.ContinueStatement {
	stmt := &ast.ContinueStatement{Token: p.curToken}
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseLetStatement() *ast.LetStatement {
	stmt := &ast.LetStatement{Token: p.curToken}

	if p.peekTokenIs(token.LBRACKET) {
		p.nextToken()
		stmt.Pattern = p.parseArrayPattern()
	} else if p.peekTokenIs(token.LBRACE) {
		p.nextToken()
		stmt.Pattern = p.parseHashPattern()
	} else if p.peekTokenIs(token.LPAREN) {
		p.nextToken()
		stmt.Pattern = p.parseTuplePattern()
	} else if p.peekTokenIs(token.IDENT) {
		p.nextToken()
		stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	} else {
		p.peekError(token.IDENT)
		return nil
	}

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseConstStatement() *ast.ConstStatement {
	stmt := &ast.ConstStatement{Token: p.curToken}

	if p.peekTokenIs(token.LBRACKET) {
		p.nextToken()
		stmt.Pattern = p.parseArrayPattern()
	} else if p.peekTokenIs(token.LBRACE) {
		p.nextToken()
		stmt.Pattern = p.parseHashPattern()
	} else if p.peekTokenIs(token.LPAREN) {
		p.nextToken()
		stmt.Pattern = p.parseTuplePattern()
	} else if p.peekTokenIs(token.IDENT) {
		p.nextToken()
		stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	} else {
		p.peekError(token.IDENT)
		return nil
	}

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.curToken}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
		return stmt
	}

	p.nextToken()

	if p.curTokenIs(token.SEMICOLON) {
		return stmt
	}

	stmt.ReturnValue = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseExpressionStatement() ast.Statement {
	firstToken := p.curToken
	expr := p.parseExpression(LOWEST)

	if p.peekTokenIs(token.ASSIGN) || p.peekTokenIs(token.PLUS_ASSIGN) ||
		p.peekTokenIs(token.MINUS_ASSIGN) || p.peekTokenIs(token.MUL_ASSIGN) ||
		p.peekTokenIs(token.DIV_ASSIGN) || p.peekTokenIs(token.COALESCE_ASSIGN) {
		p.nextToken()
		assignToken := p.curToken
		op := assignToken.Literal
		p.nextToken()
		val := p.parseExpression(LOWEST)

		if p.peekTokenIs(token.SEMICOLON) {
			p.nextToken()
		}

		if idxExp, ok := expr.(*ast.IndexExpression); ok {
			return &ast.IndexAssignStatement{
				Token:    assignToken,
				Left:     idxExp.Left,
				Index:    idxExp.Index,
				Operator: op,
				Value:    val,
			}
		}

		if dotExp, ok := expr.(*ast.DotExpression); ok {
			return &ast.IndexAssignStatement{
				Token:    assignToken,
				Left:     dotExp.Left,
				Index:    &ast.StringLiteral{Token: dotExp.Field.Token, Value: dotExp.Field.Value},
				Operator: op,
				Value:    val,
			}
		}

		msg := fmt.Sprintf("invalid assignment target at line %d, col %d", assignToken.Line, assignToken.Col)
		p.errors = append(p.errors, msg)
		p.SyntaxErrors = append(p.SyntaxErrors, SyntaxError{
			Line:    assignToken.Line,
			Col:     assignToken.Col,
			Message: "invalid assignment target",
		})
		return nil
	}

	stmt := &ast.ExpressionStatement{Token: firstToken, Expression: expr}
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseForInStatement() *ast.ForInStatement {
	stmt := &ast.ForInStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	first := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if p.peekTokenIs(token.COMMA) {
		p.nextToken()
		if !p.expectPeek(token.IDENT) {
			return nil
		}
		stmt.Index = first
		stmt.Item = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	} else {
		stmt.Item = first
	}

	if !p.expectPeek(token.IN) {
		return nil
	}

	p.nextToken()
	stmt.Iterable = p.parseExpression(LOWEST)

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()
	return stmt
}

func (p *Parser) parseImportStatement() *ast.ImportStatement {
	stmt := &ast.ImportStatement{Token: p.curToken}

	if !p.expectPeek(token.STRING) {
		return nil
	}
	stmt.Path = p.curToken.Literal

	if p.peekTokenIs(token.AS) {
		p.nextToken()
		if !p.expectPeek(token.IDENT) {
			return nil
		}
		stmt.Alias = p.curToken.Literal
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseStructStatement() *ast.StructStatement {
	stmt := &ast.StructStatement{
		Token:   p.curToken,
		Embeds:  []*ast.Identifier{},
		Fields:  []*ast.Identifier{},
		Methods: []*ast.FunctionLiteral{},
	}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	p.nextToken()

	seenFields := make(map[string]bool)
	seenMethods := make(map[string]bool)

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.EMBED) {
			if !p.expectPeek(token.IDENT) {
				return nil
			}
			stmt.Embeds = append(stmt.Embeds, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})
			if p.peekTokenIs(token.COMMA) {
				p.nextToken()
			}
		} else if p.curTokenIs(token.FUNCTION) {
			fn := p.parseFunctionLiteral()
			if fnLit, ok := fn.(*ast.FunctionLiteral); ok {
				if fnLit.Name != "" && seenMethods[fnLit.Name] {
					p.errors = append(p.errors, fmt.Sprintf("duplicate method '%s' in struct '%s' at line %d, col %d", fnLit.Name, stmt.Name.Value, fnLit.Token.Line, fnLit.Token.Col))
				}
				seenMethods[fnLit.Name] = true
				stmt.Methods = append(stmt.Methods, fnLit)
			}
		} else if p.curTokenIs(token.IDENT) {
			fieldName := p.curToken.Literal
			if seenFields[fieldName] {
				p.errors = append(p.errors, fmt.Sprintf("duplicate field '%s' in struct '%s' at line %d, col %d", fieldName, stmt.Name.Value, p.curToken.Line, p.curToken.Col))
			}
			seenFields[fieldName] = true
			stmt.Fields = append(stmt.Fields, &ast.Identifier{Token: p.curToken, Value: fieldName})
			if p.peekTokenIs(token.COMMA) {
				p.nextToken()
			}
		} else if p.curTokenIs(token.COMMA) || p.curTokenIs(token.SEMICOLON) {
			// allow separating comma or semicolon
		} else {
			p.errors = append(p.errors, fmt.Sprintf("unexpected token '%s' in struct '%s' body at line %d, col %d", p.curToken.Literal, stmt.Name.Value, p.curToken.Line, p.curToken.Col))
			p.SyntaxErrors = append(p.SyntaxErrors, SyntaxError{
				Line:    p.curToken.Line,
				Col:     p.curToken.Col,
				Message: fmt.Sprintf("unexpected token '%s' in struct definition", p.curToken.Literal),
			})
		}
		p.nextToken()
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseEnumStatement() *ast.EnumStatement {
	stmt := &ast.EnumStatement{
		Token:   p.curToken,
		Entries: []*ast.EnumEntry{},
	}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	p.nextToken()

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.IDENT) {
			entry := &ast.EnumEntry{
				Name: &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal},
			}
			if p.peekTokenIs(token.ASSIGN) {
				p.nextToken() // consume '='
				p.nextToken() // move to expression
				entry.Value = p.parseExpression(LOWEST)
			}
			stmt.Entries = append(stmt.Entries, entry)
			if p.peekTokenIs(token.COMMA) {
				p.nextToken()
			}
		} else if p.curTokenIs(token.COMMA) || p.curTokenIs(token.SEMICOLON) {
			// allow separating comma or semicolon
		} else {
			p.errors = append(p.errors, fmt.Sprintf("unexpected token '%s' in enum '%s' body at line %d, col %d", p.curToken.Literal, stmt.Name.Value, p.curToken.Line, p.curToken.Col))
			p.SyntaxErrors = append(p.SyntaxErrors, SyntaxError{
				Line:    p.curToken.Line,
				Col:     p.curToken.Col,
				Message: fmt.Sprintf("unexpected token '%s' in enum definition", p.curToken.Literal),
			})
		}
		p.nextToken()
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseTryCatchStatement() *ast.TryCatchStatement {
	stmt := &ast.TryCatchStatement{Token: p.curToken}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	stmt.Block = p.parseBlockStatement()

	if p.peekTokenIs(token.CATCH) {
		p.nextToken() // curToken is now CATCH

		if p.peekTokenIs(token.LPAREN) {
			p.nextToken() // LPAREN
			if p.peekTokenIs(token.IDENT) {
				p.nextToken()
				stmt.CatchVar = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
			}
			if !p.expectPeek(token.RPAREN) {
				return nil
			}
		} else if p.peekTokenIs(token.IDENT) {
			p.nextToken()
			stmt.CatchVar = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		}

		if !p.expectPeek(token.LBRACE) {
			return nil
		}
		stmt.CatchBlock = p.parseBlockStatement()
	}

	if p.peekTokenIs(token.FINALLY) {
		p.nextToken() // FINALLY
		if !p.expectPeek(token.LBRACE) {
			return nil
		}
		stmt.FinallyBlock = p.parseBlockStatement()
	}

	if stmt.CatchBlock == nil && stmt.FinallyBlock == nil {
		p.errors = append(p.errors, fmt.Sprintf("expected 'catch' or 'finally' after try block at line %d, col %d", p.curToken.Line, p.curToken.Col))
		p.SyntaxErrors = append(p.SyntaxErrors, SyntaxError{
			Line:    p.curToken.Line,
			Col:     p.curToken.Col,
			Message: "expected 'catch' or 'finally' after try block",
		})
		return nil
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseThrowStatement() *ast.ThrowStatement {
	stmt := &ast.ThrowStatement{Token: p.curToken}
	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseDeferStatement() *ast.DeferStatement {
	stmt := &ast.DeferStatement{Token: p.curToken}
	p.nextToken()
	stmt.Call = p.parseExpression(LOWEST)
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}
