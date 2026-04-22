package sync

import (
	"fmt"
	"os"
	"path/filepath"

	git "github.com/go-git/go-git/v5"
	"github.com/infraflakes/sro/config"
)

func Run(cfg *config.Config) error {
	if err := os.MkdirAll(cfg.Sanctuary, 0o755); err != nil {
		return fmt.Errorf("cannot create sanctuary %s: %w", cfg.Sanctuary, err)
	}

	for _, proj := range cfg.Projects {
		if proj.Sync == "ignore" {
			fmt.Printf("  skip  %s (sync=ignore)\n", proj.Name)
			continue
		}

		targetDir := filepath.Join(cfg.Sanctuary, proj.Dir)

		gitDir := filepath.Join(targetDir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			fmt.Printf("  exists  %s → %s\n", proj.Name, targetDir)
			continue
		}

		fmt.Printf("  clone  %s → %s\n", proj.Name, targetDir)
		_, err := git.PlainClone(targetDir, false, &git.CloneOptions{
			URL:      proj.URL,
			Progress: os.Stdout,
		})
		if err != nil {
			return fmt.Errorf("failed to clone %s: %w", proj.Name, err)
		}
	}

	warnUnknownRepos(cfg)
	return nil
}

func warnUnknownRepos(cfg *config.Config) {
	knownDirs := map[string]bool{}
	for _, proj := range cfg.Projects {
		knownDirs[proj.Dir] = true
	}

	entries, err := os.ReadDir(cfg.Sanctuary)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if knownDirs[name] {
			continue
		}
		gitDir := filepath.Join(cfg.Sanctuary, name, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			fmt.Printf("  warn  %s/%s is a git repo not in your config\n", cfg.Sanctuary, name)
		}
	}
}
