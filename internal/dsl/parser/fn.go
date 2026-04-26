package parser

import (
	"fmt"
	"strings"

	"github.com/infraflakes/sro/internal/dsl/ast"
	"github.com/infraflakes/sro/internal/dsl/token"
)

// parseBacktickParts splits raw backtick content into TemplateParts for interpolation.
func parseBacktickParts(raw string, line, col int) ([]ast.TemplatePart, error) {
	var parts []ast.TemplatePart
	i := 0
	for i < len(raw) {
		// Find next ${
		idx := strings.Index(raw[i:], "${")
		if idx == -1 {
			// Rest is literal text
			parts = append(parts, ast.TemplatePart{IsVar: false, Value: raw[i:]})
			break
		}
		// Add literal text before ${
		if idx > 0 {
			parts = append(parts, ast.TemplatePart{IsVar: false, Value: raw[i : i+idx]})
		}
		// Find closing }
		i += idx + 2 // skip past ${
		end := strings.Index(raw[i:], "}")
		if end == -1 {
			return nil, fmt.Errorf("unterminated ${} at %d:%d", line, col)
		}
		name := raw[i : i+end]
		if name == "" {
			return nil, fmt.Errorf("empty ${} at %d:%d", line, col)
		}
		parts = append(parts, ast.TemplatePart{IsVar: true, Value: name})
		i += end + 1 // skip past }
	}
	if len(parts) == 0 {
		parts = append(parts, ast.TemplatePart{IsVar: false, Value: ""})
	}
	return parts, nil
}

// parseExpr parses a single expression (backtick literal or variable reference).
func (p *Parser) parseExpr() ast.Expr {
	switch p.curToken.Type {
	case token.BACKTICK:
		tok := p.curToken
		parts, err := parseBacktickParts(p.curToken.Literal, p.curToken.Line, p.curToken.Col)
		if err != nil {
			p.errors = append(p.errors, err.Error())
			return nil
		}
		p.nextToken() // consume BACKTICK
		return &ast.BacktickLit{Token: tok, Parts: parts}
	case token.DOLLAR:
		p.nextToken()
		if p.curToken.Type != token.IDENT {
			p.errors = append(p.errors, fmt.Sprintf("expected identifier after $ at %d:%d", p.curToken.Line, p.curToken.Col))
			return nil
		}
		tok := p.curToken
		p.nextToken() // consume IDENT
		return &ast.VarRef{Token: tok, Name: tok.Literal}
	default:
		p.errors = append(p.errors, fmt.Sprintf("expected backtick literal or variable reference at %d:%d", p.curToken.Line, p.curToken.Col))
		return nil
	}
}

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
			p.errors = append(p.errors, fmt.Sprintf("missing closing brace at %d:%d", p.curToken.Line, p.curToken.Col))
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
			p.errors = append(p.errors, fmt.Sprintf("unexpected token %s in fn body at %d:%d", p.curToken.Type, p.curToken.Line, p.curToken.Col))
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
		p.errors = append(p.errors, fmt.Sprintf("expected ')' at %d:%d", p.curToken.Line, p.curToken.Col))
		return nil
	}
	p.nextToken() // consume )
	if !p.curTokenIs(token.SEMICOLON) {
		p.errors = append(p.errors, fmt.Sprintf("expected ';' at %d:%d", p.curToken.Line, p.curToken.Col))
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
		p.errors = append(p.errors, fmt.Sprintf("expected ')' at %d:%d", p.curToken.Line, p.curToken.Col))
		return nil
	}
	p.nextToken() // consume )
	if !p.curTokenIs(token.SEMICOLON) {
		p.errors = append(p.errors, fmt.Sprintf("expected ';' at %d:%d", p.curToken.Line, p.curToken.Col))
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
		p.errors = append(p.errors, fmt.Sprintf("expected ',' or ']' in env at %d:%d", p.peekToken.Line, p.peekToken.Col))
		return nil
	}
	if !p.curTokenIs(token.LBRACE) {
		p.errors = append(p.errors, fmt.Sprintf("expected '{' after env pairs at %d:%d", p.curToken.Line, p.curToken.Col))
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
