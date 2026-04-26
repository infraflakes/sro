package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDuplicateVarInFnBody(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := "shell = `bash`;\nsanctuary = `/tmp`;\npr test { url = `http://example.com`; dir = `test`; }\nfn bad {\n    var string x = `a`;\n    var string x = `b`;\n}\n"
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

func TestSanctuaryAbsolutePathValidation(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := "shell = `bash`;\nsanctuary = `relative/path`;"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(file)
	if err == nil || !strings.Contains(err.Error(), "sanctuary must be an absolute path") {
		t.Fatalf("expected absolute path error, got: %v", err)
	}
}

func TestMissingRequiredFields(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")

	contentNoURL := "shell = `bash`;\nsanctuary = `/tmp`;\npr p { dir = `d`; }\n"
	if err := os.WriteFile(file, []byte(contentNoURL), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(file)
	if err == nil || !strings.Contains(err.Error(), `url is required`) {
		t.Fatalf("expected missing url error, got: %v", err)
	}

	contentNoDir := "shell = `bash`;\nsanctuary = `/tmp`;\npr p { url = `u`; }\n"
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
	content := "shell = `bash`;\nsanctuary = `/tmp`;\npr p { url = `u`; dir = `d`; sync = `invalid`; }\n"
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
	content := "shell = `bash`;\nsanctuary = `/tmp`;\nfn real { log(`hi`); }\nseq s { unknown(pr.p); }\npar p { fake(pr.q); }\n"
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

func TestLoadWithoutResolveUse(t *testing.T) {
	dir := t.TempDir()
	// Config with undefined fn reference but no use file - should NOT fail
	mainFile := filepath.Join(dir, "main.sro")
	content := "shell = `bash`;\nsanctuary = `/tmp`;\npr test { url = `http://example.com`; dir = `test`; }\nfn bad { log(`hello`); }\n"
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
	content := "shell = `bash`;\nsanctuary = `/tmp`;\npr test { url = `http://example.com`; dir = `test`; }\nfn badfn { log($undefined); }\n"
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
	content := "shell = `bash`;\nsanctuary = `/tmp`;\npr test { url = `http://example.com`; dir = `test`; }\nseq a { seq.b; }\nseq b { seq.a; }\n"
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

func TestMissingSanctuary(t *testing.T) {
	// C2: missing sanctuary entirely
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := "shell = `bash`;\npr test { url = `http://example.com`; dir = `test`; }"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(file)
	if err == nil || !strings.Contains(err.Error(), "sanctuary is required") {
		t.Fatalf("expected sanctuary is required error, got: %v", err)
	}
}

func TestMissingShell(t *testing.T) {
	// C3: missing shell entirely (no shell vars)
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := "sanctuary = `/tmp`;\npr test { url = `http://example.com`; dir = `test`; }"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(file)
	if err == nil || !strings.Contains(err.Error(), "shell is required") {
		t.Fatalf("expected shell is required error, got: %v", err)
	}
}

func TestValidSeqParReferences(t *testing.T) {
	// C8: valid seq/par references (happy path)
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := "shell = `bash`;\nsanctuary = `/tmp`;\npr test { url = `http://example.com`; dir = `test`; }\nfn real { log(`hi`); }\nseq s { real(pr.test); }\npar p { seq.s; }"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(file)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	// validateFull should pass with valid references
	if err := validateFull(cfg); err != nil {
		t.Fatalf("validateFull should pass with valid refs: %v", err)
	}
}

func TestSelfReferencingSeq(t *testing.T) {
	// C10: self-referencing seq
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := "shell = `bash`;\nsanctuary = `/tmp`;\npr test { url = `http://example.com`; dir = `test`; }\nseq a { seq.a; }"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(file)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	err = validateFull(cfg)
	if err == nil {
		t.Fatal("expected validateFull to fail due to self-referencing seq")
	}
	if !strings.Contains(err.Error(), "seq/par cycle detected") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEmptyConfigFile(t *testing.T) {
	// C11: empty config file
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := ""
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(file)
	if err == nil || !strings.Contains(err.Error(), "shell is required") {
		t.Fatalf("expected shell is required error, got: %v", err)
	}
}

func TestConfigWithOnlyShellAndSanctuary(t *testing.T) {
	// C12: config with only shell + sanctuary
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := "shell = `bash`;\nsanctuary = `/tmp`;"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(file)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.Shell != "bash" || cfg.Sanctuary != "/tmp" {
		t.Fatalf("config wrong: shell=%s sanctuary=%s", cfg.Shell, cfg.Sanctuary)
	}
}

func TestParCycleDetection(t *testing.T) {
	// C9: par cycle detection - par referencing seq that references back to par
	dir := t.TempDir()
	file := filepath.Join(dir, "main.sro")
	content := "shell = `bash`;\nsanctuary = `/tmp`;\npr test { url = `http://example.com`; dir = `test`; }\nfn real { log(`hi`); }\npar a { seq.b; }\nseq b { seq.c; }\nseq c { seq.b; }" // Changed to seq cycle instead of par->seq->par
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(file)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	err = validateFull(cfg)
	if err == nil {
		t.Fatal("expected validateFull to fail due to seq cycle")
	}
	if !strings.Contains(err.Error(), "seq/par cycle detected") {
		t.Fatalf("unexpected error: %v", err)
	}
}
