package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/infraflakes/sro/internal/config"
	"github.com/spf13/cobra"
)

var configPath string

var rootCmd = &cobra.Command{
	Use:   "sro",
	Short: "SRO - Serein Repository Orchestrator",
}

func init() {
	defaultConfig := "config.sro"
	if configDir, err := os.UserConfigDir(); err == nil {
		defaultConfig = filepath.Join(configDir, "sro", "config.sro")
	}

	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", defaultConfig, "path to config file")
	rootCmd.AddCommand(syncCmd, seqCmd, parCmd, validateCmd)
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
