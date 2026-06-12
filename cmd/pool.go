package cmd

import (
	"fmt"
	"tkngate/internal/budget"
	"tkngate/internal/crypto"
	"tkngate/internal/validator"

	"github.com/google/uuid"
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
			return fmt.Errorf("must provide --key")
		}

		fmt.Printf("Validating %s API key...\n", poolProvider)
		if err := validator.ValidateKey(poolProvider, poolKey); err != nil {
			return fmt.Errorf("\n❌ Validation Failed: %v\nPlease provide a real, active API key.", err)
		}
		fmt.Printf("✅ Key is valid!\n")

		if err := crypto.InitCrypto(); err != nil {
			return fmt.Errorf("crypto init failed: %v", err)
		}

		if err := budget.InitLedger(); err != nil {
			return fmt.Errorf("ledger init failed: %v", err)
		}

		encryptedKey, err := crypto.Encrypt(poolKey)
		if err != nil {
			return fmt.Errorf("encryption failed: %v", err)
		}

		node := budget.PoolNode{
			NodeID:               uuid.New().String(),
			ProviderType:         poolProvider,
			BlindedKeyHash:       encryptedKey,
			MeasuredTpmLimit:     poolLimit,
			RemainingTokensQuota: poolLimit,
		}

		if err := budget.GlobalLedger.AddPoolNode(node); err != nil {
			return fmt.Errorf("failed to save node: %v", err)
		}

		fmt.Printf("Successfully donated %s key to the local DRR pool!\n", poolProvider)
		fmt.Printf("Encrypted Node ID: %s\n", node.NodeID)
		return nil
	},
}

var poolStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of the local donation pool",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := budget.InitLedger(); err != nil {
			return fmt.Errorf("ledger init failed: %v", err)
		}

		nodes, err := budget.GlobalLedger.GetPoolNodes(poolProvider)
		if err != nil {
			return err
		}

		fmt.Printf("Token Pool Status [%s]\n", poolProvider)
		fmt.Printf("Total Donated Keys: %d\n", len(nodes))
		for _, node := range nodes {
			fmt.Printf("- Node %s: Quota %d TPM\n", node.NodeID, node.MeasuredTpmLimit)
		}
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
