package cmd

import (
	"fmt"
	"os"

	"github.com/infraflakes/sro/internal/config"
	"github.com/infraflakes/sro/internal/runner"
	"github.com/spf13/cobra"
)

var seqCmd = &cobra.Command{
	Use:   "seq <name>",
	Short: "Run a sequential block",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runSeq(args[0])
	},
}

func runSeq(name string) {
	cfg := loadConfig()
	if err := config.ResolveUse(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	r := runner.New(cfg)
	if err := r.RunSeq(name); err != nil {
		fmt.Fprintf(os.Stderr, "seq error: %v\n", err)
		os.Exit(1)
	}
}
