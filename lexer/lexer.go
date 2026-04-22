package lexer

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/infraflakes/sro/token"
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
		if l.peek() == '=' {
			l.readChar()
			l.readChar()
			tok.Type = token.DECLARE
			tok.Literal = ":="
			tok.Line = l.line
			tok.Col = l.col - 2
			return tok
		} else {
			tok = l.makeToken(token.ILLEGAL)
			tok.Literal = "unexpected character: :"
		}
	case '"':
		tok = l.readString()
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

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' || l.ch == '\n' {
		l.readChar()
	}
}

func (l *Lexer) skipComment() {
	if l.ch != '#' {
		return
	}
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}

func (l *Lexer) peek() byte {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

func (l *Lexer) readString() token.Token {
	line := l.line
	col := l.col
	l.readChar() // consume opening quote

	var lit strings.Builder
	for l.ch != '"' && l.ch != 0 {
		switch l.ch {
		case '\\':
			l.readChar()
			switch l.ch {
			case '"':
				lit.WriteByte('"')
			case '\\':
				lit.WriteByte('\\')
			case 'n':
				lit.WriteByte('\n')
			case 't':
				lit.WriteByte('\t')
			default:
				lit.WriteByte('\\')
				lit.WriteByte(l.ch)
			}
		case '\n':
			lit.WriteByte('\n')
		default:
			lit.WriteByte(l.ch)
		}
		l.readChar()
	}

	if l.ch != '"' {
		return token.Token{
			Type:    token.ILLEGAL,
			Literal: "unterminated string",
			Line:    line,
			Col:     col,
		}
	}

	return token.Token{
		Type:    token.STRING_LIT,
		Literal: lit.String(),
		Line:    line,
		Col:     col,
	}
}

func (l *Lexer) readPath() token.Token {
	line := l.line
	col := l.col
	l.readChar() // consume '.'
	l.readChar() // consume '/'

	var lit strings.Builder
	lit.WriteString("./")

	for l.ch != 0 && l.ch != ' ' && l.ch != '\t' && l.ch != '\n' && l.ch != ',' && l.ch != ']' && l.ch != ';' {
		lit.WriteByte(l.ch)
		l.readChar()
	}

	return token.Token{
		Type:    token.PATH_LIT,
		Literal: lit.String(),
		Line:    line,
		Col:     col,
	}
}

func (l *Lexer) readIdent() token.Token {
	line := l.line
	col := l.col

	var lit strings.Builder
	for l.ch != 0 && (unicode.IsLetter(rune(l.ch)) || l.ch == '_' || l.ch == '-' || unicode.IsDigit(rune(l.ch))) {
		lit.WriteByte(l.ch)
		l.readChar()
	}

	return token.Token{
		Type:    token.LookupIdent(lit.String()),
		Literal: lit.String(),
		Line:    line,
		Col:     col,
	}
}
