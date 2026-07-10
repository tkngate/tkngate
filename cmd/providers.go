package cmd

import (
	"fmt"
	"sort"
	"strings"
	"tkngate/internal/config"
	"tkngate/internal/validator"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var providersCmd = &cobra.Command{
	Use:   "providers",
	Short: "Manage and test upstream AI providers",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var providersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured upstream providers",
	Run: func(cmd *cobra.Command, args []string) {
		spinner, _ := pterm.DefaultSpinner.Start("Loading provider configuration...")
		if err := config.LoadConfig(); err != nil {
			spinner.Fail("Failed to load config: ", err.Error())
			return
		}
		spinner.Stop()

		fmt.Println()
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println(Gold("■ CONFIGURED UPSTREAM PROVIDERS"))
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println()

		if len(config.Cfg.Providers) == 0 {
			pterm.Warning.Println("No providers configured in tkngate.yaml.")
			return
		}

		// Sort provider names for stable output
		names := make([]string, 0, len(config.Cfg.Providers))
		for name := range config.Cfg.Providers {
			names = append(names, name)
		}
		sort.Strings(names)

		tableData := pterm.TableData{
			{"PROVIDER", "DEFAULT MODEL", "ENDPOINT", "KEY STATUS"},
		}

		for _, name := range names {
			p := config.Cfg.Providers[name]
			keyStatus := pterm.LightRed("NOT SET")
			if p.APIKey != "" {
				// Mask the key: show first 4 chars and last 4 chars
				masked := maskKey(p.APIKey)
				keyStatus = Forest(masked)
			}

			endpoint := p.BaseURL
			if endpoint == "" {
				endpoint = "(default)"
			}

			tableData = append(tableData, []string{
				Gold(name),
				p.DefaultModel,
				endpoint,
				keyStatus,
			})
		}

		pterm.DefaultTable.WithHasHeader().WithHeaderRowSeparator("-").WithData(tableData).Render()
		fmt.Println()
		pterm.Info.Printf("Total providers: %d\n", len(config.Cfg.Providers))
		fmt.Println()
	},
}

var providersTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test connectivity to all configured upstream providers",
	Run: func(cmd *cobra.Command, args []string) {
		spinner, _ := pterm.DefaultSpinner.Start("Loading provider configuration...")
		if err := config.LoadConfig(); err != nil {
			spinner.Fail("Failed to load config: ", err.Error())
			return
		}
		spinner.Stop()

		fmt.Println()
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println(Gold("■ UPSTREAM PROVIDER HEALTH CHECK"))
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println()

		if len(config.Cfg.Providers) == 0 {
			pterm.Warning.Println("No providers configured. Nothing to test.")
			return
		}

		// Sort provider names for stable output
		names := make([]string, 0, len(config.Cfg.Providers))
		for name := range config.Cfg.Providers {
			names = append(names, name)
		}
		sort.Strings(names)

		passed := 0
		failed := 0

		for _, name := range names {
			p := config.Cfg.Providers[name]
			testSpinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Testing %s...", name))

			if p.APIKey == "" && name != "ollama" {
				testSpinner.Warning(fmt.Sprintf("%s — SKIPPED (no API key configured)", name))
				continue
			}

			err := validator.ValidateKey(name, p.APIKey)
			if err != nil {
				testSpinner.Fail(fmt.Sprintf("%s — FAIL: %v", name, err))
				failed++
			} else {
				testSpinner.Success(fmt.Sprintf("%s — PASS ✓", name))
				passed++
			}
		}

		fmt.Println()
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Printf("  Results: %s passed, %s failed\n",
			Forest(fmt.Sprintf("%d", passed)),
			pterm.LightRed(fmt.Sprintf("%d", failed)))
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println()
	},
}

// maskKey shows the first 4 and last 4 characters of a key, masking the rest.
func maskKey(key string) string {
	if len(key) <= 8 {
		return strings.Repeat("•", len(key))
	}
	return key[:4] + strings.Repeat("•", len(key)-8) + key[len(key)-4:]
}

func init() {
	rootCmd.AddCommand(providersCmd)
	providersCmd.AddCommand(providersListCmd)
	providersCmd.AddCommand(providersTestCmd)
}
