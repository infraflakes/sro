package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadBasic(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := `sanctuary = "$HOME/dev";
var a := "hello";
pr test { url = "http://example.com"; dir = "test"; }
fn greet { log("hi"); }
`
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(file)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.Sanctuary != os.Getenv("HOME")+"/dev" {
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

	otherContent := `var extra := "from-other";`
	mainContent := `sanctuary = "/tmp";
import [ ./other.sro ];
var x := $extra;
`

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

	contentA := `import [ ./b.sro ]; sanctuary = "/tmp";`
	contentB := `import [ ./a.sro ]; sanctuary = "/tmp";`

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
	content := `sanctuary = "/tmp";
sanctuary = "/other";
`
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(file)
	if err == nil || !strings.Contains(err.Error(), "duplicate sanctuary") {
		t.Fatalf("expected duplicate sanctuary error, got: %v", err)
	}

	content2 := `sanctuary = "/tmp";
var x := "a";
var x := "b";
`
	if err := os.WriteFile(file, []byte(content2), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = Load(file)
	if err == nil || !strings.Contains(err.Error(), "duplicate variable: x") {
		t.Fatalf("expected duplicate variable error, got: %v", err)
	}

	content3 := `sanctuary = "/tmp";
pr p1 { url = "u"; dir = "d1"; }
pr p1 { url = "u2"; dir = "d2"; }
`
	if err := os.WriteFile(file, []byte(content3), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = Load(file)
	if err == nil || !strings.Contains(err.Error(), "duplicate project: p1") {
		t.Fatalf("expected duplicate project error, got: %v", err)
	}
}

func TestDuplicateVarInFnBody(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := `sanctuary = "/tmp";
pr test { url = "http://example.com"; dir = "test"; }
fn bad {
    var x := "a";
    var x := "b";
}
`
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(file)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	err = validateFull(cfg)
	if err == nil || !strings.Contains(err.Error(), "duplicate variable") {
		t.Fatalf("expected duplicate variable error, got: %v", err)
	}
}

func TestVariableResolution(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := `sanctuary = "/tmp";
var a := "x";
var b := $a;
var c := $b;
`
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
	content := `sanctuary = "/tmp";
var x := $missing;
`
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(file)
	if err == nil || !strings.Contains(err.Error(), "undefined variable: $missing") {
		t.Fatalf("expected undefined variable error, got: %v", err)
	}
}

func TestSanctuaryEnvExpansion(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := `sanctuary = "$HOME/dev";`
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(file)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	expected := os.Getenv("HOME") + "/dev"
	if cfg.Sanctuary != expected {
		t.Fatalf("sanctuary expansion wrong: got %s, want %s", cfg.Sanctuary, expected)
	}
}

func TestMissingRequiredFields(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")

	contentNoURL := `sanctuary = "/tmp";
pr p { dir = "d"; }
`
	if err := os.WriteFile(file, []byte(contentNoURL), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(file)
	if err == nil || !strings.Contains(err.Error(), `url is required`) {
		t.Fatalf("expected missing url error, got: %v", err)
	}

	contentNoDir := `sanctuary = "/tmp";
pr p { url = "u"; }
`
	if err := os.WriteFile(file, []byte(contentNoDir), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = Load(file)
	if err == nil || !strings.Contains(err.Error(), `dir is required`) {
		t.Fatalf("expected missing dir error, got: %v", err)
	}
}

func TestInvalidSyncValue(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := `sanctuary = "/tmp";
pr p { url = "u"; dir = "d"; sync = "invalid"; }
`
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(file)
	if err == nil || !strings.Contains(err.Error(), `sync must be`) {
		t.Fatalf("expected invalid sync error, got: %v", err)
	}
}

func TestSeqParReferenceValidation(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := `sanctuary = "/tmp";
fn real { log("hi"); }
seq s { unknown(pr.p); }
par p { fake(pr.q); }
`
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(file)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// Now validateFull should catch the errors
	err = validateFull(cfg)
	if err == nil || !strings.Contains(err.Error(), "unknown function") || !strings.Contains(err.Error(), "unknown project") {
		t.Fatalf("expected unknown fn/project errors, got: %v", err)
	}
}

func TestDuplicateDir(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := `sanctuary = "/tmp";
pr a { url = "ua"; dir = "shared"; }
pr b { url = "ub"; dir = "shared"; }
`
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

	contentA := `sanctuary = "/tmp"; var a := "from-a";`
	contentB := `import [ ./a.sro ]; var b := $a;`

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

func TestResolveUse(t *testing.T) {
	dir := t.TempDir()

	// Create project directory inside sanctuary
	projDir := filepath.Join(dir, "test")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a use file inside the project directory
	useFile := filepath.Join(projDir, "use.sro")
	useContent := "var usevar := \"from-use\";\nfn usefn { log(\"from-use\"); }\nseq useseq { usefn(pr.test); }\npar usepar { usefn(pr.test); }\n"
	if err := os.WriteFile(useFile, []byte(useContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create main config that uses the use file
	mainFile := filepath.Join(dir, "main.sro")
	mainContent := fmt.Sprintf(`sanctuary = "%s";
pr test { url = "http://example.com"; dir = "test"; use = "use.sro"; }
`, dir)
	if err := os.WriteFile(mainFile, []byte(mainContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(mainFile)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// Before ResolveUse, use declarations shouldn't be in config
	if len(cfg.Functions) > 0 || len(cfg.Seqs) > 0 || len(cfg.Pars) > 0 {
		t.Fatal("use declarations should not be loaded before ResolveUse")
	}

	// ResolveUse should merge use file declarations
	if err := ResolveUse(cfg); err != nil {
		t.Fatalf("ResolveUse error: %v", err)
	}

	// Check that use file declarations were merged
	if cfg.Vars["usevar"] != "from-use" {
		t.Fatalf("use var not merged: %v", cfg.Vars)
	}
	if cfg.Functions["usefn"] == nil {
		t.Fatal("use fn not merged")
	}
	if cfg.Seqs["useseq"] == nil {
		t.Fatal("use seq not merged")
	}
	if cfg.Pars["usepar"] == nil {
		t.Fatal("use par not merged")
	}
}

func TestResolveUseFileNotFound(t *testing.T) {
	dir := t.TempDir()
	mainFile := filepath.Join(dir, "main.sro")
	content := `sanctuary = "/tmp";
pr test { url = "http://example.com"; dir = "test"; use = "nonexistent.sro"; }
`
	if err := os.WriteFile(mainFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(mainFile)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	err = ResolveUse(cfg)
	if err == nil {
		t.Fatal("expected error for missing use file")
	}
	if !strings.Contains(err.Error(), "use file not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveUseDisallowSanctuaryAndPr(t *testing.T) {
	dir := t.TempDir()

	// Create a use file with sanctuary (should be disallowed)
	useFile := filepath.Join(dir, "bad-use.sro")
	useContent := `sanctuary = "/tmp";
`
	if err := os.WriteFile(useFile, []byte(useContent), 0o644); err != nil {
		t.Fatal(err)
	}

	mainFile := filepath.Join(dir, "main.sro")
	mainContent := fmt.Sprintf(`sanctuary = "/tmp";
pr test { url = "http://example.com"; dir = "test"; use = "%s"; }
`, filepath.Base(useFile))
	if err := os.WriteFile(mainFile, []byte(mainContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(mainFile)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	err = ResolveUse(cfg)
	if err == nil {
		t.Fatal("expected error for sanctuary in use file")
	}
}

func TestLoadWithoutResolveUse(t *testing.T) {
	dir := t.TempDir()
	// Config with undefined fn reference but no use file - should NOT fail
	mainFile := filepath.Join(dir, "main.sro")
	content := `sanctuary = "/tmp";
pr test { url = "http://example.com"; dir = "test"; }
fn bad { log("hello"); }
`
	if err := os.WriteFile(mainFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(mainFile)
	if err != nil {
		t.Fatalf("Load should not fail without ResolveUse: %v", err)
	}

	// validateFull should pass since there are no undefined refs
	if err := validateFull(cfg); err != nil {
		t.Fatalf("validateFull should not fail: %v", err)
	}
}

func TestUndefinedVarInFnBody(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := `sanctuary = "/tmp";
pr test { url = "http://example.com"; dir = "test"; }
fn badfn { log($undefined); }
`
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(file)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// validateFull should catch undefined var
	err = validateFull(cfg)
	if err == nil {
		t.Fatal("expected validateFull to fail due to undefined var")
	}
	if !strings.Contains(err.Error(), "undefined variable $undefined") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSeqCycleDetection(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := `sanctuary = "/tmp";
pr test { url = "http://example.com"; dir = "test"; }
seq a { seq.b; }
seq b { seq.a; }
`
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(file)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// validateFull should catch seq cycle
	err = validateFull(cfg)
	if err == nil {
		t.Fatal("expected validateFull to fail due to seq cycle")
	}
	if !strings.Contains(err.Error(), "seq/par cycle detected") {
		t.Fatalf("unexpected error: %v", err)
	}
}
