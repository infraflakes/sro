package sync

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	git "github.com/go-git/go-git/v5"
	"github.com/infraflakes/sro/internal/config"
)

func Run(cfg *config.Config) error {
	return RunWithContext(context.Background(), cfg, os.Stdout)
}

// WriterFunc is a function that returns a writer for a given project name
type WriterFunc func(projName string) io.Writer

func RunWithWriter(cfg *config.Config, writer io.Writer) error {
	return RunWithContext(context.Background(), cfg, writer)
}

// SyncProject syncs a single project. It writes status to the given writer.
// Returns nil if the project was skipped (sync=ignore) or already exists.
func SyncProject(cfg *config.Config, proj *config.Project, writer io.Writer) error {
	return SyncProjectWithContext(context.Background(), cfg, proj, writer)
}

func SyncProjectWithContext(ctx context.Context, cfg *config.Config, proj *config.Project, writer io.Writer) error {
	if proj.Sync == "ignore" {
		_, _ = fmt.Fprintf(writer, "  skip  %s (sync=ignore)\n", proj.Name)
		return nil
	}

	targetDir := filepath.Join(cfg.Sanctuary, proj.Dir)

	gitDir := filepath.Join(targetDir, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		_, _ = fmt.Fprintf(writer, "  exists  %s → %s\n", proj.Name, targetDir)
		return nil
	}

	_, _ = fmt.Fprintf(writer, "  clone  %s → %s\n", proj.Name, targetDir)
	_, err := git.PlainCloneContext(ctx, targetDir, false, &git.CloneOptions{
		URL:      proj.URL,
		Progress: writer,
	})
	if err != nil {
		return fmt.Errorf("failed to clone %s: %w", proj.Name, err)
	}
	return nil
}

func RunWithContext(ctx context.Context, cfg *config.Config, writer io.Writer) error {
	return RunWithContextFunc(ctx, cfg, func(_ string) io.Writer { return writer })
}

func RunWithWriterFunc(cfg *config.Config, writerFunc WriterFunc) error {
	return RunWithContextFunc(context.Background(), cfg, writerFunc)
}

func RunWithContextFunc(ctx context.Context, cfg *config.Config, writerFunc WriterFunc) error {
	if err := os.MkdirAll(cfg.Sanctuary, 0o755); err != nil {
		return fmt.Errorf("cannot create sanctuary %s: %w", cfg.Sanctuary, err)
	}

	for _, proj := range cfg.Projects {
		writer := writerFunc(proj.Name)
		if writer == nil {
			writer = os.Stdout
		}
		if err := SyncProjectWithContext(ctx, cfg, proj, writer); err != nil {
			return err
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
			_, _ = fmt.Fprintf(writer, "  warn  %s/%s is a git repo not in your config\n", cfg.Sanctuary, name)
		}
	}
}

func warnUnknownReposWithWriter(cfg *config.Config, w io.Writer) {
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
			_, _ = fmt.Fprintf(w, "  warn  %s/%s is a git repo not in your config\n", cfg.Sanctuary, name)
		}
	}
}
