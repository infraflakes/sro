package parser

import (
	"fmt"

	"github.com/infraflakes/sro/internal/dsl/ast"
	"github.com/infraflakes/sro/internal/dsl/token"
)

func (p *Parser) parseFnDecl() ast.Stmt {
	tok := p.curToken
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	name := p.curToken.Literal
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	p.nextToken() // advance past {
	body := p.parseFnBody()
	// curToken is RBRACE after body
	return &ast.FnDecl{
		Token: tok,
		Name:  name,
		Body:  body,
	}
}

func (p *Parser) parseFnBody() []ast.FnStmt {
	stmts := []ast.FnStmt{}
	for !p.curTokenIs(token.RBRACE) {
		if p.curTokenIs(token.EOF) {
			p.errors = append(p.errors, ParseError{
				Message: "missing closing brace",
				Line:    p.curToken.Line,
				Col:     p.curToken.Col,
			})
			return stmts
		}
		var stmt ast.FnStmt
		switch p.curToken.Type {
		case token.LOG:
			stmt = p.parseLogStmt()
		case token.EXEC:
			stmt = p.parseExecStmt()
		case token.CD:
			stmt = p.parseCdStmt()
		case token.VAR:
			stmt = p.parseVarDecl()
		case token.ENV:
			stmt = p.parseEnvBlock()
		default:
			p.errors = append(p.errors, ParseError{
				Message: fmt.Sprintf("unexpected token %s in fn body", p.curToken.Type),
				Line:    p.curToken.Line,
				Col:     p.curToken.Col,
			})
			p.nextToken()
			continue
		}
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
		// advance past ; to next token or }
		p.nextToken()
	}
	return stmts
}

func (p *Parser) parseLogStmt() ast.FnStmt {
	tok := p.curToken
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	p.nextToken() // advance past (
	expr := p.parseExpr()
	if expr == nil {
		return nil
	}
	if !p.curTokenIs(token.RPAREN) {
		p.errors = append(p.errors, ParseError{
			Message: "expected ')'",
			Line:    p.curToken.Line,
			Col:     p.curToken.Col,
		})
		return nil
	}
	p.nextToken() // consume )
	if !p.curTokenIs(token.SEMICOLON) {
		p.errors = append(p.errors, ParseError{
			Message: "expected ';'",
			Line:    p.curToken.Line,
			Col:     p.curToken.Col,
		})
		return nil
	}
	// semicolon will be consumed by ParseProgram
	return &ast.LogStmt{
		Token: tok,
		Value: expr,
	}
}

func (p *Parser) parseExecStmt() ast.FnStmt {
	tok := p.curToken
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	p.nextToken() // advance past (
	expr := p.parseExpr()
	if expr == nil {
		return nil
	}
	if !p.curTokenIs(token.RPAREN) {
		p.errors = append(p.errors, ParseError{
			Message: "expected ')'",
			Line:    p.curToken.Line,
			Col:     p.curToken.Col,
		})
		return nil
	}
	p.nextToken() // consume )
	if !p.curTokenIs(token.SEMICOLON) {
		p.errors = append(p.errors, ParseError{
			Message: "expected ';'",
			Line:    p.curToken.Line,
			Col:     p.curToken.Col,
		})
		return nil
	}
	return &ast.ExecStmt{
		Token: tok,
		Value: expr,
	}
}

func (p *Parser) parseCdStmt() ast.FnStmt {
	tok := p.curToken
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	if !p.expectPeek(token.BACKTICK) {
		return nil
	}
	arg := p.curToken.Literal
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	if !p.expectPeek(token.SEMICOLON) {
		return nil
	}
	return &ast.CdStmt{
		Token: tok,
		Arg:   arg,
	}
}

func (p *Parser) parseEnvBlock() ast.FnStmt {
	tok := p.curToken
	if !p.expectPeek(token.LBRACKET) {
		return nil
	}
	pairs := []ast.EnvPair{}
	for {
		if !p.expectPeek(token.IDENT) {
			return nil
		}
		key := p.curToken.Literal
		if !p.expectPeek(token.ASSIGN) {
			return nil
		}
		p.nextToken() // advance past =
		value := p.parseExpr()
		if value == nil {
			return nil
		}
		pairs = append(pairs, ast.EnvPair{Key: key, Value: value})
		if p.curTokenIs(token.COMMA) {
			if p.peekTokenIs(token.RBRACKET) {
				p.nextToken() // to ]
				p.nextToken() // past ]
				break         // trailing comma
			}
			continue // expectPeek(IDENT) at loop top advances from , to IDENT
		}
		if p.curTokenIs(token.RBRACKET) {
			p.nextToken() // consume ]
			break
		}
		p.errors = append(p.errors, ParseError{
			Message: "expected ',' or ']' in env",
			Line:    p.peekToken.Line,
			Col:     p.peekToken.Col,
		})
		return nil
	}
	if !p.curTokenIs(token.LBRACE) {
		p.errors = append(p.errors, ParseError{
			Message: "expected '{' after env pairs",
			Line:    p.curToken.Line,
			Col:     p.curToken.Col,
		})
		return nil
	}
	p.nextToken() // consume {
	body := p.parseFnBody()
	// curToken is RBRACE after body - parseFnBody stops at RBRACE
	p.nextToken() // consume the env block's closing }
	// semicolon will be consumed by parseFnBody
	return &ast.EnvBlock{
		Token: tok,
		Pairs: pairs,
		Body:  body,
	}
}
