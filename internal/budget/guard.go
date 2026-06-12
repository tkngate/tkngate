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
	if zone == ZoneAmber {
		logging.Logger.Warn("budget amber zone reached", "spent", spent, "limit", limit)
	} else if zone == ZoneRed {
		logging.Logger.Error("budget red zone reached - circuit breaker tripped", "spent", spent, "limit", limit)
	}

	return status, nil
}
