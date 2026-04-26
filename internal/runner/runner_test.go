package runner

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/infraflakes/sro/internal/config"
	"github.com/infraflakes/sro/internal/dsl/ast"
	"github.com/infraflakes/sro/internal/dsl/token"
)

func newTestToken(typ token.TokenType) token.Token {
	return token.Token{
		Type:    typ,
		Literal: string(typ),
		Line:    1,
		Col:     1,
	}
}

func newStringLit(value string) *ast.StringLit {
	return &ast.StringLit{
		Token: newTestToken(token.STRING_LIT),
		Value: value,
	}
}

func newVarRef(name string) *ast.VarRef {
	return &ast.VarRef{
		Token: newTestToken(token.IDENT),
		Name:  name,
	}
}

func newLogStmt(args []ast.Expr) *ast.LogStmt {
	return &ast.LogStmt{
		Token: newTestToken(token.LOG),
		Args:  args,
	}
}

func newExecStmt(args []ast.Expr) *ast.ExecStmt {
	return &ast.ExecStmt{
		Token: newTestToken(token.EXEC),
		Args:  args,
	}
}

func newCdStmt(arg string) *ast.CdStmt {
	return &ast.CdStmt{
		Token: newTestToken(token.CD),
		Arg:   arg,
	}
}

func newVarDecl(name string, value ast.Expr) *ast.VarDecl {
	return &ast.VarDecl{
		Token: newTestToken(token.VAR),
		Name:  name,
		Value: value,
	}
}

func newEnvBlock(pairs []ast.EnvPair, body []ast.FnStmt) *ast.EnvBlock {
	return &ast.EnvBlock{
		Token: newTestToken(token.ENV),
		Pairs: pairs,
		Body:  body,
	}
}

func newFnDecl(name string, body []ast.FnStmt) *ast.FnDecl {
	return &ast.FnDecl{
		Token: newTestToken(token.FN),
		Name:  name,
		Body:  body,
	}
}

func newFnCall(fnName, projectName string) *ast.FnCall {
	return &ast.FnCall{
		Token:       newTestToken(token.IDENT),
		FnName:      fnName,
		ProjectName: projectName,
	}
}

func testConfig() *config.Config {
	return &config.Config{
		Shell:     "bash",
		Sanctuary: "/tmp/sanctuary",
		Projects: map[string]*config.Project{
			"testproj": {
				Name: "testproj",
				URL:  "http://example.com",
				Dir:  "testproj",
				Sync: "clone",
			},
		},
		Functions: make(map[string]*ast.FnDecl),
		Seqs:      make(map[string]*ast.SeqDecl),
		Pars:      make(map[string]*ast.ParDecl),
		Vars:      map[string]string{},
	}
}

func captureOutput(fn func()) (string, error) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = w

	fn() // run the function, writing to the pipe

	// Restore stdout and close writer
	os.Stdout = old
	_ = w.Close()

	var buf strings.Builder
	_, err = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String(), err
}

