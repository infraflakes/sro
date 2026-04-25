package cmd

import (
	"fmt"
	"os"

	"github.com/infraflakes/sro/internal/config"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Parse and validate the configuration file",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		runValidate()
	},
}

func runValidate() {
	cfg := loadConfig()
	if err := config.ResolveUse(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "validation error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("config is valid")
}
