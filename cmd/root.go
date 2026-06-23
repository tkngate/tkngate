package cmd

import (
	"fmt"
	"os"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "tkngate",
	Version: "v1.7.0",
	Short:   "tkngate is an enterprise token-management reverse proxy",
	Long: `tkngate is a zero-knowledge reverse proxy daemon for LLM APIs.
It provides P2P token pooling, real-time budget enforcement, and semantic caching.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		pterm.DefaultBigText.WithLetters(
			pterm.NewLettersFromStringWithStyle("TKN", pterm.NewStyle(pterm.FgCyan)),
			pterm.NewLettersFromStringWithStyle("GATE", pterm.NewStyle(pterm.FgLightBlue)),
		).Render()

		boxContent := pterm.Sprintf("The Cloudflare for Autonomous AI Agents\n%s", pterm.Gray("Enterprise zero-knowledge reverse proxy for LLM APIs"))
		pterm.DefaultBox.WithRightPadding(2).WithLeftPadding(2).Println(boxContent)

		for {
			options := []string{
				"Start Proxy Server",
				"Manage Virtual Keys",
				"Check Budget Status",
				"Configure Tkngate",
				"P2P Mesh Pool Status",
				"Exit",
			}

			selectedOption, _ := pterm.DefaultInteractiveSelect.WithDefaultText("What would you like to do?").WithOptions(options).Show()
			fmt.Println()

			switch selectedOption {
			case "Start Proxy Server":
				serveCmd.Run(serveCmd, []string{})
				return
			case "Manage Virtual Keys":
				manageOpts := []string{"List Keys", "Issue New Key", "Revoke Key", "Back"}
				action, _ := pterm.DefaultInteractiveSelect.WithDefaultText("Virtual Keys").WithOptions(manageOpts).Show()
				fmt.Println()
				switch action {
				case "List Keys":
					listCmd.Run(listCmd, []string{})
				case "Issue New Key":
					name, _ := pterm.DefaultInteractiveTextInput.Show("Enter name for the new key (e.g. Agent-1)")
					if name != "" {
						limitStr, _ := pterm.DefaultInteractiveTextInput.Show("Enter USD budget limit (e.g. 10.50)")
						var limit float64 = 10.0
						if limitStr != "" {
							fmt.Sscanf(limitStr, "%f", &limit)
						}
						virtualKeyName = name
						virtualKeyLimit = limit
						generateCmd.Run(generateCmd, []string{})
					}
				case "Revoke Key":
					name, _ := pterm.DefaultInteractiveTextInput.Show("Enter name of the key to revoke")
					if name != "" {
						virtualKeyName = name
						revokeCmd.Run(revokeCmd, []string{})
					}
				}
			case "Check Budget Status":
				statusCmd.Run(statusCmd, []string{})
			case "Configure Tkngate":
				showCmd.Run(showCmd, []string{})
			case "P2P Mesh Pool Status":
				poolOpts := []string{"View Pool Status", "Donate API Key", "Back"}
				poolAction, _ := pterm.DefaultInteractiveSelect.WithDefaultText("P2P Token Mesh").WithOptions(poolOpts).Show()
				fmt.Println()
				switch poolAction {
				case "View Pool Status":
					poolStatusCmd.RunE(poolStatusCmd, []string{})
				case "Donate API Key":
					provider, _ := pterm.DefaultInteractiveSelect.WithDefaultText("Select Provider").WithOptions([]string{"openai", "anthropic", "deepseek"}).Show()
					
					// poolKey will be prompted securely inside donateCmd if we leave it empty!
					poolProvider = provider
					poolKey = "" // reset it so the secure masked prompt triggers
					poolLimit = 100000 // default limit
					
					limitStr, _ := pterm.DefaultInteractiveTextInput.Show("Enter TPM Limit for this key (e.g. 100000)")
					if limitStr != "" {
						fmt.Sscanf(limitStr, "%d", &poolLimit)
					}
					
					donateCmd.RunE(donateCmd, []string{})
				}
			case "Exit":
				os.Exit(0)
			default:
				if selectedOption == "" {
					os.Exit(0)
				}
			}

			fmt.Println()
		}
	}
}
