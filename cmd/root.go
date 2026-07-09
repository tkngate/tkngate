package cmd

import (
	"fmt"
	"os"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "tkngate",
	Version: "v1.9.4-tool-calling-support",
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

// ── GLOBAL THEME (TrueColor RGB) ──
var (
	Gold   = pterm.NewRGBStyle(pterm.NewRGB(184, 151, 82)).Sprint // #b89752 warm amber gold
	Forest = pterm.NewRGBStyle(pterm.NewRGB(22, 43, 29)).Sprint   // #162b1d dark forest green
	Parch  = pterm.NewRGBStyle(pterm.NewRGB(244, 236, 216)).Sprint // #f4ecd8 parchment
)

func init() {
	// ── Marketing Page Colors ──────────────────────────────────────────────────
	// bg-[#162b1d] = dark forest green  (text/borders on marketing page)
	// #b89752      = warm amber gold    (accent color on marketing page)
	// #f4ecd8      = parchment          (background on marketing page)
	cssForest := pterm.NewRGB(22, 43, 29)       // #162b1d
	cssGold := pterm.NewRGB(184, 151, 82)       // #b89752
	cssParchment := pterm.NewRGB(244, 236, 216) // #f4ecd8

	// Wire pterm global sprint helpers to exact CSS hex values
	pterm.Yellow = cssGold.Sprint
	pterm.LightYellow = cssGold.Sprint
	pterm.Green = cssForest.Sprint
	pterm.LightGreen = cssForest.Sprint
	pterm.Gray = cssParchment.Sprint

	// Wire theme styles — spinners, tables, boxes, etc.
	pterm.ThemeDefault.PrimaryStyle = *pterm.NewStyle(pterm.FgYellow)
	pterm.ThemeDefault.SecondaryStyle = *pterm.NewStyle(pterm.FgYellow)
	pterm.ThemeDefault.DefaultText = *pterm.NewStyle(pterm.FgYellow)
	pterm.ThemeDefault.HighlightStyle = *pterm.NewStyle(pterm.Bold, pterm.FgYellow)
	pterm.ThemeDefault.InfoMessageStyle = *pterm.NewStyle(pterm.FgYellow)
	pterm.ThemeDefault.InfoPrefixStyle = *pterm.NewStyle(pterm.FgBlack, pterm.BgYellow)
	pterm.ThemeDefault.SuccessMessageStyle = *pterm.NewStyle(pterm.FgYellow)
	pterm.ThemeDefault.SuccessPrefixStyle = *pterm.NewStyle(pterm.FgBlack, pterm.BgYellow)
	pterm.ThemeDefault.WarningMessageStyle = *pterm.NewStyle(pterm.FgYellow)
	pterm.ThemeDefault.WarningPrefixStyle = *pterm.NewStyle(pterm.FgBlack, pterm.BgYellow)
	pterm.ThemeDefault.ErrorMessageStyle = *pterm.NewStyle(pterm.FgLightRed)
	pterm.ThemeDefault.ErrorPrefixStyle = *pterm.NewStyle(pterm.FgBlack, pterm.BgLightRed)
	pterm.ThemeDefault.SpinnerStyle = *pterm.NewStyle(pterm.FgYellow)
	pterm.ThemeDefault.SpinnerTextStyle = *pterm.NewStyle(pterm.FgYellow)
	pterm.ThemeDefault.TableHeaderStyle = *pterm.NewStyle(pterm.FgYellow)
	pterm.ThemeDefault.TableSeparatorStyle = *pterm.NewStyle(pterm.FgYellow)
	pterm.ThemeDefault.BoxStyle = *pterm.NewStyle(pterm.FgYellow)
	pterm.ThemeDefault.BoxTextStyle = *pterm.NewStyle(pterm.FgYellow)
	pterm.ThemeDefault.LetterStyle = *pterm.NewStyle(pterm.FgYellow)
	pterm.ThemeDefault.TreeStyle = *pterm.NewStyle(pterm.FgYellow)
	pterm.ThemeDefault.TreeTextStyle = *pterm.NewStyle(pterm.FgYellow)
	pterm.ThemeDefault.BulletListTextStyle = *pterm.NewStyle(pterm.FgYellow)
	pterm.ThemeDefault.BulletListBulletStyle = *pterm.NewStyle(pterm.FgYellow)
	pterm.ThemeDefault.Checkmark = pterm.Checkmark{
		Checked:   cssGold.Sprint("✓"),
		Unchecked: pterm.Red("✗"),
	}

	// ── Startup Banner ─────────────────────────────────────────────────────────
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		fmt.Println()

		// ── Logo: TKN in gold, GATE in forest green ──
		// Split into two separate art arrays so each half gets its own true RGB color
		tknArt := []string{
			"████████ ██   ██ ███    ██",
			"   ██    ██  ██  ████   ██",
			"   ██    █████   ██ ██  ██",
			"   ██    ██  ██  ██  ██ ██",
			"   ██    ██   ██ ██   ████",
		}
		gateArt := []string{
			"  ██████   █████  ████████ ███████ ",
			" ██       ██   ██    ██    ██      ",
			" ██   ███ ███████    ██    █████   ",
			" ██    ██ ██   ██    ██    ██      ",
			"  ██████  ██   ██    ██    ███████ ",
		}
		for i := range tknArt {
			fmt.Printf("%s%s\n", Gold(tknArt[i]), Forest(gateArt[i]))
		}
		fmt.Println()

		// ── Info box ──
		fmt.Println(Gold("┌────────────────────────────────────────────────────────┐"))
		fmt.Println(Gold("│") + "  The Cloudflare for Autonomous AI Agents               " + Gold("│"))
		fmt.Println(Gold("│") + "  " + Parch("Enterprise zero-knowledge reverse proxy for LLM APIs") + "  " + Gold("│"))
		fmt.Println(Gold("└────────────────────────────────────────────────────────┘"))
		fmt.Println()

		// ── Main Menu ──
		menuOptions := []string{
			"serve  (Start Proxy Server)",
			"auth   (Manage Virtual Keys)",
			"budget (Check Budget Status)",
			"cache  (Semantic Cache Status)",
			"config (Configure Tkngate)",
			"pool   (P2P Mesh Pool Status)",
			"exit   (Close)",
		}

		// Helper to create a fresh RGB menu (prevents pterm internal index panics)
		createMenu := func(title string, opts []string) string {
			colOpts := make([]string, len(opts))
			for i, o := range opts {
				colOpts[i] = Gold(o)
			}
			sel := pterm.DefaultInteractiveSelect
			sel.TextStyle = pterm.NewStyle()
			sel.SelectorStyle = pterm.NewStyle()
			sel.OptionStyle = pterm.NewStyle()
			sel.Selector = Forest(">")
			sel.Filter = false
			sel.DefaultText = Gold(title)
			sel.Options = colOpts

			res, _ := sel.Show()
			fmt.Println()
			return pterm.RemoveColorFromString(res)
		}

		for {
			selectedOption := createMenu("What would you like to do?", menuOptions)
			switch selectedOption {
			case "serve  (Start Proxy Server)":
				serveCmd.Run(serveCmd, []string{})
				return
			case "auth   (Manage Virtual Keys)":
				manageOpts := []string{"List Keys", "Issue New Key", "Revoke Key", "Back"}
				action := createMenu("Virtual Keys", manageOpts)

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
					if revokeName, _ := pterm.DefaultInteractiveTextInput.Show("Enter Key Name to revoke"); revokeName != "" {
						virtualKeyName = revokeName
						revokeCmd.Run(revokeCmd, []string{})
					}
				}
			case "budget (Check Budget Status)":
				statusCmd.Run(statusCmd, []string{})
			case "cache  (Semantic Cache Status)":
				cacheStatusCmd.Run(cacheStatusCmd, []string{})
			case "config (Configure Tkngate)":
				showCmd.Run(showCmd, []string{})
			case "pool   (P2P Mesh Pool Status)":
				poolOpts := []string{"View Pool Status", "Donate API Key", "Back"}
				poolAction := createMenu("P2P Token Mesh", poolOpts)

				switch poolAction {
				case "View Pool Status":
					poolStatusCmd.RunE(poolStatusCmd, []string{})
				case "Donate API Key":
					provOpts := []string{"openai", "anthropic", "deepseek", "mistral", "groq", "ollama"}
					provider := createMenu("Select Provider", provOpts)
					
					poolProvider = provider
					donateCmd.RunE(donateCmd, []string{})
				}
			case "exit   (Close)":
				os.Exit(0)
			default:
				if selectedOption == "" {
					os.Exit(0)
				}
			}
		}
	}
}
