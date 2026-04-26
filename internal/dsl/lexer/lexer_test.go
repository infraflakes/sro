package lexer

import (
	"testing"

	"github.com/infraflakes/sro/internal/dsl/token"
)

func TestNextToken(t *testing.T) {
	tests := []struct {
		input    string
		expected []token.Token
	}{
		// Single tokens
		{"=", []token.Token{{Type: token.ASSIGN, Literal: "="}}},
		{":=", []token.Token{{Type: token.DECLARE, Literal: ":="}}},
		{"{", []token.Token{{Type: token.LBRACE, Literal: "{"}}},
		{"}", []token.Token{{Type: token.RBRACE, Literal: "}"}}},
		{"[", []token.Token{{Type: token.LBRACKET, Literal: "["}}},
		{"]", []token.Token{{Type: token.RBRACKET, Literal: "]"}}},
		{"(", []token.Token{{Type: token.LPAREN, Literal: "("}}},
		{")", []token.Token{{Type: token.RPAREN, Literal: ")"}}},
		{",", []token.Token{{Type: token.COMMA, Literal: ","}}},
		{".", []token.Token{{Type: token.DOT, Literal: "."}}},
		{";", []token.Token{{Type: token.SEMICOLON, Literal: ";"}}},
		{"$", []token.Token{{Type: token.DOLLAR, Literal: "$"}}},

		// Keywords
		{"sanctuary", []token.Token{{Type: token.SANCTUARY, Literal: "sanctuary"}}},
		{"import", []token.Token{{Type: token.IMPORT, Literal: "import"}}},
		{"var", []token.Token{{Type: token.VAR, Literal: "var"}}},
		{"pr", []token.Token{{Type: token.PR, Literal: "pr"}}},
		{"fn", []token.Token{{Type: token.FN, Literal: "fn"}}},
		{"seq", []token.Token{{Type: token.SEQ, Literal: "seq"}}},
		{"par", []token.Token{{Type: token.PAR, Literal: "par"}}},
		{"env", []token.Token{{Type: token.ENV, Literal: "env"}}},
		{"log", []token.Token{{Type: token.LOG, Literal: "log"}}},
		{"exec", []token.Token{{Type: token.EXEC, Literal: "exec"}}},
		{"cd", []token.Token{{Type: token.CD, Literal: "cd"}}},
		{"shell", []token.Token{{Type: token.SHELL, Literal: "shell"}}},

		// Identifiers
		{"todo", []token.Token{{Type: token.IDENT, Literal: "todo"}}},
		{"calendar-ts", []token.Token{{Type: token.IDENT, Literal: "calendar-ts"}}},
		{"port1", []token.Token{{Type: token.IDENT, Literal: "port1"}}},
		{"idx_port", []token.Token{{Type: token.IDENT, Literal: "idx_port"}}},
		{"url", []token.Token{{Type: token.IDENT, Literal: "url"}}}, // not a keyword

		// String literals
		{`"hello"`, []token.Token{{Type: token.STRING_LIT, Literal: "hello"}}},
		{`""`, []token.Token{{Type: token.STRING_LIT, Literal: ""}}},

		// Shell literals (backticks)
		{"`echo hello`", []token.Token{{Type: token.SHELL_LIT, Literal: "echo hello"}}},
		{"``", []token.Token{{Type: token.SHELL_LIT, Literal: ""}}},

		// Path literals
		{"./other_config.sro", []token.Token{{Type: token.PATH_LIT, Literal: "./other_config.sro"}}},
		{"./example/work.sro", []token.Token{{Type: token.PATH_LIT, Literal: "./example/work.sro"}}},

		// Variable references
		{"$port1", []token.Token{{Type: token.DOLLAR, Literal: "$"}, {Type: token.IDENT, Literal: "port1"}}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := New(tt.input)
			var i int
			for {
				tok := l.NextToken()
				if tok.Type == token.EOF {
					break
				}
				if i >= len(tt.expected) {
					t.Fatalf("expected %d tokens, got more", len(tt.expected))
				}
				if tok.Type != tt.expected[i].Type || tok.Literal != tt.expected[i].Literal {
					t.Fatalf("token %d: expected Type=%q, Literal=%q, got Type=%q, Literal=%q at line %d, col %d",
						i, tt.expected[i].Type, tt.expected[i].Literal, tok.Type, tok.Literal, tok.Line, tok.Col)
				}
				i++
			}
			if i != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d", len(tt.expected), i)
			}
		})
	}
}

