package budget

import "time"

// BudgetZone represents the state of the budget (Traffic Light system)
type BudgetZone string

const (
	ZoneGreen BudgetZone = "GREEN"
	ZoneAmber BudgetZone = "AMBER"
	ZoneRed   BudgetZone = "RED"
)

// Transaction represents a single API request/response cycle
type Transaction struct {
	ID               int64     `json:"id"`
	Provider         string    `json:"provider"`
	Model            string    `json:"model"`
	InputTokens      int       `json:"input_tokens"`
	OutputTokens     int       `json:"output_tokens"`
	EstimatedCostUSD float64   `json:"estimated_cost_usd"`
	Timestamp        time.Time `json:"timestamp"`
}

// BudgetStatus contains the current view of the budget
type BudgetStatus struct {
	TotalSpentUSD float64    `json:"total_spent_usd"`
	LimitUSD      float64    `json:"limit_usd"`
	RemainingUSD  float64    `json:"remaining_usd"`
	Zone          BudgetZone `json:"zone"`
}
