package cmd

import (
	"fmt"
	"os"

	"github.com/infraflakes/sro/internal/config"
	"github.com/infraflakes/sro/internal/runner"
	"github.com/spf13/cobra"
)

var parCmd = &cobra.Command{
	Use:   "par <name>",
	Short: "Run a parallel block",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runPar(args[0])
	},
}

func runPar(name string) {
	cfg := loadConfig()
	if err := config.ResolveUse(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	r := runner.New(cfg)
	if err := r.RunPar(name); err != nil {
		fmt.Fprintf(os.Stderr, "par error: %v\n", err)
		os.Exit(1)
	}
}