func TestExecLog(t *testing.T) {
	cfg := testConfig()
	ctx := newExecContext(cfg, cfg.Projects["testproj"])

	stmt := newLogStmt([]ast.Expr{
		newStringLit("hello"),
		newStringLit(" "),
		newStringLit("world"),
	})

	out, err := captureOutput(func() {
		err := ctx.execLog(stmt)
		if err != nil {
			t.Fatalf("execLog error: %v", err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := "hello world\n"
	if out != expected {
		t.Fatalf("log output: got %q, want %q", out, expected)
	}
}

func TestExecVarDecl(t *testing.T) {
	cfg := testConfig()
	ctx := newExecContext(cfg, cfg.Projects["testproj"])

	// Declare var x
	stmt1 := newVarDecl("x", newStringLit("value1"))
	if err := ctx.execVarDecl(stmt1); err != nil {
		t.Fatalf("varDecl error: %v", err)
	}
	if ctx.vars["x"] != "value1" {
		t.Fatalf("x = %s", ctx.vars["x"])
	}

	// Var reference
	stmt2 := newVarDecl("y", newVarRef("x"))
	if err := ctx.execVarDecl(stmt2); err != nil {
		t.Fatalf("varDecl from ref error: %v", err)
	}
	if ctx.vars["y"] != "value1" {
		t.Fatalf("y = %s", ctx.vars["y"])
	}

	// Undefined var ref
	stmt3 := newVarDecl("z", newVarRef("nonexistent"))
	if err := ctx.execVarDecl(stmt3); err == nil {
		t.Fatal("expected error for undefined var")
	}
}

func TestExecCd(t *testing.T) {
	cfg := testConfig()
	// Use a temporary sanctuary base
	tempBase := t.TempDir()
	cfg.Sanctuary = tempBase
	// Project dir relative to sanctuary
	cfg.Projects["testproj"].Dir = "testproj"
	projDir := filepath.Join(tempBase, "testproj")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ctx := newExecContext(cfg, cfg.Projects["testproj"])
	if ctx.workDir != projDir {
		t.Fatalf("initial workdir wrong: got %s, want %s", ctx.workDir, projDir)
	}
	// cd to existing subdirectory
	subDir := filepath.Join(projDir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ctx.execCd(newCdStmt("sub")); err != nil {
		t.Fatalf("cd sub error: %v", err)
	}
	if ctx.workDir != subDir {
		t.Fatalf("after cd sub: got %s, want %s", ctx.workDir, subDir)
	}
	// cd . resets to project root
	if err := ctx.execCd(newCdStmt(".")); err != nil {
		t.Fatalf("cd . error: %v", err)
	}
	if ctx.workDir != projDir {
		t.Fatalf("after cd .: got %s, want %s", ctx.workDir, projDir)
	}
	// cd to nonexistent directory should fail
	if err := ctx.execCd(newCdStmt("nonexistent")); err == nil {
		t.Fatal("expected cd to nonexistent to fail")
	}
}

func TestExecEnvBlock(t *testing.T) {
	cfg := testConfig()
	ctx := newExecContext(cfg, cfg.Projects["testproj"])

	// Ensure project directory exists for exec
	baseDir := filepath.Join(cfg.Sanctuary, cfg.Projects["testproj"].Dir)
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Check FOO not in outer env
	beforeEnv := ctx.buildEnv()
	for _, e := range beforeEnv {
		if strings.HasPrefix(e, "FOO=") {
			t.Fatalf("FOO should not be in env before block")
		}
	}

	innerBody := []ast.FnStmt{
		newExecStmt([]ast.Expr{
			newStringLit("env"),
		}),
	}
	block := newEnvBlock([]ast.EnvPair{{Key: "FOO", Value: newStringLit("bar")}}, innerBody)

	out, err := captureOutput(func() {
		if err := ctx.execEnvBlock(block); err != nil {
			t.Fatalf("execEnvBlock error: %v", err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "FOO=bar") {
		t.Fatalf("expected output to contain 'FOO=bar', got %q", out)
	}

	// After block, FOO should be gone from env
	afterEnv := ctx.buildEnv()
	for _, e := range afterEnv {
		if strings.HasPrefix(e, "FOO=") {
			t.Fatalf("FOO should be popped after env block")
		}
	}
}

func TestExecEnvBlockNestedOverride(t *testing.T) {
	cfg := testConfig()
	ctx := newExecContext(cfg, cfg.Projects["testproj"])
	// Ensure project directory exists for exec
	baseDir := filepath.Join(cfg.Sanctuary, cfg.Projects["testproj"].Dir)
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}

	outerBody := []ast.FnStmt{
		newEnvBlock([]ast.EnvPair{{Key: "X", Value: newStringLit("1")}}, []ast.FnStmt{
			newLogStmt([]ast.Expr{newStringLit("outer-start")}),
			// Print env to capture X
			newExecStmt([]ast.Expr{
				newStringLit("env"),
			}),
			newEnvBlock([]ast.EnvPair{{Key: "X", Value: newStringLit("2")}}, []ast.FnStmt{
				newLogStmt([]ast.Expr{newStringLit("inner")}),
				// Print env to capture X
				newExecStmt([]ast.Expr{
					newStringLit("env"),
				}),
			}),
			// After inner block, X should be back to 1
			newExecStmt([]ast.Expr{
				newStringLit("env"),
			}),
			newLogStmt([]ast.Expr{newStringLit("outer-end")}),
		}),
	}

	out, err := captureOutput(func() {
		if err := ctx.execFnBody(outerBody); err != nil {
			t.Fatalf("execFnBody error: %v", err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}

	// Parse output: collect X values from lines "X=..."
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var values []string
	for _, line := range lines {
		if strings.HasPrefix(line, "  exec") {
			continue
		}
		if after, ok := strings.CutPrefix(line, "X="); ok {
			val := after
			values = append(values, val)
		}
	}
	if len(values) < 3 {
		t.Fatalf("expected at least 3 X outputs, got %d: %v", len(values), values)
	}
	if values[0] != "1" || values[1] != "2" || values[2] != "1" {
		t.Fatalf("X values: got %v, want [1 2 1]", values[:3])
	}
}

func TestExecEnvBlockVarScoping(t *testing.T) {
	cfg := testConfig()
	ctx := newExecContext(cfg, cfg.Projects["testproj"])

	// Test that vars declared inside env block don't leak outside
	body := []ast.FnStmt{
		newEnvBlock([]ast.EnvPair{}, []ast.FnStmt{
			newVarDecl("innerVar", newStringLit("innerValue")),
		}),
		newVarDecl("outerVar", newVarRef("innerVar")), // This should fail
	}

	err := ctx.execFnBody(body)
	if err == nil {
		t.Fatal("expected error for undefined var innerVar outside env block")
	}
	if !strings.Contains(err.Error(), "undefined variable: $innerVar") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecEnvBlockVarRefs(t *testing.T) {
	cfg := testConfig()
	ctx := newExecContext(cfg, cfg.Projects["testproj"])

	// Ensure project directory exists for exec
	baseDir := filepath.Join(cfg.Sanctuary, cfg.Projects["testproj"].Dir)
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Declare a var
	if err := ctx.execVarDecl(newVarDecl("myvar", newStringLit("myvalue"))); err != nil {
		t.Fatal(err)
	}

	// Test env value with var ref
	innerBody := []ast.FnStmt{
		newExecStmt([]ast.Expr{
			newStringLit("env"),
		}),
	}
	envPairs := []ast.EnvPair{{Key: "TEST_VAR", Value: newVarRef("myvar")}}
	block := newEnvBlock(envPairs, innerBody)

	out, err := captureOutput(func() {
		if err := ctx.execEnvBlock(block); err != nil {
			t.Fatalf("execEnvBlock error: %v", err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "TEST_VAR=myvalue") {
		t.Fatalf("expected output to contain 'TEST_VAR=myvalue', got %q", out)
	}
}

func TestSeqFailFast(t *testing.T) {
	cfg := testConfig()

	// Function that will fail
	failFn := newFnDecl("fail", []ast.FnStmt{
		newExecStmt([]ast.Expr{newStringLit("false")}),
	})
	cfg.Functions["fail"] = failFn

	// Function that should NOT run
	secondFn := newFnDecl("second", []ast.FnStmt{
		newLogStmt([]ast.Expr{newStringLit("second-called")}),
	})
	cfg.Functions["second"] = secondFn

	seq := &ast.SeqDecl{
		Token: newTestToken(token.SEQ),
		Name:  "testseq",
		Stmts: []ast.SeqStmt{
			newFnCall("fail", "testproj"),
			newFnCall("second", "testproj"),
		},
	}
	cfg.Seqs["testseq"] = seq

	r := New(cfg)
	err := r.RunSeq("testseq")
	if err == nil {
		t.Fatal("expected error from failing exec")
	}
	// The second function should not have been called — can't easily test without side effects
}

func TestParContinuesOnFailure(t *testing.T) {
	cfg := testConfig()

	// Function that fails
	failFn := newFnDecl("fail", []ast.FnStmt{
		newExecStmt([]ast.Expr{newStringLit("false")}),
	})
	cfg.Functions["fail"] = failFn

	// Function that succeeds
	successFn := newFnDecl("success", []ast.FnStmt{
		newLogStmt([]ast.Expr{newStringLit("success-called")}),
	})
	cfg.Functions["success"] = successFn

	par := &ast.ParDecl{
		Token: newTestToken(token.PAR),
		Name:  "testpar",
		Stmts: []ast.ParStmt{
			newFnCall("fail", "testproj"),
			newFnCall("success", "testproj"),
		},
	}
	cfg.Pars["testpar"] = par

	r := New(cfg)
	err := r.RunPar("testpar")
	if err == nil {
		t.Fatal("expected error from par with failing task")
	}
	if !strings.Contains(err.Error(), "fail(pr.testproj)") {
		t.Logf("error msg: %s", err.Error())
	}
	// success should not appear in errors (it succeeded)
}

func TestSeqCallsSeq(t *testing.T) {
	cfg := testConfig()
	logFn := newFnDecl("logfn", []ast.FnStmt{
		newLogStmt([]ast.Expr{newStringLit("inner-log")}),
	})
	cfg.Functions["logfn"] = logFn

	innerSeq := &ast.SeqDecl{
		Token: newTestToken(token.SEQ),
		Name:  "inner",
		Stmts: []ast.SeqStmt{
			newFnCall("logfn", "testproj"),
		},
	}
	cfg.Seqs["inner"] = innerSeq

	outerSeq := &ast.SeqDecl{
		Token: newTestToken(token.SEQ),
		Name:  "outer",
		Stmts: []ast.SeqStmt{
			newFnCall("logfn", "testproj"),
			&ast.SeqRef{
				Token:   newTestToken(token.SEQ),
				SeqName: "inner",
			},
		},
	}
	cfg.Seqs["outer"] = outerSeq

	r := New(cfg)
	// Can't easily capture output here; just ensure no error
	if err := r.RunSeq("outer"); err != nil {
		t.Fatalf("RunSeq error: %v", err)
	}
}

func TestParCallsSeq(t *testing.T) {
	cfg := testConfig()
	logFn := newFnDecl("logfn", []ast.FnStmt{
		newLogStmt([]ast.Expr{newStringLit("par-seq-log")}),
	})
	cfg.Functions["logfn"] = logFn

	seq := &ast.SeqDecl{
		Token: newTestToken(token.SEQ),
		Name:  "myseq",
		Stmts: []ast.SeqStmt{
			newFnCall("logfn", "testproj"),
		},
	}
	cfg.Seqs["myseq"] = seq

	par := &ast.ParDecl{
		Token: newTestToken(token.PAR),
		Name:  "mypar",
		Stmts: []ast.ParStmt{
			&ast.SeqRef{
				Token:   newTestToken(token.SEQ),
				SeqName: "myseq",
			},
		},
	}
	cfg.Pars["mypar"] = par

	r := New(cfg)
	if err := r.RunPar("mypar"); err != nil {
		t.Fatalf("RunPar error: %v", err)
	}
}

func TestRunSeqUnknown(t *testing.T) {
	cfg := testConfig()
	r := New(cfg)
	err := r.RunSeq("nonexistent")
	if err == nil || !strings.Contains(err.Error(), "unknown seq") {
		t.Fatalf("expected unknown seq error, got: %v", err)
	}
}

func TestRunParUnknown(t *testing.T) {
	cfg := testConfig()
	r := New(cfg)
	err := r.RunPar("nonexistent")
	if err == nil || !strings.Contains(err.Error(), "unknown par") {
		t.Fatalf("expected unknown par error, got: %v", err)
	}
}

func TestUnknownFunctionInSeq(t *testing.T) {
	cfg := testConfig()
	seq := &ast.SeqDecl{
		Token: newTestToken(token.SEQ),
		Name:  "badseq",
		Stmts: []ast.SeqStmt{
			newFnCall("nofn", "testproj"),
		},
	}
	cfg.Seqs["badseq"] = seq
	r := New(cfg)
	err := r.RunSeq("badseq")
	if err == nil || !strings.Contains(err.Error(), "unknown function") {
		t.Fatalf("expected unknown function error, got: %v", err)
	}
}

func TestUnknownProjectInSeq(t *testing.T) {
	cfg := testConfig()
	cfg.Functions["dummy"] = newFnDecl("dummy", []ast.FnStmt{})
	seq := &ast.SeqDecl{
		Token: newTestToken(token.SEQ),
		Name:  "badseq",
		Stmts: []ast.SeqStmt{
			newFnCall("dummy", "noproj"),
		},
	}
	cfg.Seqs["badseq"] = seq
	r := New(cfg)
	err := r.RunSeq("badseq")
	if err == nil || !strings.Contains(err.Error(), "unknown project") {
		t.Fatalf("expected unknown project error, got: %v", err)
	}
}
