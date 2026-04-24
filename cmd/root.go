package cmd

import (
	"fmt"
	"os"

	"github.com/infraflakes/sro/internal/config"
	"github.com/spf13/cobra"
)

var configPath string

var rootCmd = &cobra.Command{
	Use:   "sro",
	Short: "SRO - Serein Repository Orchestrator",
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "main.sro", "path to config file")
	rootCmd.AddCommand(syncCmd, seqCmd, parCmd)
}

func Execute() {
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
