package cmd

import (
	"fmt"
	"tkngate/internal/budget"

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
		if err := budget.InitLedger(); err != nil {
			fmt.Println("Failed to open ledger:", err)
			return
		}
		spent, err := budget.GlobalLedger.GetTotalSpend()
		if err != nil {
			fmt.Println("Failed to get spend:", err)
			return
		}
		fmt.Printf("💰 Total API Spend: $%.5f\n", spent)
	},
}

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Manually resets the budget ledger",
	Run: func(cmd *cobra.Command, args []string) {
		if err := budget.InitLedger(); err != nil {
			fmt.Println("Failed to open ledger:", err)
			return
		}
		if err := budget.GlobalLedger.Reset(); err != nil {
			fmt.Println("Failed to reset ledger:", err)
			return
		}
		fmt.Println("🗑️ Budget ledger reset successfully.")
	},
}

func init() {
	rootCmd.AddCommand(budgetCmd)
	budgetCmd.AddCommand(statusCmd)
	budgetCmd.AddCommand(resetCmd)
}
