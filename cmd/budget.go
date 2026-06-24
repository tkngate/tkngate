package cmd

import (
	"fmt"
	"tkngate/internal/budget"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var budgetCmd = &cobra.Command{
	Use:   "budget",
	Short: "Budget related commands",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Shows current spend vs. limits for each provider",
	Run: func(cmd *cobra.Command, args []string) {
		spinner, _ := pterm.DefaultSpinner.Start("Fetching budget data...")
		if err := budget.InitLedger(); err != nil {
			spinner.Fail("Failed to open ledger: ", err.Error())
			return
		}
		spent, err := budget.GlobalLedger.GetTotalSpend()
		if err != nil {
			spinner.Fail("Failed to get spend: ", err.Error())
			return
		}
		spinner.Stop()

		fmt.Println()
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println(Gold("■ BUDGET LEDGER"))
		fmt.Printf("  Total API Spend: %s\n", Gold(fmt.Sprintf("$%.5f", spent)))
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println()
	},
}

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Manually resets the budget ledger",
	Run: func(cmd *cobra.Command, args []string) {
		spinner, _ := pterm.DefaultSpinner.Start("Resetting budget ledger...")
		if err := budget.InitLedger(); err != nil {
			spinner.Fail("Failed to open ledger: ", err.Error())
			return
		}
		if err := budget.GlobalLedger.Reset(); err != nil {
			spinner.Fail("Failed to reset ledger: ", err.Error())
			return
		}
		spinner.Success("Budget ledger reset successfully.")
	},
}

func init() {
	rootCmd.AddCommand(budgetCmd)
	budgetCmd.AddCommand(statusCmd)
	budgetCmd.AddCommand(resetCmd)
}
