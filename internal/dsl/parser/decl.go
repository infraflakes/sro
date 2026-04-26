package parser

import (
	"fmt"

	"github.com/infraflakes/sro/internal/dsl/ast"
	"github.com/infraflakes/sro/internal/dsl/token"
)

func (p *Parser) parseShellDecl() ast.Stmt {
	tok := p.curToken
	if !p.expectPeek(token.ASSIGN) {
		return nil
	}
	if !p.expectPeek(token.STRING_LIT) {
		return nil
	}
	value := p.curToken.Literal
	if !p.expectPeek(token.SEMICOLON) {
		return nil
	}
	return &ast.ShellDecl{
		Token: tok,
		Value: value,
	}
}

func (p *Parser) parseSanctuaryDecl() ast.Stmt {
	tok := p.curToken
	if !p.expectPeek(token.ASSIGN) {
		return nil
	}
	p.nextToken() // move to value
	var value ast.Expr
	switch p.curToken.Type {
	case token.STRING_LIT:
		value = &ast.StringLit{Token: p.curToken, Value: p.curToken.Literal}
	case token.DOLLAR:
		p.nextToken()
		if p.curToken.Type != token.IDENT {
			p.errors = append(p.errors, fmt.Sprintf("expected identifier after $ at %d:%d", p.curToken.Line, p.curToken.Col))
			return nil
		}
		value = &ast.VarRef{Token: p.curToken, Name: p.curToken.Literal}
	default:
		p.errors = append(p.errors, fmt.Sprintf("expected string or variable reference at %d:%d", p.curToken.Line, p.curToken.Col))
		return nil
	}
	if !p.expectPeek(token.SEMICOLON) {
		return nil
	}
	return &ast.SanctuaryDecl{Token: tok, Value: value}
}

func (p *Parser) parseImportDecl() ast.Stmt {
	tok := p.curToken
	if !p.expectPeek(token.LBRACKET) {
		return nil
	}
	p.nextToken() // advance past [
	paths := []string{}
	for p.curToken.Type != token.RBRACKET {
		if p.curToken.Type != token.PATH_LIT {
			p.errors = append(p.errors, fmt.Sprintf("expected path literal at %d:%d", p.curToken.Line, p.curToken.Col))
			return nil
		}
		paths = append(paths, p.curToken.Literal)
		p.nextToken() // move past this path token
		if p.curToken.Type == token.COMMA {
			p.nextToken() // skip comma, move to next path or RBRACKET
		} else if p.curToken.Type != token.RBRACKET {
			p.errors = append(p.errors, fmt.Sprintf("expected ',' or ']' after path at %d:%d", p.curToken.Line, p.curToken.Col))
			return nil
		}
	}
	// curToken is RBRACKET
	if !p.expectPeek(token.SEMICOLON) {
		return nil
	}
	return &ast.ImportDecl{
		Token: tok,
		Paths: paths,
	}
}

func (p *Parser) parseVarDecl() *ast.VarDecl {
	tok := p.curToken
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	name := p.curToken.Literal
	if !p.expectPeek(token.DECLARE) {
		return nil
	}
	p.nextToken() // move to value (STRING or DOLLAR)
	var value ast.Expr
	switch p.curToken.Type {
	case token.STRING_LIT:
		value = &ast.StringLit{Token: p.curToken, Value: p.curToken.Literal}
	case token.DOLLAR:
		p.nextToken() // consume $
		if p.curToken.Type != token.IDENT {
			p.errors = append(p.errors, fmt.Sprintf("expected identifier after $ at %d:%d", p.curToken.Line, p.curToken.Col))
			return nil
		}
		value = &ast.VarRef{Token: p.curToken, Name: p.curToken.Literal}
	case token.SHELL_LIT:
		value = &ast.ShellExec{Token: p.curToken, Command: p.curToken.Literal}
	default:
		p.errors = append(p.errors, fmt.Sprintf("expected string or variable reference at %d:%d", p.curToken.Line, p.curToken.Col))
		return nil
	}
	if !p.expectPeek(token.SEMICOLON) {
		return nil
	}
	return &ast.VarDecl{
		Token: tok,
		Name:  name,
		Value: value,
	}
}

func (p *Parser) parseProjectDecl() ast.Stmt {
	tok := p.curToken
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	name := p.curToken.Literal
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	p.nextToken() // advance past {
	fields := []ast.ProjectField{}
	for !p.curTokenIs(token.RBRACE) {
		if p.curTokenIs(token.EOF) {
			p.errors = append(p.errors, fmt.Sprintf("missing closing brace for project at %d:%d", p.curToken.Line, p.curToken.Col))
			return nil
		}
		key := p.curToken.Literal
		validKeys := map[string]bool{"url": true, "dir": true, "sync": true, "use": true}
		if !validKeys[key] {
			p.errors = append(p.errors, fmt.Sprintf("invalid project field key '%s' at %d:%d", key, p.curToken.Line, p.curToken.Col))
			p.nextToken()
			continue
		}
		if !p.expectPeek(token.ASSIGN) {
			return nil
		}
		if !p.expectPeek(token.STRING_LIT) {
			return nil
		}
		value := p.curToken.Literal
		fields = append(fields, ast.ProjectField{Key: key, Value: value})
		if !p.expectPeek(token.SEMICOLON) {
			return nil
		}
		p.nextToken() // advance to next field or }
	}
	// curToken is RBRACE
	return &ast.ProjectDecl{
		Token:  tok,
		Name:   name,
		Fields: fields,
	}
}
