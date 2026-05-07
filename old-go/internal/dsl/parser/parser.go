package parser

import (
	"fmt"

	"github.com/infraflakes/sro/internal/dsl/ast"
	"github.com/infraflakes/sro/internal/dsl/lexer"
	"github.com/infraflakes/sro/internal/dsl/token"
)

type ParseError struct {
	Message string
	Line    int
	Col     int
}

func (e ParseError) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Line, e.Col, e.Message)
}

type Parser struct {
	l         *lexer.Lexer
	curToken  token.Token
	peekToken token.Token
	errors    []ParseError
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []ParseError{},
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
	p.errors = append(p.errors, ParseError{
		Message: fmt.Sprintf("expected %s, got %s", t, p.peekToken.Type),
		Line:    p.peekToken.Line,
		Col:     p.peekToken.Col,
	})
}

func (p *Parser) Errors() []ParseError {
	return p.errors
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{Statements: []ast.Stmt{}}

	for p.curToken.Type != token.EOF {
		var stmt ast.Stmt
		switch p.curToken.Type {
		case token.SHELL:
			stmt = p.parseShellDecl()
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
			p.errors = append(p.errors, ParseError{
				Message: fmt.Sprintf("unexpected token %s", p.curToken.Type),
				Line:    p.curToken.Line,
				Col:     p.curToken.Col,
			})
			p.nextToken()
			continue
		}
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken() // advance past failed keyword or statement delimiter
	}

	return program
}
