package main

import (
	"fmt"
	"os"

	"github.com/infraflakes/sro/config"
	"github.com/infraflakes/sro/runner"
	srSync "github.com/infraflakes/sro/sync"
	"github.com/spf13/cobra"
)

var configPath string

func main() {
	var rootCmd = &cobra.Command{
		Use:   "sro",
		Short: "SRO - Serein Repository Orchestrator",
	}

	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "main.sro", "path to config file")

	var syncCmd = &cobra.Command{
		Use:   "sync",
		Short: "Clone/sync project repositories",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			runSync()
		},
	}

	var seqCmd = &cobra.Command{
		Use:   "seq <name>",
		Short: "Run a sequential block",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runSeq(args[0])
		},
	}

	var parCmd = &cobra.Command{
		Use:   "par <name>",
		Short: "Run a parallel block",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runPar(args[0])
		},
	}

	rootCmd.AddCommand(syncCmd, seqCmd, parCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func loadConfig() *config.Config {
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	return cfg
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
