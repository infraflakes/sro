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
