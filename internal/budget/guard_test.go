package budget

import (
	_ "modernc.org/sqlite"
	"testing"
	"tkngate/internal/config"
	"tkngate/internal/logging"
)

// MockLedger implements Ledger interface just for testing guard logic.
// But we can just use the real SQLite ledger in memory!
func setupInMemoryLedger(t *testing.T) {
	err := InitMemoryLedger()
	if err != nil {
		t.Fatalf("Failed to init memory db: %v", err)
	}
}

func TestCheckBudget_Green(t *testing.T) {
	logging.InitLogger()
	setupInMemoryLedger(t)

	config.Cfg.Budget.GlobalLimitUSD = 100.0
	config.Cfg.Budget.AmberThresholdPct = 70
	config.Cfg.Budget.RedThresholdPct = 95

	// 0 spend
	status, err := CheckBudget()
	if err != nil {
		t.Fatalf("CheckBudget error: %v", err)
	}

	if status.Zone != ZoneGreen {
		t.Errorf("Expected ZoneGreen, got %s", status.Zone)
	}
}

func TestCheckBudget_AmberAndRed(t *testing.T) {
	logging.InitLogger()
	setupInMemoryLedger(t)

	config.Cfg.Budget.GlobalLimitUSD = 100.0
	config.Cfg.Budget.AmberThresholdPct = 70
	config.Cfg.Budget.RedThresholdPct = 95

	// Log a transaction of $75 to push it to Amber
	err := GlobalLedger.RecordTransaction(Transaction{
		SessionID:        "test-session",
		Provider:         "openai",
		Model:            "gpt-4",
		InputTokens:      1000,
		OutputTokens:     1000,
		EstimatedCostUSD: 75.0,
	})
	if err != nil {
		t.Fatalf("LogTransaction error: %v", err)
	}

	status, _ := CheckBudget()
	if status.Zone != ZoneAmber {
		t.Errorf("Expected ZoneAmber, got %s. Spent: %f", status.Zone, status.TotalSpentUSD)
	}

	// Log another $25 to push it to Red
	GlobalLedger.RecordTransaction(Transaction{
		SessionID:        "test-session",
		Provider:         "openai",
		Model:            "gpt-4",
		InputTokens:      1000,
		OutputTokens:     1000,
		EstimatedCostUSD: 25.0,
	})

	status, _ = CheckBudget()
	if status.Zone != ZoneRed {
		t.Errorf("Expected ZoneRed, got %s", status.Zone)
	}
}

func TestCheckSessionBudget(t *testing.T) {
	logging.InitLogger()
	setupInMemoryLedger(t)

	config.Cfg.Budget.MaxSessionCostUSD = 10.0
	config.Cfg.Budget.AmberThresholdPct = 70
	config.Cfg.Budget.RedThresholdPct = 95

	sessionID := "test-session"

	// $8 spend -> Amber
	GlobalLedger.RecordTransaction(Transaction{
		SessionID:        sessionID,
		Provider:         "openai",
		Model:            "gpt-4",
		InputTokens:      100,
		OutputTokens:     100,
		EstimatedCostUSD: 8.0,
	})

	status, err := CheckSessionBudget(sessionID)
	if err != nil {
		t.Fatalf("CheckSessionBudget error: %v", err)
	}

	if status.Zone != ZoneAmber {
		t.Errorf("Expected ZoneAmber, got %s", status.Zone)
	}
}
