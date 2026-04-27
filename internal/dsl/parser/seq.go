package parser

import (
	"github.com/infraflakes/sro/internal/dsl/ast"
	"github.com/infraflakes/sro/internal/dsl/token"
)

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
			p.errors = append(p.errors, ParseError{
				Message: "missing closing brace for seq",
				Line:    p.curToken.Line,
				Col:     p.curToken.Col,
			})
			return &ast.SeqDecl{Token: tok, Name: name, Stmts: stmts}
		}
		var stmt ast.SeqStmt
		if p.curTokenIs(token.SEQ) && p.peekTokenIs(token.DOT) {
			stmt = p.parseSeqRef()
		} else if p.curTokenIs(token.PAR) && p.peekTokenIs(token.DOT) {
			p.errors = append(p.errors, ParseError{
				Message: "par blocks cannot be referenced, use CLI to run par",
				Line:    p.curToken.Line,
				Col:     p.curToken.Col,
			})
			p.nextToken() // .
			p.nextToken() // X
			p.nextToken() // ;
			// fall through — stmt is nil, bottom p.nextToken() will advance further if needed
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
			p.errors = append(p.errors, ParseError{
				Message: "missing closing brace for par",
				Line:    p.curToken.Line,
				Col:     p.curToken.Col,
			})
			return &ast.ParDecl{Token: tok, Name: name, Stmts: stmts}
		}
		var stmt ast.ParStmt
		if p.curTokenIs(token.SEQ) && p.peekTokenIs(token.DOT) {
			stmt = p.parseSeqRef()
		} else if p.curTokenIs(token.PAR) && p.peekTokenIs(token.DOT) {
			p.errors = append(p.errors, ParseError{
				Message: "par blocks cannot be referenced, use CLI to run par",
				Line:    p.curToken.Line,
				Col:     p.curToken.Col,
			})
			p.nextToken() // .
			p.nextToken() // X
			p.nextToken() // ;
			// fall through — stmt is nil, bottom p.nextToken() will advance further if needed
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
