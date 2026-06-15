package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"tkngate/internal/auth"
	"tkngate/internal/budget"
)

var (
	virtualKeyName  string
	virtualKeyLimit float64
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
			color.Red("Error: --name is required")
			os.Exit(1)
		}

		if err := budget.InitLedger(); err != nil {
			color.Red("Error initializing ledger: %v", err)
			os.Exit(1)
		}

		key, err := auth.GenerateVirtualKey()
		if err != nil {
			color.Red("Error generating key: %v", err)
			os.Exit(1)
		}

		if err := budget.GlobalLedger.RegisterVirtualKey(key.Hash, virtualKeyName, virtualKeyLimit); err != nil {
			color.Red("Error saving key to ledger: %v", err)
			os.Exit(1)
		}

		fmt.Println()
		color.Green("=== Virtual Key Generated Successfully ===")
		fmt.Printf("Name:   %s\n", virtualKeyName)
		fmt.Printf("Limit:  $%.2f\n", virtualKeyLimit)
		fmt.Printf("Key:    %s\n\n", color.CyanString(key.Plaintext))
		
		color.Yellow("IMPORTANT: Store this key safely. You will not be able to see it again!")
		color.White("Use this key as a Bearer token in your Authorization header.")
		fmt.Println()
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all active Virtual Keys",
	Run: func(cmd *cobra.Command, args []string) {
		if err := budget.InitLedger(); err != nil {
			color.Red("Error initializing ledger: %v", err)
			os.Exit(1)
		}

		keys, err := budget.GlobalLedger.GetVirtualKeys()
		if err != nil {
			color.Red("Error fetching keys: %v", err)
			os.Exit(1)
		}

		fmt.Println()
		color.HiMagenta("  ✦ ACTIVE VIRTUAL KEYS ✦")
		fmt.Println()
		if len(keys) == 0 {
			color.White("  No active keys found. Use 'tkngate auth issue' to create one.")
			fmt.Println()
			return
		}
		
		fmt.Printf("  %-25s %-20s %-20s %s\n", "NAME", "CONSUMED", "ALLOCATED", "CREATED")
		fmt.Println(color.HiBlackString("  ────────────────────────────────────────────────────────────────────────────────────────"))
		for _, k := range keys {
			consumedStr := fmt.Sprintf("$%.2f", k.ConsumedBudget)
			allocatedStr := fmt.Sprintf("$%.2f", k.AllocatedBudget)
			
			// Highlight if consumed is close to allocated
			if k.ConsumedBudget >= k.AllocatedBudget {
				consumedStr = color.RedString(consumedStr)
			} else if k.ConsumedBudget >= k.AllocatedBudget*0.75 {
				consumedStr = color.YellowString(consumedStr)
			} else {
				consumedStr = color.GreenString(consumedStr)
			}
			
			fmt.Printf("  %-25s %-20s %-20s %s\n", color.CyanString(k.Name), consumedStr, allocatedStr, color.HiBlackString(k.CreatedAt))
		}
		fmt.Println(color.HiBlackString("  ────────────────────────────────────────────────────────────────────────────────────────"))
		fmt.Println()
	},
}

var revokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "Revoke a Virtual Key by name",
	Run: func(cmd *cobra.Command, args []string) {
		if virtualKeyName == "" {
			color.Red("Error: --name is required")
			os.Exit(1)
		}

		if err := budget.InitLedger(); err != nil {
			color.Red("Error initializing ledger: %v", err)
			os.Exit(1)
		}

		err := budget.GlobalLedger.RevokeVirtualKey(virtualKeyName)
		if err != nil {
			if err.Error() == "sql: no rows in result set" {
				color.Red("Error: Virtual Key '%s' not found", virtualKeyName)
			} else {
				color.Red("Error revoking key: %v", err)
			}
			os.Exit(1)
		}

		color.Green("Successfully revoked Virtual Key: %s", virtualKeyName)
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(generateCmd)
	authCmd.AddCommand(listCmd)
	authCmd.AddCommand(revokeCmd)

	generateCmd.Flags().StringVar(&virtualKeyName, "name", "", "Name of the key (e.g. 'agent-1')")
	generateCmd.Flags().Float64Var(&virtualKeyLimit, "limit", 10.0, "Budget limit in USD for this key")
	
	revokeCmd.Flags().StringVar(&virtualKeyName, "name", "", "Name of the key to revoke")
}
