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
		var stmt ast.Stmt
		switch p.curToken.Type {
		case token.SANCTUARY:
			stmt = p.parseSanctuaryDecl()
		case token.IMPORT:
			stmt = p.parseImportDecl()
		case token.VAR:
			stmt = p.parseVarDecl()
		case token.PR:
			stmt = p.parseProjectDecl()
		case token.FN:
			stmt = p.parseFnDecl()
		case token.SEQ:
			stmt = p.parseSeqDecl()
		case token.PAR:
			stmt = p.parseParDecl()
		default:
			p.errors = append(p.errors, fmt.Sprintf("unexpected token %s at %d:%d", p.curToken.Type, p.curToken.Line, p.curToken.Col))
			p.nextToken()
			continue
		}
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
			p.nextToken() // advance past statement token (semicolon or })
		}
	}

	return program
}

func (p *Parser) parseSanctuaryDecl() ast.Stmt {
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
	return &ast.SanctuaryDecl{
		Token: tok,
		Value: value,
	}
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
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	if !p.expectPeek(token.SEMICOLON) {
		return nil
	}
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
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	if !p.expectPeek(token.SEMICOLON) {
		return nil
	}
	return &ast.ExecStmt{
		Token: tok,
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
			p.nextToken() // to COMMA
			p.nextToken() // to next argument
			continue
		}
		break
	}
	return args
}

func (p *Parser) parseCdStmt() ast.FnStmt {
	tok := p.curToken
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

func (p *Parser) parseSeqDecl() ast.Stmt {
	tok := p.curToken
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	name := p.curToken.Literal
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	p.nextToken() // advance past {
	stmts := []ast.SeqStmt{}
	for !p.curTokenIs(token.RBRACE) {
		if p.curTokenIs(token.EOF) {
			p.errors = append(p.errors, fmt.Sprintf("missing closing brace for seq at %d:%d", p.curToken.Line, p.curToken.Col))
			return &ast.SeqDecl{Token: tok, Name: name, Stmts: stmts}
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
		p.nextToken() // advance past ; to next token or }
	}
	// curToken is RBRACE
	return &ast.SeqDecl{
		Token: tok,
		Name:  name,
		Stmts: stmts,
	}
}

func (p *Parser) parseParDecl() ast.Stmt {
	tok := p.curToken
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	name := p.curToken.Literal
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	p.nextToken() // advance past {
	stmts := []ast.ParStmt{}
	for !p.curTokenIs(token.RBRACE) {
		if p.curTokenIs(token.EOF) {
			p.errors = append(p.errors, fmt.Sprintf("missing closing brace for par at %d:%d", p.curToken.Line, p.curToken.Col))
			return &ast.ParDecl{Token: tok, Name: name, Stmts: stmts}
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
		p.nextToken() // advance past ; to next token or }
	}
	// curToken is RBRACE
	return &ast.ParDecl{
		Token: tok,
		Name:  name,
		Stmts: stmts,
	}
}

func (p *Parser) parseFnCall() *ast.FnCall {
	fnNameToken := p.curToken
	fnName := fnNameToken.Literal
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
		Token:       fnNameToken,
		FnName:      fnName,
		ProjectName: projectName,
	}
}

func (p *Parser) parseSeqRef() *ast.SeqRef {
	seqToken := p.curToken
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
		Token:   seqToken,
		SeqName: seqName,
	}
}
