package parser

import (
	"testing"

	"github.com/infraflakes/sro/internal/dsl/ast"
	"github.com/infraflakes/sro/internal/dsl/lexer"
)

func TestParseFnWithEnv(t *testing.T) {
	input := "\nfn init {\n    log(`a`);\n    var string x = `b`;\n    env [KEY = `val`] {\n        log(`c`);\n    };\n}"
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
	if logStmt.Value == nil {
		t.Fatalf("expected Value to be non-nil")
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

func TestParseEnvWithTrailingComma(t *testing.T) {
	input := "\nfn init {\n    env [X = `1`,] {\n        log(`x`);\n    };\n}"
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
	if len(fn.Body) != 1 {
		t.Fatalf("expected 1 fn stmt, got %d", len(fn.Body))
	}
	env, ok := fn.Body[0].(*ast.EnvBlock)
	if !ok {
		t.Fatalf("expected EnvBlock, got %T", fn.Body[0])
	}
	if len(env.Pairs) != 1 {
		t.Fatalf("expected 1 env pair, got %d", len(env.Pairs))
	}
	if env.Pairs[0].Key != "X" {
		t.Fatalf("expected key X, got %s", env.Pairs[0].Key)
	}
}

func TestParseEmptyFnBody(t *testing.T) {
	// P3: empty fn body
	input := "\nfn empty {}"
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
	if len(fn.Body) != 0 {
		t.Fatalf("expected 0 fn stmts, got %d", len(fn.Body))
	}
}

func TestParseEmptyEnvPairs(t *testing.T) {
	// P6: empty env pairs - this is actually a syntax error in the language
	// The parser expects at least one env pair inside []
	// So this test documents that empty env pairs are not allowed
	input := "\nfn init {\n    env [] { log(`x`); };\n}"
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	// This should produce a parse error
	if len(p.Errors()) == 0 {
		t.Fatalf("expected parse error for empty env pairs")
	}
}

func TestParseExecWithInterpolation(t *testing.T) {
	// P9: interpolation in exec
	input := "\nfn init {\n    exec(`hello ${name}`);\n}"
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
	exec, ok := fn.Body[0].(*ast.ExecStmt)
	if !ok {
		t.Fatalf("expected ExecStmt, got %T", fn.Body[0])
	}
	backtick, ok := exec.Value.(*ast.BacktickLit)
	if !ok {
		t.Fatalf("expected BacktickLit, got %T", exec.Value)
	}
	if len(backtick.Parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(backtick.Parts))
	}
	if backtick.Parts[0].Value != "hello " || backtick.Parts[0].IsVar {
		t.Fatalf("first part wrong")
	}
	if backtick.Parts[1].Value != "name" || !backtick.Parts[1].IsVar {
		t.Fatalf("second part wrong")
	}
}

func TestParseLogWithVarRef(t *testing.T) {
	// P10: log with bare $var ref
	input := "\nfn init {\n    log($myvar);\n}"
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
	log, ok := fn.Body[0].(*ast.LogStmt)
	if !ok {
		t.Fatalf("expected LogStmt, got %T", fn.Body[0])
	}
	varRef, ok := log.Value.(*ast.VarRef)
	if !ok {
		t.Fatalf("expected VarRef, got %T", log.Value)
	}
	if varRef.Name != "myvar" {
		t.Fatalf("expected var name myvar, got %s", varRef.Name)
	}
}
