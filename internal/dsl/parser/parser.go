package parser

import (
	"fmt"

	"github.com/infraflakes/sro/internal/dsl/ast"
	"github.com/infraflakes/sro/internal/dsl/lexer"
	"github.com/infraflakes/sro/internal/dsl/token"
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
		}
		p.nextToken() // advance past failed keyword or statement delimiter
	}

	return program
}
