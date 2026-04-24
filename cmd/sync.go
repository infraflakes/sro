package cmd

import (
	"fmt"
	"os"

	srSync "github.com/infraflakes/sro/internal/sync"
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
	fmt.Printf("sanctuary: %s\n", cfg.Sanctuary)
	fmt.Printf("projects:  %d\n\n", len(cfg.Projects))
	if err := srSync.Run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "sync error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("done")
}
