package cmd

import (
	"fmt"
	"os"
	"tkngate/internal/budget"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	orgName  string
	orgLimit float64
)

var orgCmd = &cobra.Command{
	Use:   "org",
	Short: "Manage Multi-Tenant Organizations",
}

var orgCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new Organization",
	Run: func(cmd *cobra.Command, args []string) {
		if orgName == "" {
			pterm.Error.Println("--name is required")
			os.Exit(1)
		}

		spinner, _ := pterm.DefaultSpinner.Start("Creating organization...")
		if err := budget.InitLedger(); err != nil {
			spinner.Fail("Error initializing ledger: ", err.Error())
			os.Exit(1)
		}

		if err := budget.GlobalLedger.CreateOrganization(orgName, orgLimit); err != nil {
			spinner.Fail("Error creating org: ", err.Error())
			os.Exit(1)
		}

		spinner.Success("Organization Created Successfully")
		fmt.Printf("Org Name: %s | Limit: $%.2f\n", orgName, orgLimit)
	},
}

var orgListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all Organizations",
	Run: func(cmd *cobra.Command, args []string) {
		spinner, _ := pterm.DefaultSpinner.Start("Fetching organizations...")
		if err := budget.InitLedger(); err != nil {
			spinner.Fail("Error initializing ledger: ", err.Error())
			os.Exit(1)
		}

		orgs, err := budget.GlobalLedger.GetOrganizations()
		if err != nil {
			spinner.Fail("Error fetching orgs: ", err.Error())
			os.Exit(1)
		}
		spinner.Stop()

		fmt.Println()
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println(Gold("■ ORGANIZATIONS"))
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println()

		if len(orgs) == 0 {
			pterm.Info.Println("No organizations found. Use 'tkngate org create' to make one.")
			return
		}

		tableData := pterm.TableData{
			{"ID", "NAME", "CONSUMED", "LIMIT", "CREATED"},
		}

		for _, o := range orgs {
			consumedStr := fmt.Sprintf("$%.2f", o.ConsumedUSD)
			limitStr := fmt.Sprintf("$%.2f", o.BudgetLimitUSD)

			if o.ConsumedUSD >= o.BudgetLimitUSD {
				consumedStr = pterm.LightRed(consumedStr)
			} else if o.ConsumedUSD >= o.BudgetLimitUSD*0.75 {
				consumedStr = Gold(consumedStr)
			} else {
				consumedStr = Forest(consumedStr)
			}

			tableData = append(tableData, []string{
				fmt.Sprintf("%d", o.ID),
				Gold(o.Name),
				consumedStr,
				limitStr,
				Parch(o.CreatedAt),
			})
		}

		pterm.DefaultTable.WithHasHeader().WithHeaderRowSeparator("-").WithData(tableData).Render()
		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(orgCmd)
	orgCmd.AddCommand(orgCreateCmd)
	orgCmd.AddCommand(orgListCmd)

	orgCreateCmd.Flags().StringVar(&orgName, "name", "", "Name of the organization")
	orgCreateCmd.Flags().Float64Var(&orgLimit, "limit", 100.0, "Budget limit in USD")
}
