package sync

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/infraflakes/sro/internal/config"
)

func makeRepo(t *testing.T, dir string) {
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "README.md")
	if err := os.WriteFile(file, []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}
	w, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Add("README.md"); err != nil {
		t.Fatal(err)
	}
	_, err = w.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@test.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
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
	if _, err := git.PlainOpen(clonedPath); err != nil {
		t.Fatalf("cannot open cloned repo: %v", err)
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
	if _, err := git.PlainInit(clonedPath, false); err != nil {
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
	if _, err := git.PlainInit(unknownDir, false); err != nil {
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
