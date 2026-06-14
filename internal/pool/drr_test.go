package pool

import (
	"testing"
	"tkngate/internal/budget"
)

func TestDRREngine_Init(t *testing.T) {
	InitDRR()
	if GlobalDRR == nil {
		t.Fatal("Expected GlobalDRR to be initialized")
	}

	if GlobalDRR.deficits == nil {
		t.Error("deficits map not initialized")
	}
}

func TestDRREngine_GetNextKey_NoLedger(t *testing.T) {
	InitDRR()
	budget.GlobalLedger = nil // Ensure ledger is nil
	
	_, err := GlobalDRR.GetNextKey("openai", "test-session", 100)
	if err == nil {
		t.Error("Expected error when ledger is not initialized")
	}
}

func TestDRREngine_FairnessEngine(t *testing.T) {
	InitDRR()
	budget.InitMemoryLedger()
	budget.GlobalLedger.AddPoolNode(budget.PoolNode{
		NodeID: "test-node",
		ProviderType: "openai",
		BlindedKeyHash: "somehash",
		MeasuredTpmLimit: 100,
		RemainingTokensQuota: 100,
	})
	
	sessionID := "greedy-user"
	GlobalDRR.sessionUsage[sessionID] = 10001 // Exceed 10,000 token public limit
	
	_, err := GlobalDRR.GetNextKey("openai", sessionID, 100)
	if err == nil {
		t.Error("Expected error from Fairness Engine for free-rider exceeding 10k tokens")
	}
	
	if err.Error() != "Fairness Engine: Token bucket exhausted for free-rider session "+sessionID+". Please donate keys to increase your priority limit" {
		t.Errorf("Unexpected error message: %v", err)
	}
}
