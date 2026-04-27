package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/gdamore/tcell/v3/vt"
	"github.com/infraflakes/sro/internal/config"
	srSync "github.com/infraflakes/sro/internal/sync"
	"github.com/infraflakes/sro/internal/tui"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Clone/sync project repositories",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		runSync()
	},
}

func runSync() {
	cfg := loadConfig()

	// Fallback to plain stdout if --no-tui is set
	if noTui {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		fmt.Printf("sanctuary: %s\n", cfg.Sanctuary)
		fmt.Printf("projects:  %d\n\n", len(cfg.Projects))
		if err := srSync.RunWithContext(ctx, cfg, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "sync error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("done")
		return
	}

	// Ensure sanctuary directory exists before launching goroutines
	if err := os.MkdirAll(cfg.Sanctuary, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "cannot create sanctuary: %v\n", err)
		os.Exit(1)
	}

	model := &tui.Model{
		Type:         "sync",
		Name:         "sync",
		Status:       "running",
		Selected:     0,
		ScrollOffset: 0,
	}

	// Create a vterm per project
	projNames := make([]string, 0, len(cfg.Projects))
	for name := range cfg.Projects {
		projNames = append(projNames, name)
	}

	for _, name := range projNames {
		proj := cfg.Projects[name]
		vterm := vt.NewMockTerm(vt.MockOptSize(vt.Coord{X: 120, Y: 100}), vt.MockOptColors(1<<24))
		if err := vterm.Start(); err != nil {
			continue
		}
		_, _ = vterm.Write([]byte("\x1b[20h")) // enable newline mode
		model.Tasks = append(model.Tasks, tui.Task{
			Label:    proj.Name,
			Status:   "running", // ALL start as "running" immediately
			Expanded: false,
			VTerm:    vterm,
		})
	}

	// Run sync concurrently — one goroutine per project (like par.go)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		var wg sync.WaitGroup
		var mu sync.Mutex
		var hasFailed bool

		for i, name := range projNames {
			proj := cfg.Projects[name]
			wg.Add(1)
			go func(idx int, p *config.Project) {
				defer wg.Done()

				// Status is already "running" from initialization

				err := srSync.SyncProjectWithContext(ctx, cfg, p, tui.NewLineCountingWriter(model.Tasks[idx].VTerm, &model.Tasks[idx].TotalLines))

				if err != nil {
					model.Tasks[idx].Status = "failed"
					// Write error to vterm so user can see it when expanded
					_, _ = fmt.Fprintf(model.Tasks[idx].VTerm, "\033[38;2;224;92;106m%v\033[0m\n", err)
					mu.Lock()
					hasFailed = true
					mu.Unlock()
				} else {
					model.Tasks[idx].Status = "ok"
				}
			}(i, proj)
		}

		wg.Wait()

		// Skip warnUnknownRepos in TUI mode

		mu.Lock()
		if hasFailed {
			model.Status = "failed"
		} else {
			model.Status = "ok"
		}
		mu.Unlock()
	}()

	if err := tui.RunWithContext(ctx, model); err != nil {
		fmt.Fprintf(os.Stderr, "tui error: %v\n", err)
		os.Exit(1)
	}
}
