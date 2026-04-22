package parser

import (
	"fmt"

	"github.com/infraflakes/sro/ast"
	"github.com/infraflakes/sro/lexer"
	"github.com/infraflakes/sro/token"
)

type Parser struct {
	l         *lexer.Lexer
	curToken  token.Token
	peekToken token.Token
	errors    []string
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}
	p.nextToken()
	p.nextToken()
	return p
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
	msg := fmt.Sprintf("expected %s, got %s at %d:%d", t, p.peekToken.Type, p.peekToken.Line, p.peekToken.Col)
	p.errors = append(p.errors, msg)
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{Statements: []ast.Node{}}

	for p.curToken.Type != token.EOF {
		switch p.curToken.Type {
		case token.SANCTUARY:
			stmt := p.parseSanctuaryDecl()
			if stmt != nil {
				program.Statements = append(program.Statements, stmt)
			}
		case token.IMPORT:
			stmt := p.parseImportDecl()
			if stmt != nil {
				program.Statements = append(program.Statements, stmt)
			}
		case token.VAR:
			stmt := p.parseVarDecl()
			if stmt != nil {
				program.Statements = append(program.Statements, stmt)
			}
		case token.PR:
			stmt := p.parseProjectDecl()
			if stmt != nil {
				program.Statements = append(program.Statements, stmt)
			}
		case token.FN:
			stmt := p.parseFnDecl()
			if stmt != nil {
				program.Statements = append(program.Statements, stmt)
			}
		case token.SEQ:
			stmt := p.parseSeqDecl()
			if stmt != nil {
				program.Statements = append(program.Statements, stmt)
			}
		case token.PAR:
			stmt := p.parseParDecl()
			if stmt != nil {
				program.Statements = append(program.Statements, stmt)
			}
		default:
			p.errors = append(p.errors, fmt.Sprintf("unexpected token %s at %d:%d", p.curToken.Type, p.curToken.Line, p.curToken.Col))
			p.nextToken()
		}
	}

	return program
}

func (p *Parser) parseSanctuaryDecl() ast.Stmt {
	p.nextToken() // consume SANCTUARY
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
	return &ast.SanctuaryDecl{
		Token: p.curToken,
		Value: value,
	}
}

func (p *Parser) parseImportDecl() ast.Stmt {
	p.nextToken() // consume IMPORT
	if !p.expectPeek(token.LBRACKET) {
		return nil
	}
	paths := []string{}
	for {
		if !p.expectPeek(token.PATH_LIT) {
			return nil
		}
		paths = append(paths, p.curToken.Literal)
		if p.peekTokenIs(token.COMMA) {
			p.nextToken()
			continue
		}
		if p.peekTokenIs(token.RBRACKET) {
			break
		}
		p.errors = append(p.errors, fmt.Sprintf("expected ',' or ']' in import list at %d:%d", p.peekToken.Line, p.peekToken.Col))
		return nil
	}
	if !p.expectPeek(token.RBRACKET) {
		return nil
	}
	if !p.expectPeek(token.SEMICOLON) {
		return nil
	}
	return &ast.ImportDecl{
		Token: p.curToken,
		Paths: paths,
	}
}

func (p *Parser) parseVarDecl() ast.Stmt {
	p.nextToken() // consume VAR
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	name := p.curToken.Literal
	if !p.expectPeek(token.DECLARE) {
		return nil
	}
	var value ast.Expr
	switch p.peekToken.Type {
	case token.STRING_LIT:
		p.nextToken()
		value = &ast.StringLit{Token: p.curToken, Value: p.curToken.Literal}
	case token.DOLLAR:
		p.nextToken() // consume $
		if !p.expectPeek(token.IDENT) {
			return nil
		}
		value = &ast.VarRef{Token: p.curToken, Name: p.curToken.Literal}
	default:
		p.errors = append(p.errors, fmt.Sprintf("expected string or variable reference at %d:%d", p.peekToken.Line, p.peekToken.Col))
		return nil
	}
	if !p.expectPeek(token.SEMICOLON) {
		return nil
	}
	return &ast.VarDecl{
		Token: p.curToken,
		Name:  name,
		Value: value,
	}
}

func (p *Parser) parseProjectDecl() ast.Stmt {
	p.nextToken() // consume PR
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	name := p.curToken.Literal
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
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
	}
	if !p.expectPeek(token.RBRACE) {
		return nil
	}
	return &ast.ProjectDecl{
		Token:  p.curToken,
		Name:   name,
		Fields: fields,
	}
}

func (p *Parser) parseFnDecl() ast.Stmt {
	p.nextToken() // consume FN
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	name := p.curToken.Literal
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	body := p.parseFnBody()
	if !p.expectPeek(token.RBRACE) {
		return nil
	}
	return &ast.FnDecl{
		Token: p.curToken,
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
	}
	return stmts
}

