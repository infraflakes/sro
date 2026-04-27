package parser

import (
	"strings"
	"testing"

	"github.com/infraflakes/sro/internal/dsl/ast"
	"github.com/infraflakes/sro/internal/dsl/lexer"
	"github.com/infraflakes/sro/internal/dsl/token"
)

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

func TestParseEmptySeqAndPar(t *testing.T) {
	// P4: empty seq/par bodies
	input := "\nseq empty {}\npar empty {}"
	l := lexer.New(input)
	p := New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("errors: %v", p.Errors())
	}
	if len(prog.Statements) != 2 {
		t.Fatalf("got %d statements", len(prog.Statements))
	}
	seq, ok := prog.Statements[0].(*ast.SeqDecl)
	if !ok {
		t.Fatalf("expected SeqDecl")
	}
	if len(seq.Stmts) != 0 {
		t.Fatalf("expected 0 seq stmts, got %d", len(seq.Stmts))
	}
	par, ok := prog.Statements[1].(*ast.ParDecl)
	if !ok {
		t.Fatalf("expected ParDecl")
	}
	if len(par.Stmts) != 0 {
		t.Fatalf("expected 0 par stmts, got %d", len(par.Stmts))
	}
}

func TestFullFile(t *testing.T) {
	input := "sanctuary = `$HOME/dev`;\nimport [ ./other_config.sro, ./example/work.sro ];\nvar string port1 = `127.0.0.1:8080`;\nvar string port2 = `192.168.1.3:2425`;\nvar string port3 = `${port1}`;\nvar string idx_port = `3`;\npr todo {\n    url = `git@github.com:yourname/todo.git`;\n    dir = `todo`;\n    sync = `clone`;\n    use = `./main.sro`;\n}\nfn init {\n    log(`Installing dependencies!`);\n    var string deps = `4`;\n    log(`Currently we have ${deps} dependencies!`);\n    cd(`cmd`);\n    env [\n          GOFLAGS = `-mod=mod`,\n          CGO_ENABLED = `0`,\n          DB_URL = `postgres://localhost:5432`\n        ] {\n          env [CGO_ENABLED = `1`] {\n            exec(`go build .`);\n          };\n          exec(`go mod download`);\n          exec(`go generate ./...`);\n        };\n    cd(`.`);\n    exec(`go test ./...`);\n    exec(`staticcheck ./...`);\n}\nseq init {\n    check(pr.todo);\n    init(pr.calendar-ts);\n}\npar ci {\n    build(pr.todo);\n    seq.init;\n}"
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
	if bl, ok := san.Value.(*ast.BacktickLit); !ok || len(bl.Parts) != 1 || bl.Parts[0].Value != "$HOME/dev" {
		t.Fatalf("sanctuary value wrong")
	}
	// - Import paths
	imp, _ := prog.Statements[1].(*ast.ImportDecl)
	if len(imp.Paths) != 2 {
		t.Fatalf("import paths count: want 2, got %d", len(imp.Paths))
	}
	// - VarRef in var port3
	var3, _ := prog.Statements[4].(*ast.VarDecl)
	bl, ok := var3.Value.(*ast.BacktickLit)
	if !ok || len(bl.Parts) != 1 || !bl.Parts[0].IsVar || bl.Parts[0].Value != "port1" {
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
		{"sanctuary = `$HOME`", "expected ';'"}, // missing semicolon
		{"pr x {", "missing closing brace"},
		{"fn bad { unknown }", "unexpected token"},
		{"pr x { url = `x`; unknown = `y`;", "invalid project field key"},
		{"seq s { par.x; }", "par blocks cannot be referenced"},
		{"fn bad", "expected {"},                                // P11: missing { after fn name
		{"seq bad", "expected {"},                               // P11: missing { after seq name
		{"par bad", "expected {"},                               // P11: missing { after par name
		{"par p { par.x; }", "par blocks cannot be referenced"}, // P12: par.X in par body
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
				if strings.Contains(err.Error(), tt.wantErrSubj) {
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
