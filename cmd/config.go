package cmd

import (
	"fmt"
	"tkngate/internal/config"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration related commands",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Parses and validates the configuration file",
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.LoadConfig(); err != nil {
			fmt.Println("❌ Configuration validation failed:", err)
			return
		}
		fmt.Println("✅ Configuration is valid!")
	},
}

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Prints resolved configuration",
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.LoadConfig(); err != nil {
			fmt.Println("Error loading config:", err)
			return
		}
		fmt.Printf("Server: %s:%d\n", config.Cfg.Server.Host, config.Cfg.Server.Port)
		fmt.Printf("Budget Limit: $%.2f\n", config.Cfg.Budget.GlobalLimitUSD)
		for name, p := range config.Cfg.Providers {
			fmt.Printf("Provider [%s]: %s (Default Model: %s)\n", name, p.BaseURL, p.DefaultModel)
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(validateCmd)
	configCmd.AddCommand(showCmd)
}
