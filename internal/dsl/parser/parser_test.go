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
			input: `sanctuary = "$HOME/dev";`,
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
			input: `var port1 := "127.0.0.1:8080";`,
			wantStmts: map[token.TokenType]int{
				token.VAR: 1,
			},
		},
		{
			name:  "var with var ref",
			input: `var port1 := "a"; var port2 := $port1;`,
			wantStmts: map[token.TokenType]int{
				token.VAR: 2,
			},
		},
		{
			name:  "shell declaration",
			input: `shell = "bash";`,
			wantStmts: map[token.TokenType]int{
				token.SHELL: 1,
			},
		},
		{
			name:  "var with shell exec",
			input: `shell = "bash"; var x := ` + "`echo hello`;",
			wantStmts: map[token.TokenType]int{
				token.SHELL: 1,
				token.VAR:   1,
			},
		},
		{
			name:  "sanctuary with var ref",
			input: `shell = "bash"; var dir := ` + "`echo /tmp`" + `; sanctuary = $dir;`,
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
	input := `
pr todo {
    url = "git@github.com:yourname/todo.git";
    dir = "todo";
    sync = "clone";
    use = "./main.sro";
}`
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
		if f.Value != want {
			t.Fatalf("field %s: want %q, got %q", f.Key, want, f.Value)
		}
	}
}

func TestParseFnWithEnv(t *testing.T) {
	input := `
fn init {
    log("a");
    var x := "b";
    env [KEY = "val"] {
        log("c");
    };
}`
	l := lexer.New(input)
	p := New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("errors: %v", p.Errors())
	}
	if len(prog.Statements) != 1 {
		t.Fatalf("got %d statements", len(prog.Statements))
	}
	fn, ok := prog.Statements[0].(*ast.FnDecl)
	if !ok {
		t.Fatalf("expected FnDecl")
	}
	if len(fn.Body) != 3 {
		t.Fatalf("expected 3 fn stmts, got %d", len(fn.Body))
	}
	// Check first: log
	logStmt, ok := fn.Body[0].(*ast.LogStmt)
	if !ok {
		t.Fatalf("expected LogStmt at index 0, got %T", fn.Body[0])
	}
	if len(logStmt.Args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(logStmt.Args))
	}
	// Check second: var
	_, ok = fn.Body[1].(*ast.VarDecl)
	if !ok {
		t.Fatalf("expected VarDecl at index 1, got %T", fn.Body[1])
	}
	// Check third: env block
	env, ok := fn.Body[2].(*ast.EnvBlock)
	if !ok {
		t.Fatalf("expected EnvBlock at index 2, got %T", fn.Body[2])
	}
	if len(env.Pairs) != 1 {
		t.Fatalf("expected 1 env pair, got %d", len(env.Pairs))
	}
	if len(env.Body) != 1 {
		t.Fatalf("expected 1 stmt inside env body, got %d", len(env.Body))
	}
}

func TestParseSeqAndPar(t *testing.T) {
	input := `
seq build {
    check(pr.todo);
    build(pr.todo);
}
par test {
    seq.init;
    test(pr.calendar);
}`
	l := lexer.New(input)
	p := New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("errors: %v", p.Errors())
	}
	// Validate seq
	seq, ok := prog.Statements[0].(*ast.SeqDecl)
	if !ok {
		t.Fatalf("expected SeqDecl")
	}
	if len(seq.Stmts) != 2 {
		t.Fatalf("expected 2 seq stmts, got %d", len(seq.Stmts))
	}
	// first is FnCall
	_, ok = seq.Stmts[0].(*ast.FnCall)
	if !ok {
		t.Fatalf("expected FnCall in seq, got %T", seq.Stmts[0])
	}
	// second is FnCall as well
	_, ok = seq.Stmts[1].(*ast.FnCall)
	if !ok {
		t.Fatalf("expected FnCall in seq, got %T", seq.Stmts[1])
	}
	// Validate par
	par, ok := prog.Statements[1].(*ast.ParDecl)
	if !ok {
		t.Fatalf("expected ParDecl")
	}
	if len(par.Stmts) != 2 {
		t.Fatalf("expected 2 par stmts, got %d", len(par.Stmts))
	}
	// first: SeqRef
	_, ok = par.Stmts[0].(*ast.SeqRef)
	if !ok {
		t.Fatalf("expected SeqRef in par, got %T", par.Stmts[0])
	}
	// second: FnCall
	_, ok = par.Stmts[1].(*ast.FnCall)
	if !ok {
		t.Fatalf("expected FnCall in par, got %T", par.Stmts[1])
	}
}

