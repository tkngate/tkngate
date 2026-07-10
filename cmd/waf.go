package cmd

import (
	"fmt"
	"tkngate/internal/config"
	"tkngate/internal/waf"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var wafCmd = &cobra.Command{
	Use:   "waf",
	Short: "AI Web Application Firewall management",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var wafStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current WAF engine status and rule counts",
	Run: func(cmd *cobra.Command, args []string) {
		spinner, _ := pterm.DefaultSpinner.Start("Loading WAF configuration...")
		if err := config.LoadConfig(); err != nil {
			spinner.Fail("Failed to load config: ", err.Error())
			return
		}
		waf.InitWAF()
		spinner.Stop()

		fmt.Println()
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println(Gold("■ AI-WAF ENGINE STATUS"))
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println()

		// Engine status
		if config.Cfg.WAF.Enabled {
			pterm.Success.Println("WAF Engine is ACTIVE")
		} else {
			pterm.Warning.Println("WAF Engine is DISABLED")
			pterm.Info.Println("Enable it in tkngate.yaml under 'waf.enabled: true'")
			return
		}

		// Rule counts
		fmt.Println()
		fmt.Printf("  Prompt Injection Signatures: %s\n", Gold(fmt.Sprintf("%d rules", len(waf.KnownPromptInjections))))
		fmt.Printf("  Custom Blocklist Regexes:    %s\n", Gold(fmt.Sprintf("%d rules", len(config.Cfg.WAF.Blocklist))))
		fmt.Printf("  PII Redaction Rules:         %s\n", Gold("10 categories"))
		fmt.Println()

		// PII categories
		piiCategories := []string{
			"JWT Tokens", "API Keys (OpenAI/Anthropic/etc)", "AWS Access Keys",
			"AWS Secret Keys", "GitHub Tokens", "Environment Secrets",
			"Email Addresses", "Phone Numbers", "Social Security Numbers", "Credit Cards",
		}
		fmt.Printf("  PII Categories Protected:    %s\n", Gold(fmt.Sprintf("%d", len(piiCategories))))
		for _, cat := range piiCategories {
			fmt.Printf("    %s %s\n", Forest("•"), cat)
		}
		fmt.Println()
	},
}

var wafRulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "List all active WAF detection rules",
	Run: func(cmd *cobra.Command, args []string) {
		spinner, _ := pterm.DefaultSpinner.Start("Loading WAF rules...")
		if err := config.LoadConfig(); err != nil {
			spinner.Fail("Failed to load config: ", err.Error())
			return
		}
		waf.InitWAF()
		spinner.Stop()

		if !config.Cfg.WAF.Enabled {
			pterm.Warning.Println("WAF Engine is DISABLED. No rules are active.")
			return
		}

		// ── Standard Heuristic Rules ──
		fmt.Println()
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println(Gold("■ STANDARD PROMPT INJECTION SIGNATURES"))
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println()

		tableData := pterm.TableData{
			{"#", "SIGNATURE"},
		}
		for i, sig := range waf.KnownPromptInjections {
			tableData = append(tableData, []string{
				Gold(fmt.Sprintf("%d", i+1)),
				string(sig),
			})
		}
		pterm.DefaultTable.WithHasHeader().WithHeaderRowSeparator("-").WithData(tableData).Render()

		// ── Custom Blocklist Rules ──
		fmt.Println()
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println(Gold("■ CUSTOM BLOCKLIST REGEXES (from tkngate.yaml)"))
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println()

		if len(config.Cfg.WAF.Blocklist) == 0 {
			pterm.Info.Println("No custom blocklist rules configured.")
		} else {
			customTable := pterm.TableData{
				{"#", "REGEX PATTERN"},
			}
			for i, pattern := range config.Cfg.WAF.Blocklist {
				customTable = append(customTable, []string{
					Gold(fmt.Sprintf("%d", i+1)),
					pattern,
				})
			}
			pterm.DefaultTable.WithHasHeader().WithHeaderRowSeparator("-").WithData(customTable).Render()
		}

		// ── PII Redaction Rules ──
		fmt.Println()
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println(Gold("■ PII REDACTION RULES (DLP Engine)"))
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println()

		piiTable := pterm.TableData{
			{"CATEGORY", "REDACTION MARKER"},
		}
		piiRulesList := []struct{ name, marker string }{
			{"JWT Tokens", "[REDACTED_JWT]"},
			{"API Keys", "[REDACTED_API_KEY]"},
			{"AWS Access Keys", "[REDACTED_AWS_KEY]"},
			{"AWS Secret Keys", "[REDACTED_AWS_SECRET]"},
			{"GitHub Tokens", "[REDACTED_GITHUB_TOKEN]"},
			{"Environment Secrets", "[REDACTED_ENV_SECRET]"},
			{"Email Addresses", "[REDACTED_EMAIL]"},
			{"Phone Numbers", "[REDACTED_PHONE]"},
			{"SSN", "[REDACTED_SSN]"},
			{"Credit Cards", "[REDACTED_CREDIT_CARD]"},
		}
		for _, rule := range piiRulesList {
			piiTable = append(piiTable, []string{
				rule.name,
				Gold(rule.marker),
			})
		}
		pterm.DefaultTable.WithHasHeader().WithHeaderRowSeparator("-").WithData(piiTable).Render()
		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(wafCmd)
	wafCmd.AddCommand(wafStatusCmd)
	wafCmd.AddCommand(wafRulesCmd)
}
