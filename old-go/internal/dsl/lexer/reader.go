package lexer

import (
	"strings"
	"unicode"

	"github.com/infraflakes/sro/internal/dsl/token"
)

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

func (l *Lexer) readBacktick() token.Token {
	line := l.line
	col := l.col
	l.readChar() // consume opening `

	var lit strings.Builder
	for l.ch != '`' && l.ch != 0 {
		lit.WriteByte(l.ch)
		l.readChar()
	}

	if l.ch != '`' {
		return token.Token{
			Type:    token.ILLEGAL,
			Literal: "unterminated backtick string",
			Line:    line,
			Col:     col,
		}
	}

	return token.Token{
		Type:    token.BACKTICK,
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