func TestFullFile(t *testing.T) {
	input := `sanctuary = "$HOME/dev";
import [ ./other_config.sro, ./example/work.sro ];
var port1 := "127.0.0.1:8080";
var port2 := "192.168.1.3:2425";
var port3 := $port1;
var idx_port := "3";
pr todo {
    url = "git@github.com:yourname/todo.git";
    dir = "todo";
    sync = "clone";
    use = "./main.sro";
}
fn init {
    log("Installing dependencies!");
    var deps := "4";
    log("Currently we have", $deps, "dependencies!");
    cd("cmd");
    env [
          GOFLAGS = "-mod=mod",
          CGO_ENABLED = "0",
          DB_URL = "postgres://localhost:5432"
        ] {
          env [CGO_ENABLED = "1"] {
            exec("go build .");
          };
          exec("go mod download");
          exec("go generate ./...");
        };
    cd(".");
    exec("go test ./...");
    exec("staticcheck ./...");
}
seq init {
    check(pr.todo);
    init(pr.calendar-ts);
}
par ci {
    build(pr.todo);
    seq.init;
}`
	l := lexer.New(input)
	p := New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors:\n%s", p.Errors())
	}
	// Count top-level declarations
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
		default:
			t.Fatalf("unknown statement type: %T", stmt)
		}
	}
	expected := map[token.TokenType]int{
		token.SANCTUARY: 1,
		token.IMPORT:    1,
		token.VAR:       4,
		token.PR:        1,
		token.FN:        1,
		token.SEQ:       1,
		token.PAR:       1,
	}
	for k, want := range expected {
		got := counts[k]
		if got != want {
			t.Fatalf("count %s: want %d, got %d", k, want, got)
		}
	}
	// Spot-check some nodes:
	// - Sanctuary value
	san, _ := prog.Statements[0].(*ast.SanctuaryDecl)
	if sl, ok := san.Value.(*ast.StringLit); !ok || sl.Value != "$HOME/dev" {
		t.Fatalf("sanctuary value wrong")
	}
	// - Import paths
	imp, _ := prog.Statements[1].(*ast.ImportDecl)
	if len(imp.Paths) != 2 {
		t.Fatalf("import paths count: want 2, got %d", len(imp.Paths))
	}
	// - VarRef in var port3
	var3, _ := prog.Statements[4].(*ast.VarDecl)
	if vr, ok := var3.Value.(*ast.VarRef); !ok || vr.Name != "port1" {
		t.Fatalf("var port3 value incorrect")
	}
	// - fn body has env block with nested env
	fn, _ := prog.Statements[7].(*ast.FnDecl)
	envBlock, _ := fn.Body[4].(*ast.EnvBlock) // fifth statement in fn body
	if len(envBlock.Body) != 3 {
		t.Fatalf("outer env body len: want 3, got %d", len(envBlock.Body))
	}
	nestedEnv, _ := envBlock.Body[0].(*ast.EnvBlock) // first stmt in outer env
	if len(nestedEnv.Body) != 1 {
		t.Fatalf("nested env body len: want 1, got %d", len(nestedEnv.Body))
	}
}

func TestErrorCases(t *testing.T) {
	tests := []struct {
		input       string
		wantErrSubj string
	}{
		{`sanctuary = "$HOME"`, "expected ;"}, // missing semicolon
		{`pr x {`, "missing closing brace"},
		{`fn bad { unknown }`, "unexpected token"},
		{`pr x { url = "x"; unknown = "y";`, "invalid project field key"},
		{`seq s { par.x; }`, "par blocks cannot be referenced"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			p.ParseProgram()
			if len(p.Errors()) == 0 {
				t.Fatalf("expected error containing %q, got none", tt.wantErrSubj)
			}
			found := false
			for _, err := range p.Errors() {
				if strings.Contains(err, tt.wantErrSubj) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected error containing %q, got %v", tt.wantErrSubj, p.Errors())
			}

		})
	}
}
