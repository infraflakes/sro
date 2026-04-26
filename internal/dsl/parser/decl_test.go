package parser

import (
	"slices"
	"strings"
	"testing"

	"github.com/infraflakes/sro/internal/dsl/ast"
	"github.com/infraflakes/sro/internal/dsl/lexer"
	"github.com/infraflakes/sro/internal/dsl/token"
)

func TestParseProgram(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantStmts   map[token.TokenType]int // count of each top-level stmt type
		wantErr     bool
		errContains string
	}{
		{
			name:  "sanctuary declaration",
			input: "sanctuary = `$HOME/dev`;",
			wantStmts: map[token.TokenType]int{
				token.SANCTUARY: 1,
			},
		},
		{
			name:  "import declaration",
			input: `import [ ./a.sro, ./b.sro ];`,
			wantStmts: map[token.TokenType]int{
				token.IMPORT: 1,
			},
		},
		{
			name:  "var with string",
			input: "var string port1 = `127.0.0.1:8080`;",
			wantStmts: map[token.TokenType]int{
				token.VAR: 1,
			},
		},
		{
			name:  "var with var ref",
			input: "var string port1 = `a`; var string port2 = `${port1}`;",
			wantStmts: map[token.TokenType]int{
				token.VAR: 2,
			},
		},
		{
			name:  "shell declaration",
			input: "shell = `bash`;",
			wantStmts: map[token.TokenType]int{
				token.SHELL: 1,
			},
		},
		{
			name:  "var with shell exec",
			input: "shell = `bash`; var shell x = `echo hello`;",
			wantStmts: map[token.TokenType]int{
				token.SHELL: 1,
				token.VAR:   1,
			},
		},
		{
			name:  "sanctuary with var ref",
			input: "shell = `bash`; var shell dir = `echo /tmp`; sanctuary = `${dir}`;",
			wantStmts: map[token.TokenType]int{
				token.SHELL:     1,
				token.VAR:       1,
				token.SANCTUARY: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			prog := p.ParseProgram()

			if len(p.Errors()) > 0 && !tt.wantErr {
				t.Fatalf("unexpected errors: %v", p.Errors())
			}
			if tt.wantErr {
				if p.Errors() == nil {
					t.Fatalf("expected error containing %q, got none", tt.errContains)
				}
				found := slices.Contains(p.Errors(), tt.errContains)
				if !found {
					t.Fatalf("expected error containing %q, got %v", tt.errContains, p.Errors())
				}
				return
			}

			counts := make(map[token.TokenType]int)
			for _, stmt := range prog.Statements {
				switch stmt.(type) {
				case *ast.SanctuaryDecl:
					counts[token.SANCTUARY]++
				case *ast.ImportDecl:
					counts[token.IMPORT]++
				case *ast.VarDecl:
					counts[token.VAR]++
				case *ast.ProjectDecl:
					counts[token.PR]++
				case *ast.FnDecl:
					counts[token.FN]++
				case *ast.SeqDecl:
					counts[token.SEQ]++
				case *ast.ParDecl:
					counts[token.PAR]++
				case *ast.ShellDecl:
					counts[token.SHELL]++
				}
			}
			for k, want := range tt.wantStmts {
				got := counts[k]
				if got != want {
					t.Fatalf("count for %s: want %d, got %d", k, want, got)
				}
			}
		})
	}
}

func TestParseProjectDecl(t *testing.T) {
	input := "\npr todo {\n    url = `git@github.com:yourname/todo.git`;\n    dir = `todo`;\n    sync = `clone`;\n    use = `./main.sro`;\n}"
	l := lexer.New(input)
	p := New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	if len(prog.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(prog.Statements))
	}
	proj, ok := prog.Statements[0].(*ast.ProjectDecl)
	if !ok {
		t.Fatalf("expected *ProjectDecl, got %T", prog.Statements[0])
	}
	if proj.Name != "todo" {
		t.Fatalf("project name: want 'todo', got %s", proj.Name)
	}
	if len(proj.Fields) != 4 {
		t.Fatalf("expected 4 fields, got %d", len(proj.Fields))
	}
	expected := map[string]string{
		"url":  "git@github.com:yourname/todo.git",
		"dir":  "todo",
		"sync": "clone",
		"use":  "./main.sro",
	}
	for _, f := range proj.Fields {
		want, ok := expected[f.Key]
		if !ok {
			t.Fatalf("unexpected field key: %s", f.Key)
		}
		// f.Value is now ast.Expr, need to resolve it
		var got strings.Builder
		switch v := f.Value.(type) {
		case *ast.BacktickLit:
			// For simple literals without interpolation, just join the parts
			for _, part := range v.Parts {
				got.WriteString(part.Value)
			}
		default:
			t.Fatalf("unexpected value type: %T", f.Value)
		}
		if got.String() != want {
			t.Fatalf("field %s: want %q, got %q", f.Key, want, got.String())
		}
	}
}