func TestLexerLineCol(t *testing.T) {
	input := `var x := "hello";
var y := "world";`
	l := New(input)
	tokens := []token.Token{}
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == token.EOF {
			break
		}
	}

	// SANCTUARY not present, so first tokens should be VAR at line 1 col 1
	if tokens[0].Type != token.VAR || tokens[0].Line != 1 || tokens[0].Col != 1 {
		t.Fatalf("first token wrong: got line=%d col=%d", tokens[0].Line, tokens[0].Col)
	}

	// Find VAR y
	var secondVar token.Token
	for _, tok := range tokens {
		if tok.Type == token.VAR && tok.Literal == "var" && tok.Line == 2 {
			secondVar = tok
			break
		}
	}
	if secondVar.Line != 2 || secondVar.Col != 1 {
		t.Fatalf("second var token wrong: got line=%d col=%d", secondVar.Line, secondVar.Col)
	}
}

func TestComments(t *testing.T) {
	input := `var x := "test"; # comment
var y := "next";`
	l := New(input)
	tokens := []token.Token{}
	for {
		tok := l.NextToken()
		if tok.Type == token.EOF {
			break
		}
		tokens = append(tokens, tok)
	}

	// Should have 10 tokens: VAR, IDENT, DECLARE, STRING_LIT, SEMICOLON, VAR, IDENT, DECLARE, STRING_LIT, SEMICOLON
	expectedTypes := []token.TokenType{token.VAR, token.IDENT, token.DECLARE, token.STRING_LIT, token.SEMICOLON, token.VAR, token.IDENT, token.DECLARE, token.STRING_LIT, token.SEMICOLON}
	if len(tokens) != len(expectedTypes) {
		t.Fatalf("token count mismatch: got %d, want %d", len(tokens), len(expectedTypes))
	}
	for i, tt := range expectedTypes {
		if tokens[i].Type != tt {
			t.Fatalf("token %d: expected %q, got %q", i, tt, tokens[i].Type)
		}
	}
}

func TestErrorCases(t *testing.T) {
	tests := []struct {
		input  string
		errMsg string
	}{
		{`"unterminated`, "unterminated string"},
		{"bare:", "unexpected character: :"},
		{`"test`, "unterminated string"},
		{"@", "unexpected character: @"},
		{"`unterminated", "unterminated shell string"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := New(tt.input)
			var foundErr bool
			for {
				tok := l.NextToken()
				if tok.Type == token.EOF {
					break
				}
				if tok.Type == token.ILLEGAL && tok.Literal == tt.errMsg {
					foundErr = true
					break
				}
			}
			if !foundErr {
				t.Errorf("expect error %q, got none or different", tt.errMsg)
			}
		})
	}
}

func TestFullSnippet(t *testing.T) {
	input := `sanctuary = "$HOME/dev";
import [ ./a.sro, ./b.sro ];
var port1 := "127.0.0.1:8080";
pr hello {
    url = "git@github.com:foo/bar.git";
    dir = "bar";
}
fn init {
    log("starting");
    exec("go build");
}`
	l := New(input)
	tokens := []token.Token{}
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == token.EOF {
			break
		}
	}

	// Sanity: check some token counts/types
	var sanctuary, importTok, varTok, prTok, fnTok, logTok, execTok token.Token
	for _, tok := range tokens {
		switch tok.Type {
		case token.SANCTUARY:
			sanctuary = tok
		case token.IMPORT:
			importTok = tok
		case token.VAR:
			varTok = tok
		case token.PR:
			prTok = tok
		case token.FN:
			fnTok = tok
		case token.LOG:
			logTok = tok
		case token.EXEC:
			execTok = tok
		}
	}

	if sanctuary.Type != token.SANCTUARY {
		t.Fatalf("missing SANCTUARY token")
	}
	if importTok.Type != token.IMPORT {
		t.Fatalf("missing IMPORT token")
	}
	if varTok.Type != token.VAR {
		t.Fatalf("missing VAR token")
	}
	if prTok.Type != token.PR {
		t.Fatalf("missing PR token")
	}
	if fnTok.Type != token.FN {
		t.Fatalf("missing FN token")
	}
	if logTok.Type != token.LOG {
		t.Fatalf("missing LOG token")
	}
	if execTok.Type != token.EXEC {
		t.Fatalf("missing EXEC token")
	}
}
