package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveUse(t *testing.T) {
	dir := t.TempDir()

	// Create project directory inside sanctuary
	projDir := filepath.Join(dir, "test")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a use file inside the project directory
	useFile := filepath.Join(projDir, "use.sro")
	useContent := "var string usevar = `from-use`;\nfn usefn { log(`from-use`); }\nseq useseq { usefn(pr.test); }\npar usepar { usefn(pr.test); }\n"
	if err := os.WriteFile(useFile, []byte(useContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create main config that uses the use file
	mainFile := filepath.Join(dir, "main.sro")
	mainContent := fmt.Sprintf("shell = `bash`;\nsanctuary = `%s`;\npr test { url = `http://example.com`; dir = `test`; use = `use.sro`; }\n", dir)
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
	content := "shell = `bash`;\nsanctuary = `/tmp`;\npr test { url = `http://example.com`; dir = `test`; use = `nonexistent.sro`; }\n"
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
	useContent := "shell = `bash`; sanctuary = `/tmp`;\n"
	if err := os.WriteFile(useFile, []byte(useContent), 0o644); err != nil {
		t.Fatal(err)
	}

	mainFile := filepath.Join(dir, "main.sro")
	mainContent := fmt.Sprintf("shell = `bash`;\nsanctuary = `/tmp`;\npr test { url = `http://example.com`; dir = `test`; use = `%s`; }\n", filepath.Base(useFile))
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

func TestResolveUseDisallowPr(t *testing.T) {
	// C7: use file containing pr block
	dir := t.TempDir()

	// Create a use file with pr block (should be disallowed)
	useFile := filepath.Join(dir, "bad-use.sro")
	useContent := "shell = `bash`; pr x { url = `u`; dir = `d`; }"
	if err := os.WriteFile(useFile, []byte(useContent), 0o644); err != nil {
		t.Fatal(err)
	}

	mainFile := filepath.Join(dir, "main.sro")
	mainContent := fmt.Sprintf("shell = `bash`;\nsanctuary = `/tmp`;\npr test { url = `http://example.com`; dir = `test`; use = `%s`; }\n", filepath.Base(useFile))
	if err := os.WriteFile(mainFile, []byte(mainContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(mainFile)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	err = ResolveUse(cfg)
	if err == nil {
		t.Fatal("expected error for pr in use file")
	}
	// The error could be about the use file not being found or about pr being disallowed
	if !strings.Contains(err.Error(), "cannot declare projects") && !strings.Contains(err.Error(), "use file not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}
