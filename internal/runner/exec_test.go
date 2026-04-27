package runner

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/infraflakes/sro/internal/dsl/ast"
	"github.com/infraflakes/sro/internal/dsl/token"
)

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	ansi := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansi.ReplaceAllString(s, "")
}

func TestExecLog(t *testing.T) {
	cfg := testConfig(t)
	var buf bytes.Buffer
	ctx := newExecContext(cfg, cfg.Projects["testproj"], &buf)

	stmt := newLogStmt(newBacktickLit("hello world"))

	if err := ctx.execLog(stmt); err != nil {
		t.Fatalf("execLog error: %v", err)
	}
	// Output now includes "log  " prefix with ANSI coloring
	expected := "log  hello world\n"
	if stripANSI(buf.String()) != expected {
		t.Fatalf("log output: got %q, want %q", stripANSI(buf.String()), expected)
	}
}

func TestExecVarDecl(t *testing.T) {
	cfg := testConfig(t)
	ctx := newExecContext(cfg, cfg.Projects["testproj"], nil)

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
	cfg := testConfig(t)
	// Project dir relative to sanctuary
	cfg.Projects["testproj"].Dir = "testproj"
	projDir := filepath.Join(cfg.Sanctuary, "testproj")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ctx := newExecContext(cfg, cfg.Projects["testproj"], nil)
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
	cfg := testConfig(t)
	var buf bytes.Buffer
	ctx := newExecContext(cfg, cfg.Projects["testproj"], &buf)

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

	if err := ctx.execEnvBlock(block); err != nil {
		t.Fatalf("execEnvBlock error: %v", err)
	}
	if !strings.Contains(buf.String(), "FOO=bar") {
		t.Fatalf("expected output to contain 'FOO=bar', got %q", buf.String())
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
	cfg := testConfig(t)
	var buf bytes.Buffer
	ctx := newExecContext(cfg, cfg.Projects["testproj"], &buf)
	// Ensure project directory exists for exec
	baseDir := filepath.Join(cfg.Sanctuary, cfg.Projects["testproj"].Dir)
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}

	outerBody := []ast.FnStmt{
		newEnvBlock([]ast.EnvPair{{Key: "X", Value: newBacktickLit("1")}}, []ast.FnStmt{
			newLogStmt(newBacktickLit("outer-start")),
			// Echo X value directly
			newExecStmt(
				newBacktickLit("echo $X"),
			),
			newEnvBlock([]ast.EnvPair{{Key: "X", Value: newBacktickLit("2")}}, []ast.FnStmt{
				newLogStmt(newBacktickLit("inner")),
				// Echo X value directly
				newExecStmt(
					newBacktickLit("echo $X"),
				),
			}),
			// After inner block, X should be back to 1
			newExecStmt(
				newBacktickLit("echo $X"),
			),
			newLogStmt(newBacktickLit("outer-end")),
		}),
	}

	if err := ctx.execFnBody(outerBody); err != nil {
		t.Fatalf("execFnBody error: %v", err)
	}

	// Parse output: collect X values from lines containing just the value
	// Strip ANSI codes first
	cleanOut := stripANSI(buf.String())
	lines := strings.Split(strings.TrimSpace(cleanOut), "\n")
	var values []string
	for _, line := range lines {
		// Skip log and env primitive lines
		if strings.Contains(line, "log") || strings.Contains(line, "env") || strings.Contains(line, "exec") {
			continue
		}
		// The echo $X will output just the value (with indentation)
		if line != "" {
			values = append(values, strings.TrimSpace(line))
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
	cfg := testConfig(t)
	ctx := newExecContext(cfg, cfg.Projects["testproj"], nil)

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
	cfg := testConfig(t)
	var buf bytes.Buffer
	ctx := newExecContext(cfg, cfg.Projects["testproj"], &buf)

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

	if err := ctx.execEnvBlock(block); err != nil {
		t.Fatalf("execEnvBlock error: %v", err)
	}
	if !strings.Contains(buf.String(), "TEST_VAR=myvalue") {
		t.Fatalf("expected output to contain 'TEST_VAR=myvalue', got %q", buf.String())
	}
}

func TestExecActuallyRunsCommand(t *testing.T) {
	cfg := testConfig(t)
	// Ensure project directory exists for exec
	baseDir := filepath.Join(cfg.Sanctuary, cfg.Projects["testproj"].Dir)
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	ctx := newExecContext(cfg, cfg.Projects["testproj"], &buf)

	stmt := newExecStmt(newBacktickLit("echo hello"))
	if err := ctx.execExec(stmt); err != nil {
		t.Fatalf("execExec error: %v", err)
	}
	if !strings.Contains(buf.String(), "hello") {
		t.Fatalf("expected output to contain 'hello', got %q", buf.String())
	}
}

func TestExecWithNonZeroExitCode(t *testing.T) {
	cfg := testConfig(t)
	// Ensure project directory exists for exec
	baseDir := filepath.Join(cfg.Sanctuary, cfg.Projects["testproj"].Dir)
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ctx := newExecContext(cfg, cfg.Projects["testproj"], nil)

	stmt := newExecStmt(newBacktickLit("false"))
	err := ctx.execExec(stmt)
	if err == nil {
		t.Fatal("expected error for non-zero exit code")
	}
	if !strings.Contains(err.Error(), "exit code") {
		t.Fatalf("expected error to contain exit code, got: %v", err)
	}
}

func TestExecWithInterpolation(t *testing.T) {
	cfg := testConfig(t)
	// Ensure project directory exists for exec
	baseDir := filepath.Join(cfg.Sanctuary, cfg.Projects["testproj"].Dir)
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	ctx := newExecContext(cfg, cfg.Projects["testproj"], &buf)

	// Declare a var
	if err := ctx.execVarDecl(newVarDecl("name", "string", newBacktickLit("world"))); err != nil {
		t.Fatal(err)
	}

	// Create backtick with interpolation
	backtick := &ast.BacktickLit{
		Token: newTestToken(token.BACKTICK),
		Parts: []ast.TemplatePart{
			{IsVar: false, Value: "echo hello "},
			{IsVar: true, Value: "name"},
		},
	}
	stmt := newExecStmt(backtick)
	if err := ctx.execExec(stmt); err != nil {
		t.Fatalf("execExec error: %v", err)
	}
	if !strings.Contains(buf.String(), "hello world") {
		t.Fatalf("expected output to contain 'hello world', got %q", buf.String())
	}
}

func TestLogWithInterpolation(t *testing.T) {
	cfg := testConfig(t)
	var buf bytes.Buffer
	ctx := newExecContext(cfg, cfg.Projects["testproj"], &buf)

	// Declare a var
	if err := ctx.execVarDecl(newVarDecl("name", "string", newBacktickLit("world"))); err != nil {
		t.Fatal(err)
	}

	// Create backtick with interpolation
	backtick := &ast.BacktickLit{
		Token: newTestToken(token.BACKTICK),
		Parts: []ast.TemplatePart{
			{IsVar: false, Value: "hello "},
			{IsVar: true, Value: "name"},
		},
	}
	stmt := newLogStmt(backtick)
	if err := ctx.execLog(stmt); err != nil {
		t.Fatalf("execLog error: %v", err)
	}
	if !strings.Contains(buf.String(), "hello world") {
		t.Fatalf("expected output to contain 'hello world', got %q", buf.String())
	}
}

func TestLogWithVarRef(t *testing.T) {
	cfg := testConfig(t)
	var buf bytes.Buffer
	ctx := newExecContext(cfg, cfg.Projects["testproj"], &buf)

	// Declare a var
	if err := ctx.execVarDecl(newVarDecl("myvar", "string", newBacktickLit("myvalue"))); err != nil {
		t.Fatal(err)
	}

	stmt := newLogStmt(newVarRef("myvar"))
	if err := ctx.execLog(stmt); err != nil {
		t.Fatalf("execLog error: %v", err)
	}
	if !strings.Contains(buf.String(), "myvalue") {
		t.Fatalf("expected output to contain 'myvalue', got %q", buf.String())
	}
}

func TestShellVarInFnBody(t *testing.T) {
	cfg := testConfig(t)
	// Ensure project directory exists for exec
	baseDir := filepath.Join(cfg.Sanctuary, cfg.Projects["testproj"].Dir)
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ctx := newExecContext(cfg, cfg.Projects["testproj"], nil)

	stmt := newVarDecl("x", "shell", newBacktickLit("echo hello"))
	if err := ctx.execVarDecl(stmt); err != nil {
		t.Fatalf("execVarDecl error: %v", err)
	}
	if ctx.vars["x"] != "hello" {
		t.Fatalf("shell var wrong: got %s, want hello", ctx.vars["x"])
	}
}

func TestCdFollowedByExec(t *testing.T) {
	cfg := testConfig(t)
	cfg.Projects["testproj"].Dir = "testproj"
	projDir := filepath.Join(cfg.Sanctuary, "testproj")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Create a file in subdirectory
	subDir := filepath.Join(projDir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	testFile := filepath.Join(subDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := newExecContext(cfg, cfg.Projects["testproj"], nil)
	// cd to subdirectory
	if err := ctx.execCd(newCdStmt("sub")); err != nil {
		t.Fatal(err)
	}
	// exec should run in the cd'd directory
	stmt := newExecStmt(newBacktickLit("ls test.txt"))
	err := ctx.execExec(stmt)
	if err != nil {
		t.Fatalf("execExec error: %v", err)
	}
}

func TestEnvBlockWithMultiplePairs(t *testing.T) {
	cfg := testConfig(t)
	var buf bytes.Buffer
	ctx := newExecContext(cfg, cfg.Projects["testproj"], &buf)

	// Ensure project directory exists for exec
	baseDir := filepath.Join(cfg.Sanctuary, cfg.Projects["testproj"].Dir)
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}

	envPairs := []ast.EnvPair{
		{Key: "FOO", Value: newBacktickLit("bar")},
		{Key: "BAZ", Value: newBacktickLit("qux")},
	}
	innerBody := []ast.FnStmt{
		newExecStmt(newBacktickLit("env")),
	}
	block := newEnvBlock(envPairs, innerBody)

	if err := ctx.execEnvBlock(block); err != nil {
		t.Fatalf("execEnvBlock error: %v", err)
	}
	if !strings.Contains(buf.String(), "FOO=bar") {
		t.Fatalf("expected output to contain 'FOO=bar', got %q", buf.String())
	}
	if !strings.Contains(buf.String(), "BAZ=qux") {
		t.Fatalf("expected output to contain 'BAZ=qux', got %q", buf.String())
	}
}

func TestEmptyFnBodyExecution(t *testing.T) {
	// R9: empty fn body execution
	cfg := testConfig(t)
	ctx := newExecContext(cfg, cfg.Projects["testproj"], nil)

	body := []ast.FnStmt{}
	err := ctx.execFnBody(body)
	if err != nil {
		t.Fatalf("execFnBody with empty body should succeed: %v", err)
	}
}
