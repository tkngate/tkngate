package pool

import (
	"fmt"
	"sync"
	"tkngate/internal/budget"
	"tkngate/internal/crypto"
)

type DRREngine struct {
	mu           sync.Mutex
	deficits     map[string]float64
	currentIndex map[string]int
	sessionUsage map[string]int // tracks free tokens consumed by sessionID
}

var GlobalDRR *DRREngine

func InitDRR() {
	GlobalDRR = &DRREngine{
		deficits:     make(map[string]float64),
		currentIndex: make(map[string]int),
		sessionUsage: make(map[string]int),
	}
}

// GetNextKey rotates through donated keys for a provider using Deficit Round Robin.
// Includes the BitTorrent Fairness Engine: blocks free-riders from exceeding the public token bucket quota.
func (d *DRREngine) GetNextKey(provider string, sessionID string, estimatedTokens int) (string, error) {
	if budget.GlobalLedger == nil {
		return "", fmt.Errorf("ledger not initialized")
	}

	nodes, err := budget.GlobalLedger.GetPoolNodes(provider)
	if err != nil || len(nodes) == 0 {
		return "", fmt.Errorf("no keys in pool")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// 1. BitTorrent Fairness Engine (Free-Rider Limit)
	// If the session has consumed more than 10,000 free tokens from the public pool, block them.
	// In a real production deployment, we would check if they are a "donor" to increase this quota.
	if d.sessionUsage[sessionID] > 10000 {
		return "", fmt.Errorf("Fairness Engine: Token bucket exhausted for free-rider session %s. Please donate keys to increase your priority limit", sessionID)
	}

	idx := d.currentIndex[provider]
	if idx >= len(nodes) {
		idx = 0
	}

	// Try to find a valid key with remaining quota
	for i := 0; i < len(nodes); i++ {
		node := nodes[idx]
		d.currentIndex[provider] = (idx + 1) % len(nodes)

		// Skip exhausted keys
		if node.RemainingTokensQuota <= 0 {
			idx = d.currentIndex[provider]
			continue
		}

		plaintextKey, err := crypto.Decrypt(node.BlindedKeyHash)
		if err == nil {
			// Decrement quota from the pool key
			budget.GlobalLedger.DecrementPoolQuota(node.NodeID, estimatedTokens)
			
			// Increment consumption for the free-rider session
			if sessionID != "" {
				d.sessionUsage[sessionID] += estimatedTokens
			}
			
			return plaintextKey, nil
		}
		idx = d.currentIndex[provider]
	}

	return "", fmt.Errorf("failed to decrypt any pool keys")
}
