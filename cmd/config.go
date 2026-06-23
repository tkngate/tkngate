package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"tkngate/internal/config"

	"github.com/pterm/pterm"
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
		spinner, _ := pterm.DefaultSpinner.Start("Validating configuration...")
		if err := config.LoadConfig(); err != nil {
			spinner.Fail("Configuration validation failed: ", err.Error())
			return
		}
		spinner.Success("Configuration is valid!")
	},
}

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Prints resolved configuration",
	Run: func(cmd *cobra.Command, args []string) {
		spinner, _ := pterm.DefaultSpinner.Start("Loading configuration...")
		if err := config.LoadConfig(); err != nil {
			spinner.Fail("Error loading config: ", err.Error())
			return
		}
		spinner.Stop()

		fmt.Println()
		pterm.DefaultHeader.WithFullWidth().WithBackgroundStyle(pterm.NewStyle(pterm.BgCyan)).Println(pterm.Sprintf("Tkngate Configuration (%s)", rootCmd.Version))
		fmt.Println()

		var shadowStatus string
		if config.Cfg.Shadow.Enabled {
			shadowStatus = pterm.LightGreen("ACTIVE")
		} else {
			shadowStatus = pterm.LightRed("DISABLED")
		}

		tree := pterm.TreeNode{
			Text: "Configuration",
			Children: []pterm.TreeNode{
				{
					Text: "Server",
					Children: []pterm.TreeNode{
						{Text: pterm.Sprintf("URL:      %s", pterm.LightGreen(fmt.Sprintf("http://%s:%d", config.Cfg.Server.Host, config.Cfg.Server.Port)))},
						{Text: pterm.Sprintf("Metrics:  %s", pterm.LightGreen(fmt.Sprintf("http://%s:%d", config.Cfg.Telemetry.Host, config.Cfg.Telemetry.Port)))},
					},
				},
				{
					Text: "Budget & Safety",
					Children: []pterm.TreeNode{
						{Text: pterm.Sprintf("Global Limit:     %s", pterm.LightYellow(fmt.Sprintf("$%.2f / %s", config.Cfg.Budget.GlobalLimitUSD, config.Cfg.Budget.ResetInterval)))},
						{Text: pterm.Sprintf("Session Limit:    %s", pterm.LightYellow(fmt.Sprintf("$%.2f", config.Cfg.Budget.MaxSessionCostUSD)))},
						{Text: pterm.Sprintf("Fallback Model:   %s (via %s)", pterm.LightMagenta(config.Cfg.Budget.FallbackModel), config.Cfg.Budget.FallbackProvider)},
					},
				},
				{
					Text: "Shadow Mode",
					Children: []pterm.TreeNode{
						{Text: pterm.Sprintf("Status:           %s", shadowStatus)},
						{Text: pterm.Sprintf("Target:           %s (via %s)", pterm.LightMagenta(config.Cfg.Shadow.TargetModel), config.Cfg.Shadow.TargetProvider)},
						{Text: pterm.Sprintf("Traffic Fraction: %s", pterm.LightYellow(fmt.Sprintf("%.0f%%", config.Cfg.Shadow.TrafficFraction*100)))},
					},
				},
			},
		}

		providerNodes := []pterm.TreeNode{}
		for name, p := range config.Cfg.Providers {
			providerNodes = append(providerNodes, pterm.TreeNode{
				Text: pterm.LightBlue(name),
				Children: []pterm.TreeNode{
					{Text: pterm.Sprintf("Default Model: %s", pterm.LightMagenta(p.DefaultModel))},
					{Text: pterm.Sprintf("Endpoint:      %s", p.BaseURL)},
				},
			})
		}
		tree.Children = append(tree.Children, pterm.TreeNode{
			Text:     "Configured Providers",
			Children: providerNodes,
		})

		pterm.DefaultTree.WithRoot(tree).Render()
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
		spinner, _ := pterm.DefaultSpinner.Start("Generating secure master key...")
		bytes := make([]byte, 16)
		if _, err := rand.Read(bytes); err != nil {
			spinner.Fail("Error generating key: ", err.Error())
			return
		}
		spinner.Success("Successfully generated a secure TKNGATE_MASTER_KEY!")
		key := hex.EncodeToString(bytes)

		fmt.Println()

		boxContent := pterm.Sprintf("%s\n\n%s\n%s\n\nLinux/macOS: %s\nWindows:     %s",
			pterm.LightCyan(key),
			pterm.LightYellow("IMPORTANT: Copy this key and set it as an environment variable."),
			pterm.LightYellow("Do not lose this key! If lost, all donated mesh keys will be unrecoverable."),
			pterm.Gray("export TKNGATE_MASTER_KEY=\""+key+"\""),
			pterm.Gray("$env:TKNGATE_MASTER_KEY=\""+key+"\""))

		pterm.DefaultBox.WithTitle("Master Key").WithRightPadding(2).WithLeftPadding(2).Println(boxContent)
		fmt.Println()
	},
}
