package sync

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	git "github.com/go-git/go-git/v5"
	"github.com/infraflakes/sro/internal/config"
)

func Run(cfg *config.Config) error {
	return RunWithWriter(cfg, os.Stdout)
}

// WriterFunc is a function that returns a writer for a given project name
type WriterFunc func(projName string) io.Writer

func RunWithWriter(cfg *config.Config, writer io.Writer) error {
	return RunWithWriterFunc(cfg, func(_ string) io.Writer { return writer })
}

func RunWithWriterFunc(cfg *config.Config, writerFunc WriterFunc) error {
	if err := os.MkdirAll(cfg.Sanctuary, 0o755); err != nil {
		return fmt.Errorf("cannot create sanctuary %s: %w", cfg.Sanctuary, err)
	}

	for _, proj := range cfg.Projects {
		writer := writerFunc(proj.Name)
		if writer == nil {
			writer = os.Stdout
		}

		if proj.Sync == "ignore" {
			fmt.Fprintf(writer, "  skip  %s (sync=ignore)\n", proj.Name)
			continue
		}

		targetDir := filepath.Join(cfg.Sanctuary, proj.Dir)

		gitDir := filepath.Join(targetDir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			fmt.Fprintf(writer, "  exists  %s → %s\n", proj.Name, targetDir)
			continue
		}

		fmt.Fprintf(writer, "  clone  %s → %s\n", proj.Name, targetDir)
		_, err := git.PlainClone(targetDir, false, &git.CloneOptions{
			URL:      proj.URL,
			Progress: writer,
		})
		if err != nil {
			return fmt.Errorf("failed to clone %s: %w", proj.Name, err)
		}
	}

	warnUnknownRepos(cfg, writerFunc)
	return nil
}

func warnUnknownRepos(cfg *config.Config, writerFunc WriterFunc) {
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
			writer := writerFunc("")
			if writer == nil {
				writer = os.Stdout
			}
			fmt.Fprintf(writer, "  warn  %s/%s is a git repo not in your config\n", cfg.Sanctuary, name)
		}
	}
}
