package budget

import (
	"fmt"
	"tkngate/internal/config"
	"tkngate/internal/logging"
)

// CheckBudget compares the current spend against the config limits
func CheckBudget() (BudgetStatus, error) {
	if GlobalLedger == nil {
		return BudgetStatus{}, fmt.Errorf("ledger not initialized")
	}

	spent, err := GlobalLedger.GetTotalSpend()
	if err != nil {
		return BudgetStatus{}, err
	}

	limit := config.Cfg.Budget.GlobalLimitUSD
	if limit == 0 {
		limit = 999999.0 // Unbounded if not set
	}

	amberLimit := limit * (float64(config.Cfg.Budget.AmberThresholdPct) / 100.0)
	redLimit := limit * (float64(config.Cfg.Budget.RedThresholdPct) / 100.0)

	zone := ZoneGreen
	if spent >= redLimit {
		zone = ZoneRed
	} else if spent >= amberLimit {
		zone = ZoneAmber
	}

	status := BudgetStatus{
		TotalSpentUSD: spent,
		LimitUSD:      limit,
		RemainingUSD:  limit - spent,
		Zone:          zone,
	}

	// Log warnings if we shift zones
	switch zone {
	case ZoneAmber:
		logging.Logger.Warn("budget amber zone reached", "spent", spent, "limit", limit)
	case ZoneRed:
		logging.Logger.Error("budget red zone reached - circuit breaker tripped", "spent", spent, "limit", limit)
	}

	return status, nil
}

// CheckSessionBudget compares the current spend of a specific session against the config limits
func CheckSessionBudget(sessionID string) (BudgetStatus, error) {
	if GlobalLedger == nil {
		return BudgetStatus{}, fmt.Errorf("ledger not initialized")
	}

	limit := config.Cfg.Budget.MaxSessionCostUSD
	if limit == 0 {
		limit = 5.0 // default fallback
	}
	
	if err := GlobalLedger.EnsureSession(sessionID, limit); err != nil {
		return BudgetStatus{}, err
	}

	spent, err := GlobalLedger.GetSessionSpend(sessionID)
	if err != nil {
		return BudgetStatus{}, err
	}

	amberLimit := limit * (float64(config.Cfg.Budget.AmberThresholdPct) / 100.0)
	redLimit := limit * (float64(config.Cfg.Budget.RedThresholdPct) / 100.0)

	zone := ZoneGreen
	if spent >= redLimit {
		zone = ZoneRed
	} else if spent >= amberLimit {
		zone = ZoneAmber
	}

	status := BudgetStatus{
		TotalSpentUSD: spent,
		LimitUSD:      limit,
		RemainingUSD:  limit - spent,
		Zone:          zone,
	}

	switch zone {
	case ZoneAmber:
		logging.Logger.Warn("session budget amber zone reached", "session", sessionID, "spent", spent, "limit", limit)
	case ZoneRed:
		logging.Logger.Error("session budget red zone reached - circuit breaker tripped", "session", sessionID, "spent", spent, "limit", limit)
	}

	return status, nil
}
