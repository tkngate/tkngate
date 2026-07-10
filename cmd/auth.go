package cmd

import (
	"fmt"
	"os"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"tkngate/internal/auth"
	"tkngate/internal/budget"
)

var (
	virtualKeyName     string
	virtualKeyLimit    float64
	virtualKeyOrgID    int
	virtualKeyProviders string
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage tkngate Virtual Keys and Authentication",
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new Virtual Key",
	Run: func(cmd *cobra.Command, args []string) {
		if virtualKeyName == "" {
			pterm.Error.Println("--name is required")
			os.Exit(1)
		}

		spinner, _ := pterm.DefaultSpinner.Start("Generating virtual key...")
		if err := budget.InitLedger(); err != nil {
			spinner.Fail("Error initializing ledger: ", err.Error())
			os.Exit(1)
		}

		key, err := auth.GenerateVirtualKey()
		if err != nil {
			spinner.Fail("Error generating key: ", err.Error())
			os.Exit(1)
		}

		if err := budget.GlobalLedger.RegisterVirtualKey(key.Hash, virtualKeyName, virtualKeyLimit, virtualKeyOrgID, virtualKeyProviders); err != nil {
			spinner.Fail("Error saving key to ledger: ", err.Error())
			os.Exit(1)
		}
		spinner.Success("Virtual Key Generated Successfully")

		fmt.Println()

		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println(Gold("■ NEW VIRTUAL KEY"))
		fmt.Printf("  Name:      %s\n", Gold(virtualKeyName))
		fmt.Printf("  Limit:     $%.2f\n", virtualKeyLimit)
		if virtualKeyOrgID > 0 {
			fmt.Printf("  Org ID:    %d\n", virtualKeyOrgID)
		}
		if virtualKeyProviders != "" {
			fmt.Printf("  Providers: %s\n", virtualKeyProviders)
		}
		fmt.Printf("  Key:       %s\n", Gold(key.Plaintext))
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		
		pterm.Warning.Println("Store this key safely. You will not be able to see it again!")
		pterm.Info.Println("Use this key as a Bearer token in your Authorization header.")
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all active Virtual Keys",
	Run: func(cmd *cobra.Command, args []string) {
		spinner, _ := pterm.DefaultSpinner.Start("Fetching virtual keys...")
		if err := budget.InitLedger(); err != nil {
			spinner.Fail("Error initializing ledger: ", err.Error())
			os.Exit(1)
		}

		keys, err := budget.GlobalLedger.GetVirtualKeys()
		if err != nil {
			spinner.Fail("Error fetching keys: ", err.Error())
			os.Exit(1)
		}
		spinner.Stop()

		fmt.Println()
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println(Gold("■ ACTIVE VIRTUAL KEYS"))
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println()

		if len(keys) == 0 {
			pterm.Info.Println("No active keys found. Use 'tkngate auth generate' to create one.")
			return
		}

		tableData := pterm.TableData{
			{"NAME", "CONSUMED", "ALLOCATED", "ORG ID", "RESTRICTED", "CREATED"},
		}

		for _, k := range keys {
			consumedStr := fmt.Sprintf("$%.2f", k.ConsumedBudget)
			allocatedStr := fmt.Sprintf("$%.2f", k.AllocatedBudget)

			// Highlight if consumed is close to allocated
			if k.ConsumedBudget >= k.AllocatedBudget {
				consumedStr = pterm.LightRed(consumedStr)
			} else if k.ConsumedBudget >= k.AllocatedBudget*0.75 {
				consumedStr = Gold(consumedStr)
			} else {
				consumedStr = Forest(consumedStr)
			}
			
			orgStr := "Global"
			if k.OrgID > 0 {
				orgStr = fmt.Sprintf("%d", k.OrgID)
			}
			
			restStr := "None"
			if k.AllowedProviders != "" {
				restStr = k.AllowedProviders
			}

			tableData = append(tableData, []string{
				Gold(k.Name),
				consumedStr,
				allocatedStr,
				orgStr,
				restStr,
				Parch(k.CreatedAt),
			})
		}

		pterm.DefaultTable.WithHasHeader().WithHeaderRowSeparator("-").WithData(tableData).Render()
		fmt.Println()
	},
}

var revokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "Revoke a Virtual Key by name",
	Run: func(cmd *cobra.Command, args []string) {
		if virtualKeyName == "" {
			pterm.Error.Println("--name is required")
			os.Exit(1)
		}

		spinner, _ := pterm.DefaultSpinner.Start("Revoking virtual key...")
		if err := budget.InitLedger(); err != nil {
			spinner.Fail("Error initializing ledger: ", err.Error())
			os.Exit(1)
		}

		err := budget.GlobalLedger.RevokeVirtualKey(virtualKeyName)
		if err != nil {
			if err.Error() == "sql: no rows in result set" {
				spinner.Fail(fmt.Sprintf("Virtual Key '%s' not found", virtualKeyName))
			} else {
				spinner.Fail("Error revoking key: ", err.Error())
			}
			os.Exit(1)
		}

		spinner.Success(fmt.Sprintf("Successfully revoked Virtual Key: %s", virtualKeyName))
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(generateCmd)
	authCmd.AddCommand(listCmd)
	authCmd.AddCommand(revokeCmd)

	generateCmd.Flags().StringVar(&virtualKeyName, "name", "", "Name of the key (e.g. 'agent-1')")
	generateCmd.Flags().Float64Var(&virtualKeyLimit, "limit", 10.0, "Budget limit in USD for this key")
	generateCmd.Flags().IntVar(&virtualKeyOrgID, "org", 0, "Organization ID to assign this key to")
	generateCmd.Flags().StringVar(&virtualKeyProviders, "providers", "", "Comma-separated list of allowed providers (e.g. 'openai,anthropic')")

	revokeCmd.Flags().StringVar(&virtualKeyName, "name", "", "Name of the key to revoke")
}
