package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadBasic(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := "shell = `bash`;\nsanctuary = `/tmp/dev`;\nvar string a = `hello`;\npr test { url = `http://example.com`; dir = `test`; }\nfn greet { log(`hi`); }\n"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(file)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.Shell != "bash" {
		t.Fatalf("shell wrong: got %s", cfg.Shell)
	}
	if cfg.Sanctuary != "/tmp/dev" {
		t.Fatalf("sanctuary wrong: got %s", cfg.Sanctuary)
	}
	if len(cfg.Vars) != 1 || cfg.Vars["a"] != "hello" {
		t.Fatalf("vars wrong: %v", cfg.Vars)
	}
	if len(cfg.Projects) != 1 || cfg.Projects["test"].URL != "http://example.com" {
		t.Fatalf("projects wrong: %v", cfg.Projects)
	}
	if len(cfg.Functions) != 1 || cfg.Functions["greet"] == nil {
		t.Fatalf("functions wrong: %v", cfg.Functions)
	}
}

func TestImportResolution(t *testing.T) {
	dir := t.TempDir()
	mainFile := filepath.Join(dir, "main.sro")
	otherFile := filepath.Join(dir, "other.sro")

	otherContent := "var string extra = `from-other`;"
	mainContent := "shell = `bash`;\nsanctuary = `/tmp`;\nimport [ ./other.sro ];\nvar string x = $extra;\n"

	if err := os.WriteFile(otherFile, []byte(otherContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mainFile, []byte(mainContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(mainFile)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.Vars["x"] != "from-other" {
		t.Fatalf("var x should resolve to 'from-other', got %s", cfg.Vars["x"])
	}
}

func TestCircularImport(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.sro")
	b := filepath.Join(dir, "b.sro")

	contentA := "shell = `bash`; import [ ./b.sro ]; sanctuary = `/tmp`;"
	contentB := "shell = `bash`; import [ ./a.sro ]; sanctuary = `/tmp`;"

	if err := os.WriteFile(a, []byte(contentA), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte(contentB), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(a)
	if err == nil || !strings.Contains(err.Error(), "circular import") {
		t.Fatalf("expected circular import error, got: %v", err)
	}
}

func TestDuplicates(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := "shell = `bash`;\nsanctuary = `/tmp`;\nsanctuary = `/other`;\n"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(file)
	if err == nil || !strings.Contains(err.Error(), "duplicate sanctuary") {
		t.Fatalf("expected duplicate sanctuary error, got: %v", err)
	}

	content2 := "shell = `bash`;\nsanctuary = `/tmp`;\nvar string x = `a`;\nvar string x = `b`;\n"
	if err := os.WriteFile(file, []byte(content2), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = Load(file)
	if err == nil || !strings.Contains(err.Error(), "duplicate variable: x") {
		t.Fatalf("expected duplicate variable error, got: %v", err)
	}

	content3 := "shell = `bash`;\nsanctuary = `/tmp`;\npr p1 { url = `u`; dir = `d1`; }\npr p1 { url = `u2`; dir = `d2`; }\n"
	if err := os.WriteFile(file, []byte(content3), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = Load(file)
	if err == nil || !strings.Contains(err.Error(), "duplicate project: p1") {
		t.Fatalf("expected duplicate project error, got: %v", err)
	}
}

func TestVariableResolution(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := "shell = `bash`;\nsanctuary = `/tmp`;\nvar string a = `x`;\nvar string b = $a;\nvar string c = $b;\n"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(file)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.Vars["a"] != "x" || cfg.Vars["b"] != "x" || cfg.Vars["c"] != "x" {
		t.Fatalf("variable chain wrong: %v", cfg.Vars)
	}
}

func TestUndefinedVariable(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := "shell = `bash`;\nsanctuary = `/tmp`;\nvar string x = $missing;\n"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(file)
	if err == nil || !strings.Contains(err.Error(), "undefined variable: $missing") {
		t.Fatalf("expected undefined variable error, got: %v", err)
	}
}

func TestShellExecResolution(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := "shell = `bash`;\nsanctuary = `/tmp`;\nvar shell test_var = `echo hello`;"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(file)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.Vars["test_var"] != "hello" {
		t.Fatalf("shell exec resolution wrong: got %s, want hello", cfg.Vars["test_var"])
	}
}

func TestMissingShellError(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := "sanctuary = `/tmp`;\nvar shell test_var = `echo hello`;"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(file)
	if err == nil || !strings.Contains(err.Error(), "shell must be declared") {
		t.Fatalf("expected shell must be declared error, got: %v", err)
	}
}

func TestSanctuaryWithVarRef(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := "shell = `bash`;\nvar shell workdir = `echo /tmp/test`;\nsanctuary = $workdir;\n"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(file)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.Sanctuary != "/tmp/test" {
		t.Fatalf("sanctuary with var ref wrong: got %s, want /tmp/test", cfg.Sanctuary)
	}
}

func TestSanctuaryEnvExpansion(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	// Use shell exec to expand HOME instead of os.ExpandEnv
	content := "shell = `bash`;\nvar shell sanctuary_path = `echo $HOME/dev`;\nsanctuary = $sanctuary_path;"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(file)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	expected := os.Getenv("HOME") + "/dev"
	if cfg.Sanctuary != expected {
		t.Fatalf("sanctuary with shell exec wrong: got %s, want %s", cfg.Sanctuary, expected)
	}
}

func TestDuplicateDir(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := "shell = `bash`;\nsanctuary = `/tmp`;\npr a { url = `ua`; dir = `shared`; }\npr b { url = `ub`; dir = `shared`; }\n"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(file)
	if err == nil || !strings.Contains(err.Error(), "same dir") {
		t.Fatalf("expected duplicate dir error, got: %v", err)
	}
}

func TestMultiFileParseOrder(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.sro")
	b := filepath.Join(dir, "b.sro")

	contentA := "sanctuary = `/tmp`; var string a = `from-a`;"
	contentB := "shell = `bash`; import [ ./a.sro ]; var string b = $a;"

	if err := os.WriteFile(a, []byte(contentA), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte(contentB), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(b)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.Vars["b"] != "from-a" {
		t.Fatalf("variable across files not resolved: %v", cfg.Vars)
	}
}

func TestDuplicateFnSeqParNames(t *testing.T) {
	// C1: duplicate fn/seq/par names
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := "shell = `bash`;\nsanctuary = `/tmp`;\nfn dup { log(`a`); }\nfn dup { log(`b`); }"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(file)
	if err == nil || !strings.Contains(err.Error(), "duplicate function: dup") {
		t.Fatalf("expected duplicate function error, got: %v", err)
	}

	content2 := "shell = `bash`;\nsanctuary = `/tmp`;\npr test { url = `http://example.com`; dir = `test`; }\nseq dup { check(test); }\nseq dup { build(test); }"
	if err := os.WriteFile(file, []byte(content2), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = Load(file)
	if err == nil || !strings.Contains(err.Error(), "duplicate seq: dup") {
		t.Fatalf("expected duplicate seq error, got: %v", err)
	}

	content3 := "shell = `bash`;\nsanctuary = `/tmp`;\npr test { url = `http://example.com`; dir = `test`; }\npar dup { check(test); }\npar dup { build(test); }"
	if err := os.WriteFile(file, []byte(content3), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = Load(file)
	if err == nil || !strings.Contains(err.Error(), "duplicate par: dup") {
		t.Fatalf("expected duplicate par error, got: %v", err)
	}
}

func TestInterpolationInBacktick(t *testing.T) {
	// C4: ${var} interpolation in backtick during merge
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := "shell = `bash`;\nsanctuary = `/tmp`;\nvar string name = `world`;\nvar string greeting = `hello ${name}`;"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(file)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.Vars["greeting"] != "hello world" {
		t.Fatalf("interpolation wrong: got %s, want hello world", cfg.Vars["greeting"])
	}
}

func TestShellVarWithInterpolation(t *testing.T) {
	// C5: shell var with ${var} interpolation
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := "shell = `bash`;\nsanctuary = `/tmp`;\nvar string name = `world`;\nvar shell greeting = `echo hello ${name}`;"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(file)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.Vars["greeting"] != "hello world" {
		t.Fatalf("shell var interpolation wrong: got %s, want hello world", cfg.Vars["greeting"])
	}
}

func TestProjectFieldWithVarRef(t *testing.T) {
	// C6: project field with $var reference
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := "shell = `bash`;\nsanctuary = `/tmp`;\nvar string myurl = `http://example.com`;\npr x { url = $myurl; dir = `d`; }"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(file)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.Projects["x"].URL != "http://example.com" {
		t.Fatalf("project field with var ref wrong: got %s", cfg.Projects["x"].URL)
	}
}
