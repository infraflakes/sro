package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"

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
		fmt.Printf("sanctuary: %s\n", cfg.Sanctuary)
		fmt.Printf("projects:  %d\n\n", len(cfg.Projects))
		if err := srSync.Run(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "sync error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("done")
		return
	}

	model := &tui.Model{
		Type:     "sync",
		Name:     "sync",
		Status:   "running",
		Selected: 0,
	}

	// Create a buffer per repo
	for _, proj := range cfg.Projects {
		buf := &bytes.Buffer{}
		model.Tasks = append(model.Tasks, tui.Task{
			Label:    proj.Name,
			Status:   "pending",
			Expanded: false,
			Output:   buf,
			Writer:   buf,
		})
	}

	// Run sync in background goroutine
	go func() {
		if err := srSync.RunWithWriterFunc(cfg, func(projName string) io.Writer {
			for _, task := range model.Tasks {
				if task.Label == projName {
					return task.Writer
				}
			}
			return os.Stdout
		}); err != nil {
			model.Status = "failed"
		} else {
			model.Status = "ok"
		}
	}()

	if err := tui.Run(model); err != nil {
		fmt.Fprintf(os.Stderr, "tui error: %v\n", err)
		os.Exit(1)
	}
}
