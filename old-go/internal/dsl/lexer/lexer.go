package lexer

import (
	"fmt"
	"unicode"

	"github.com/infraflakes/sro/internal/dsl/token"
)

type Lexer struct {
	input   string
	pos     int
	readPos int
	ch      byte
	line    int
	col     int
}

func New(input string) *Lexer {
	l := &Lexer{
		input: input,
		line:  1,
		col:   0,
	}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++
	if l.ch == '\n' {
		l.line++
		l.col = 0
	} else {
		l.col++
	}
}

func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	for {
		l.skipWhitespace()
		if l.ch != '#' {
			break
		}
		l.skipComment()
	}

	tok.Line = l.line
	tok.Col = l.col

	switch l.ch {
	case 0:
		tok = l.makeToken(token.EOF)
	case '{':
		tok = l.makeToken(token.LBRACE)
	case '}':
		tok = l.makeToken(token.RBRACE)
	case '[':
		tok = l.makeToken(token.LBRACKET)
	case ']':
		tok = l.makeToken(token.RBRACKET)
	case '(':
		tok = l.makeToken(token.LPAREN)
	case ')':
		tok = l.makeToken(token.RPAREN)
	case ',':
		tok = l.makeToken(token.COMMA)
	case '.':
		if l.peek() == '/' {
			return l.readPath()
		} else {
			tok = l.makeToken(token.DOT)
		}
	case ';':
		tok = l.makeToken(token.SEMICOLON)
	case '$':
		tok = l.makeToken(token.DOLLAR)
	case '=':
		tok = l.makeToken(token.ASSIGN)
	case ':':
		tok = l.makeToken(token.ILLEGAL)
		tok.Literal = "unexpected character: :"
	case '`':
		tok = l.readBacktick()
	default:
		if unicode.IsLetter(rune(l.ch)) || l.ch == '_' {
			tok = l.readIdent()
			return tok
		} else {
			tok = l.makeToken(token.ILLEGAL)
			tok.Literal = fmt.Sprintf("unexpected character: %c", l.ch)
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) makeToken(tt token.TokenType) token.Token {
	return token.Token{
		Type:    tt,
		Literal: string(l.ch),
		Line:    l.line,
		Col:     l.col,
	}
}
