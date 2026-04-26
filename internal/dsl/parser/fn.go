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
		p.nextToken() // advance past ; to next token or }
	}
	return stmts
}

func (p *Parser) parseLogStmt() ast.FnStmt {
	tok := p.curToken
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	p.nextToken() // advance past (
	args := p.parseArgList()
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
		Args:  args,
	}
}

func (p *Parser) parseExecStmt() ast.FnStmt {
	tok := p.curToken
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	p.nextToken() // advance past (
	args := p.parseArgList()
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
		Args:  args,
	}
}

func (p *Parser) parseArgList() []ast.Expr {
	args := []ast.Expr{}
	if p.curTokenIs(token.RPAREN) {
		return args // empty arg list
	}
	for {
		switch p.curToken.Type {
		case token.BACKTICK:
			args = append(args, &ast.BacktickLit{Token: p.curToken, Value: p.curToken.Literal})
		case token.DOLLAR:
			p.nextToken()
			if p.curToken.Type != token.IDENT {
				p.errors = append(p.errors, fmt.Sprintf("expected identifier after $ at %d:%d", p.curToken.Line, p.curToken.Col))
				return args
			}
			args = append(args, &ast.VarRef{Token: p.curToken, Name: p.curToken.Literal})
		default:
			p.errors = append(p.errors, fmt.Sprintf("expected string or variable at %d:%d", p.curToken.Line, p.curToken.Col))
			return args
		}
		if p.peekTokenIs(token.COMMA) {
			p.nextToken() // to COMMA
			p.nextToken() // to next argument or RPAREN
			if p.curTokenIs(token.RPAREN) {
				break // trailing comma
			}
			continue
		}
		if p.peekTokenIs(token.RPAREN) {
			p.nextToken() // advance to RPAREN
			break
		}
		// If neither comma nor RPAREN after argument, it's an error but will be caught by caller's RPAREN check
		break
	}
	return args
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
		var value ast.Expr
		switch p.curToken.Type {
		case token.BACKTICK:
			value = &ast.BacktickLit{Token: p.curToken, Value: p.curToken.Literal}
		case token.DOLLAR:
			p.nextToken()
			if p.curToken.Type != token.IDENT {
				p.errors = append(p.errors, fmt.Sprintf("expected identifier after $ at %d:%d", p.curToken.Line, p.curToken.Col))
				return nil
			}
			value = &ast.VarRef{Token: p.curToken, Name: p.curToken.Literal}
		default:
			p.errors = append(p.errors, fmt.Sprintf("expected string literal or variable reference at %d:%d", p.curToken.Line, p.curToken.Col))
			return nil
		}
		pairs = append(pairs, ast.EnvPair{Key: key, Value: value})
		if p.peekTokenIs(token.COMMA) {
			p.nextToken()
			if p.peekTokenIs(token.RBRACKET) {
				break // trailing comma
			}
			continue
		}
		if p.peekTokenIs(token.RBRACKET) {
			break
		}
		p.errors = append(p.errors, fmt.Sprintf("expected ',' or ']' in env at %d:%d", p.peekToken.Line, p.peekToken.Col))
		return nil
	}
	if !p.expectPeek(token.RBRACKET) {
		return nil
	}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	p.nextToken() // advance past {
	body := p.parseFnBody()
	// curToken is RBRACE after body
	if !p.expectPeek(token.SEMICOLON) {
		return nil
	}
	return &ast.EnvBlock{
		Token: tok,
		Pairs: pairs,
		Body:  body,
	}
}
