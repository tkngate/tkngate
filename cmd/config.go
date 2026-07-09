package cmd

import (
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
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println(Gold("■ TKNGATE CONFIGURATION (") + Gold(rootCmd.Version) + Gold(")"))
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println()

		var shadowStatus string
		if config.Cfg.Shadow.Enabled {
			shadowStatus = Forest("ACTIVE")
		} else {
			shadowStatus = pterm.LightRed("DISABLED")
		}

		tree := pterm.TreeNode{
			Text: "Configuration",
			Children: []pterm.TreeNode{
				{
					Text: "Server",
					Children: []pterm.TreeNode{
						{Text: pterm.Sprintf("URL:      %s", Forest(fmt.Sprintf("http://%s:%d", config.Cfg.Server.Host, config.Cfg.Server.Port)))},
						{Text: pterm.Sprintf("Metrics:  %s", Forest(fmt.Sprintf("http://%s:%d", config.Cfg.Telemetry.Host, config.Cfg.Telemetry.Port)))},
					},
				},
				{
					Text: "Budget & Safety",
					Children: []pterm.TreeNode{
						{Text: pterm.Sprintf("Global Limit:     %s", Gold(fmt.Sprintf("$%.2f / %s", config.Cfg.Budget.GlobalLimitUSD, config.Cfg.Budget.ResetInterval)))},
						{Text: pterm.Sprintf("Session Limit:    %s", Gold(fmt.Sprintf("$%.2f", config.Cfg.Budget.MaxSessionCostUSD)))},
						{Text: pterm.Sprintf("Fallback Model:   %s (via %s)", Gold(config.Cfg.Budget.FallbackModel), config.Cfg.Budget.FallbackProvider)},
					},
				},
				{
					Text: "Shadow Mode",
					Children: []pterm.TreeNode{
						{Text: pterm.Sprintf("Status:           %s", shadowStatus)},
						{Text: pterm.Sprintf("Target:           %s (via %s)", Gold(config.Cfg.Shadow.TargetModel), config.Cfg.Shadow.TargetProvider)},
						{Text: pterm.Sprintf("Traffic Fraction: %s", Gold(fmt.Sprintf("%.0f%%", config.Cfg.Shadow.TrafficFraction*100)))},
					},
				},
			},
		}

		providerNodes := []pterm.TreeNode{}
		for name, p := range config.Cfg.Providers {
			providerNodes = append(providerNodes, pterm.TreeNode{
				Text: Gold(name),
				Children: []pterm.TreeNode{
					{Text: pterm.Sprintf("Default Model: %s", Gold(p.DefaultModel))},
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
}


