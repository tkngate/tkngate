package cmd

import (
	"fmt"
	"tkngate/internal/budget"
	"tkngate/internal/crypto"
	"tkngate/internal/validator"

	"github.com/google/uuid"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	poolProvider string
	poolKey      string
	poolLimit    int
)

var poolCmd = &cobra.Command{
	Use:   "pool",
	Short: "Manage the local P2P token donation pool",
}

var donateCmd = &cobra.Command{
	Use:   "donate",
	Short: "Donate an API key to the local pool",
	RunE: func(cmd *cobra.Command, args []string) error {
		if poolKey == "" {
			// Prompt securely with masked input to prevent shell history leakage
			key, _ := pterm.DefaultInteractiveTextInput.WithMask("*").Show("Enter the API key to donate (input is hidden)")
			if key == "" {
				return fmt.Errorf("must provide an API key via prompt or --key flag")
			}
			poolKey = key
		}

		spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Validating %s API key...", poolProvider))
		if err := validator.ValidateKey(poolProvider, poolKey); err != nil {
			spinner.Fail(fmt.Sprintf("Validation Failed: %v\nPlease provide a real, active API key.", err))
			return nil // Return nil since we already printed the error beautifully
		}
		spinner.Success("Key is valid!")

		if err := ensureCryptoInitialized(); err != nil {
			pterm.Error.Println(fmt.Sprintf("crypto init failed: %v", err))
			return nil
		}

		spinner, _ = pterm.DefaultSpinner.Start("Encrypting key and saving to pool...")

		if err := budget.InitLedger(); err != nil {
			spinner.Fail(fmt.Sprintf("ledger init failed: %v", err))
			return nil
		}

		encryptedKey, err := crypto.Encrypt(poolKey)
		if err != nil {
			spinner.Fail(fmt.Sprintf("encryption failed: %v", err))
			return nil
		}

		node := budget.PoolNode{
			NodeID:               uuid.New().String(),
			ProviderType:         poolProvider,
			BlindedKeyHash:       encryptedKey,
			MeasuredTpmLimit:     poolLimit,
			RemainingTokensQuota: poolLimit,
		}

		if err := budget.GlobalLedger.AddPoolNode(node); err != nil {
			spinner.Fail(fmt.Sprintf("failed to save node: %v", err))
			return nil
		}
		spinner.Success(fmt.Sprintf("Successfully donated %s key to the local DRR pool!", poolProvider))

		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println(Gold("■ DONATION SUCCESSFUL"))
		fmt.Printf("  Provider:   %s\n", Gold(poolProvider))
		fmt.Printf("  Node ID:    %s\n", Gold(node.NodeID))
		fmt.Printf("  TPM Quota:  %d\n", poolLimit)
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println()
		return nil
	},
}

var poolStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of the local donation pool",
	RunE: func(cmd *cobra.Command, args []string) error {
		spinner, _ := pterm.DefaultSpinner.Start("Fetching pool nodes...")
		if err := budget.InitLedger(); err != nil {
			spinner.Fail(fmt.Sprintf("ledger init failed: %v", err))
			return nil
		}

		nodes, err := budget.GlobalLedger.GetPoolNodes(poolProvider)
		if err != nil {
			spinner.Fail(fmt.Sprintf("Failed to get pool nodes: %v", err))
			return nil
		}
		spinner.Stop()

		fmt.Println()
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Printf("%s\n", Gold(fmt.Sprintf("■ TOKEN POOL STATUS [%s]", poolProvider)))
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println()

		pterm.Info.Printf("Total Donated Keys: %d\n\n", len(nodes))

		if len(nodes) == 0 {
			pterm.Warning.Println("No nodes found for this provider.")
			return nil
		}

		tableData := pterm.TableData{
			{"NODE ID", "QUOTA (TPM)"},
		}

		for _, node := range nodes {
			tableData = append(tableData, []string{
				Gold(node.NodeID),
				Gold(fmt.Sprintf("%d", node.MeasuredTpmLimit)),
			})
		}

		pterm.DefaultTable.WithHasHeader().WithHeaderRowSeparator("-").WithData(tableData).Render()
		fmt.Println()
		return nil
	},
}

func init() {
	donateCmd.Flags().StringVar(&poolProvider, "provider", "openai", "Provider (e.g. openai, anthropic)")
	donateCmd.Flags().StringVar(&poolKey, "key", "", "The actual API key to encrypt and donate")
	donateCmd.Flags().IntVar(&poolLimit, "limit", 100000, "TPM token limit for this key")

	poolStatusCmd.Flags().StringVar(&poolProvider, "provider", "openai", "Provider (e.g. openai, anthropic)")

	poolCmd.AddCommand(donateCmd)
	poolCmd.AddCommand(poolStatusCmd)
	rootCmd.AddCommand(poolCmd)
}
