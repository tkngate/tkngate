package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"tkngate/internal/config"

	"github.com/fatih/color"
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
			color.Red("❌ Error loading config: %v", err)
			return
		}
		
		fmt.Println()
		color.Cyan("=== Tkngate Configuration (v1.2.0) ===")
		fmt.Println()
		
		// Server
		color.White("🌐 Server")
		fmt.Printf("   URL:      %s\n", color.GreenString("http://%s:%d", config.Cfg.Server.Host, config.Cfg.Server.Port))
		fmt.Printf("   Metrics:  %s\n", color.GreenString("http://%s:%d", config.Cfg.Telemetry.Host, config.Cfg.Telemetry.Port))
		fmt.Println()

		// Budget
		color.White("💰 Budget & Safety")
		fmt.Printf("   Global Limit:     %s\n", color.YellowString("$%.2f / %s", config.Cfg.Budget.GlobalLimitUSD, config.Cfg.Budget.ResetInterval))
		fmt.Printf("   Session Limit:    %s\n", color.YellowString("$%.2f", config.Cfg.Budget.MaxSessionCostUSD))
		fmt.Printf("   Fallback Model:   %s (via %s)\n", color.MagentaString(config.Cfg.Budget.FallbackModel), config.Cfg.Budget.FallbackProvider)
		fmt.Println()

		// Shadow Mode
		color.White("👻 Shadow Mode")
		if config.Cfg.Shadow.Enabled {
			fmt.Printf("   Status:           %s\n", color.GreenString("ACTIVE"))
			fmt.Printf("   Target:           %s (via %s)\n", color.MagentaString(config.Cfg.Shadow.TargetModel), config.Cfg.Shadow.TargetProvider)
			fmt.Printf("   Traffic Fraction: %s\n", color.YellowString("%.0f%%", config.Cfg.Shadow.TrafficFraction*100))
		} else {
			fmt.Printf("   Status:           %s\n", color.RedString("DISABLED"))
		}
		fmt.Println()

		// Providers
		color.White("🔌 Configured Providers")
		for name, p := range config.Cfg.Providers {
			fmt.Printf("   - %s\n", color.BlueString(name))
			fmt.Printf("       Default Model: %s\n", color.MagentaString(p.DefaultModel))
			fmt.Printf("       Endpoint:      %s\n", p.BaseURL)
		}
		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(validateCmd)
	configCmd.AddCommand(showCmd)
	configCmd.AddCommand(generateMasterKeyCmd)
}

var generateMasterKeyCmd = &cobra.Command{
	Use:   "generate-master-key",
	Short: "Generates a 32-character secure random master key",
	Run: func(cmd *cobra.Command, args []string) {
		bytes := make([]byte, 16)
		if _, err := rand.Read(bytes); err != nil {
			color.Red("❌ Error generating key: %v", err)
			return
		}
		key := hex.EncodeToString(bytes)
		
		fmt.Println()
		color.Green("✅ Successfully generated a secure TKNGATE_MASTER_KEY!")
		fmt.Println()
		color.Cyan("   %s", key)
		fmt.Println()
		color.Yellow("⚠️  IMPORTANT: Copy this key and set it as an environment variable.")
		color.Yellow("   Do not lose this key! If lost, all donated mesh keys will be unrecoverable.")
		fmt.Println()
		fmt.Println("   Linux/macOS: export TKNGATE_MASTER_KEY=\"" + key + "\"")
		fmt.Println("   Windows:     $env:TKNGATE_MASTER_KEY=\"" + key + "\"")
		fmt.Println()
	},
}
