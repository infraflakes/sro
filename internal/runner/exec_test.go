package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/infraflakes/sro/internal/dsl/ast"
)

func TestExecLog(t *testing.T) {
	cfg := testConfig()
	ctx := newExecContext(cfg, cfg.Projects["testproj"])

	stmt := newLogStmt(newBacktickLit("hello world"))

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
	stmt1 := newVarDecl("x", "string", newBacktickLit("value1"))
	if err := ctx.execVarDecl(stmt1); err != nil {
		t.Fatalf("varDecl error: %v", err)
	}
	if ctx.vars["x"] != "value1" {
		t.Fatalf("x = %s", ctx.vars["x"])
	}

	// Var reference
	stmt2 := newVarDecl("y", "string", newVarRef("x"))
	if err := ctx.execVarDecl(stmt2); err != nil {
		t.Fatalf("varDecl from ref error: %v", err)
	}
	if ctx.vars["y"] != "value1" {
		t.Fatalf("y = %s", ctx.vars["y"])
	}

	// Undefined var ref
	stmt3 := newVarDecl("z", "string", newVarRef("nonexistent"))
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
		newExecStmt(
			newBacktickLit("env"),
		),
	}
	block := newEnvBlock([]ast.EnvPair{{Key: "FOO", Value: newBacktickLit("bar")}}, innerBody)

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
		newEnvBlock([]ast.EnvPair{{Key: "X", Value: newBacktickLit("1")}}, []ast.FnStmt{
			newLogStmt(newBacktickLit("outer-start")),
			// Print env to capture X
			newExecStmt(
				newBacktickLit("env"),
			),
			newEnvBlock([]ast.EnvPair{{Key: "X", Value: newBacktickLit("2")}}, []ast.FnStmt{
				newLogStmt(newBacktickLit("inner")),
				// Print env to capture X
				newExecStmt(
					newBacktickLit("env"),
				),
			}),
			// After inner block, X should be back to 1
			newExecStmt(
				newBacktickLit("env"),
			),
			newLogStmt(newBacktickLit("outer-end")),
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
			newVarDecl("innerVar", "string", newBacktickLit("innerValue")),
		}),
		newVarDecl("outerVar", "string", newVarRef("innerVar")), // This should fail
	}

	err := ctx.execFnBody(body)
	if err == nil {
		t.Fatal("expected error for undefined var innerVar outside env block")
	}
	if !strings.Contains(err.Error(), "undefined variable $innerVar") {
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
	if err := ctx.execVarDecl(newVarDecl("myvar", "string", newBacktickLit("myvalue"))); err != nil {
		t.Fatal(err)
	}

	// Test env value with var ref
	innerBody := []ast.FnStmt{
		newExecStmt(
			newBacktickLit("env"),
		),
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
