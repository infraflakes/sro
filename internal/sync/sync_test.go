package sync

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/infraflakes/sro/internal/config"
)

func makeRepo(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
	run("init")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "README.md")
	run("commit", "-m", "initial commit")
}

func TestRun_CloneNewRepo(t *testing.T) {
	sanctuary := t.TempDir()
	repoDir := t.TempDir()
	makeRepo(t, repoDir)

	cfg := &config.Config{
		Sanctuary: sanctuary,
		Projects: map[string]*config.Project{
			"test": {
				Name: "test",
				URL:  repoDir,
				Dir:  "cloned",
				Sync: "clone",
			},
		},
	}

	if err := Run(cfg); err != nil {
		t.Fatalf("Run error: %v", err)
	}

	clonedPath := filepath.Join(sanctuary, "cloned")
	if _, err := os.Stat(clonedPath); err != nil {
		t.Fatalf("cloned dir does not exist: %v", err)
	}
	cmd := exec.Command("git", "-C", clonedPath, "status")
	if err := cmd.Run(); err != nil {
		t.Fatalf("cannot verify cloned repo: %v", err)
	}
}

func TestRun_SkipExistingRepo(t *testing.T) {
	sanctuary := t.TempDir()
	repoDir := t.TempDir()
	makeRepo(t, repoDir)

	clonedPath := filepath.Join(sanctuary, "cloned")
	if err := os.MkdirAll(clonedPath, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init", clonedPath)
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Sanctuary: sanctuary,
		Projects: map[string]*config.Project{
			"test": {
				Name: "test",
				URL:  repoDir,
				Dir:  "cloned",
				Sync: "clone",
			},
		},
	}

	if err := Run(cfg); err != nil {
		t.Fatalf("Run error: %v", err)
	}
}

func TestRun_IgnoreSync(t *testing.T) {
	sanctuary := t.TempDir()
	cfg := &config.Config{
		Sanctuary: sanctuary,
		Projects: map[string]*config.Project{
			"ignored": {
				Name: "ignored",
				URL:  "http://fake.url",
				Dir:  "ignored_dir",
				Sync: "ignore",
			},
		},
	}
	if err := Run(cfg); err != nil {
		t.Fatalf("Run error: %v", err)
	}
	ignoredPath := filepath.Join(sanctuary, "ignored_dir")
	if _, err := os.Stat(ignoredPath); err == nil || !os.IsNotExist(err) {
		t.Fatalf("ignored project dir should not exist")
	}
}

func TestRun_CreateSanctuary(t *testing.T) {
	sanctuary := filepath.Join(t.TempDir(), "new-sanctuary")
	repoDir := t.TempDir()
	makeRepo(t, repoDir)

	cfg := &config.Config{
		Sanctuary: sanctuary,
		Projects: map[string]*config.Project{
			"test": {
				Name: "test",
				URL:  repoDir,
				Dir:  "cloned",
				Sync: "clone",
			},
		},
	}

	if err := Run(cfg); err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if _, err := os.Stat(sanctuary); err != nil {
		t.Fatalf("sanctuary directory not created")
	}
}

func TestRun_WarnUnknownRepo(t *testing.T) {
	sanctuary := t.TempDir()
	unknownDir := filepath.Join(sanctuary, "unknown-repo")
	if err := os.MkdirAll(unknownDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init", unknownDir)
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Sanctuary: sanctuary,
		Projects:  map[string]*config.Project{},
	}
	if err := Run(cfg); err != nil {
		t.Fatalf("Run error: %v", err)
	}
}

func TestWarnUnknownReposOutput(t *testing.T) {
	sanctuary := t.TempDir()
	unknownDir := filepath.Join(sanctuary, "unknown-repo")
	if err := os.MkdirAll(unknownDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init", unknownDir)
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Sanctuary: sanctuary,
		Projects:  map[string]*config.Project{},
	}

	var buf bytes.Buffer
	warnUnknownReposWithWriter(cfg, &buf)

	output := buf.String()
	if !strings.Contains(output, "warn") {
		t.Fatalf("expected output to contain 'warn', got: %q", output)
	}
	if !strings.Contains(output, "unknown-repo") {
		t.Fatalf("expected output to contain 'unknown-repo', got: %q", output)
	}
	if !strings.Contains(output, "git repo not in your config") {
		t.Fatalf("expected output to contain 'git repo not in your config', got: %q", output)
	}
}

func TestRun_CloneFailure(t *testing.T) {
	sanctuary := t.TempDir()
	cfg := &config.Config{
		Sanctuary: sanctuary,
		Projects: map[string]*config.Project{
			"bad": {Name: "bad", URL: "http://invalid.invalid/no.git", Dir: "bad", Sync: "clone"},
		},
	}
	err := Run(cfg)
	if err == nil {
		t.Fatal("expected clone error")
	}
	if !strings.Contains(err.Error(), "failed to clone bad") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_SanctuaryPermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	parent := t.TempDir()
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })
	cfg := &config.Config{
		Sanctuary: filepath.Join(parent, "child"),
		Projects:  map[string]*config.Project{},
	}
	err := Run(cfg)
	if err == nil {
		t.Fatal("expected sanctuary creation error")
	}
	if !strings.Contains(err.Error(), "cannot create sanctuary") {
		t.Fatalf("unexpected error: %v", err)
	}
}