func (p *Parser) parseLogStmt() ast.FnStmt {
	p.nextToken() // consume LOG
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	args := p.parseArgList()
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	if !p.expectPeek(token.SEMICOLON) {
		return nil
	}
	return &ast.LogStmt{
		Token: p.curToken,
		Args:  args,
	}
}

func (p *Parser) parseExecStmt() ast.FnStmt {
	p.nextToken() // consume EXEC
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	args := p.parseArgList()
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	if !p.expectPeek(token.SEMICOLON) {
		return nil
	}
	return &ast.ExecStmt{
		Token: p.curToken,
		Args:  args,
	}
}

func (p *Parser) parseArgList() []ast.Expr {
	args := []ast.Expr{}
	for {
		switch p.curToken.Type {
		case token.STRING_LIT:
			args = append(args, &ast.StringLit{Token: p.curToken, Value: p.curToken.Literal})
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
			p.nextToken()
			continue
		}
		break
	}
	return args
}

func (p *Parser) parseCdStmt() ast.FnStmt {
	p.nextToken() // consume CD
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	if !p.expectPeek(token.STRING_LIT) {
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
		Token: p.curToken,
		Arg:   arg,
	}
}

func (p *Parser) parseEnvBlock() ast.FnStmt {
	p.nextToken() // consume ENV
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
		if !p.expectPeek(token.STRING_LIT) {
			return nil
		}
		value := p.curToken.Literal
		pairs = append(pairs, ast.EnvPair{Key: key, Value: value})
		if p.peekTokenIs(token.COMMA) {
			p.nextToken()
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
	body := p.parseFnBody()
	if !p.expectPeek(token.RBRACE) {
		return nil
	}
	if !p.expectPeek(token.SEMICOLON) {
		return nil
	}
	return &ast.EnvBlock{
		Token: p.curToken,
		Pairs: pairs,
		Body:  body,
	}
}

func (p *Parser) parseSeqDecl() ast.Stmt {
	p.nextToken() // consume SEQ
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	name := p.curToken.Literal
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	stmts := []ast.SeqStmt{}
	for !p.curTokenIs(token.RBRACE) {
		if p.curTokenIs(token.EOF) {
			p.errors = append(p.errors, fmt.Sprintf("missing closing brace for seq at %d:%d", p.curToken.Line, p.curToken.Col))
			return &ast.SeqDecl{Name: name, Stmts: stmts}
		}
		var stmt ast.SeqStmt
		if p.curTokenIs(token.SEQ) && p.peekTokenIs(token.DOT) {
			stmt = p.parseSeqRef()
		} else {
			stmt = p.parseFnCall()
		}
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
	}
	if !p.expectPeek(token.RBRACE) {
		return nil
	}
	return &ast.SeqDecl{
		Token: p.curToken,
		Name:  name,
		Stmts: stmts,
	}
}

func (p *Parser) parseParDecl() ast.Stmt {
	p.nextToken() // consume PAR
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	name := p.curToken.Literal
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	stmts := []ast.ParStmt{}
	for !p.curTokenIs(token.RBRACE) {
		if p.curTokenIs(token.EOF) {
			p.errors = append(p.errors, fmt.Sprintf("missing closing brace for par at %d:%d", p.curToken.Line, p.curToken.Col))
			return &ast.ParDecl{Name: name, Stmts: stmts}
		}
		var stmt ast.ParStmt
		if p.curTokenIs(token.SEQ) && p.peekTokenIs(token.DOT) {
			stmt = p.parseSeqRef()
		} else {
			stmt = p.parseFnCall()
		}
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
	}
	if !p.expectPeek(token.RBRACE) {
		return nil
	}
	return &ast.ParDecl{
		Token: p.curToken,
		Name:  name,
		Stmts: stmts,
	}
}

func (p *Parser) parseFnCall() ast.SeqStmt {
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	fnName := p.curToken.Literal
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	if !p.expectPeek(token.PR) {
		return nil
	}
	if !p.expectPeek(token.DOT) {
		return nil
	}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	projectName := p.curToken.Literal
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	if !p.expectPeek(token.SEMICOLON) {
		return nil
	}
	return &ast.FnCall{
		Token:       p.curToken,
		FnName:      fnName,
		ProjectName: projectName,
	}
}

func (p *Parser) parseSeqRef() ast.SeqStmt {
	p.nextToken() // consume SEQ
	if !p.expectPeek(token.DOT) {
		return nil
	}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	seqName := p.curToken.Literal
	if !p.expectPeek(token.SEMICOLON) {
		return nil
	}
	return &ast.SeqRef{
		Token:   p.curToken,
		SeqName: seqName,
	}
}
