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
	SessionID        string    `json:"session_id,omitempty"`
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

type PoolNode struct {
	NodeID               string
	ProviderType         string
	BlindedKeyHash       string
	MeasuredTpmLimit     int
	RemainingTokensQuota int
}

// SessionState represents the persistent state of an autonomous agent session
type SessionState struct {
	SessionID         string     `json:"session_id"`
	AllocatedBudget   float64    `json:"allocated_budget_usd"`
	ConsumedBudget    float64    `json:"consumed_budget_usd"`
	CurrentState      BudgetZone `json:"current_state"`
	CreatedAt         time.Time  `json:"created_at"`
}
