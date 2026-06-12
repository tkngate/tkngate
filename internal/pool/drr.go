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
}

var GlobalDRR *DRREngine

func InitDRR() {
	GlobalDRR = &DRREngine{
		deficits:     make(map[string]float64),
		currentIndex: make(map[string]int),
	}
}

// GetNextKey rotates through donated keys for a provider using Deficit Round Robin.
func (d *DRREngine) GetNextKey(provider string, estimatedCost float64) (string, error) {
	if budget.GlobalLedger == nil {
		return "", fmt.Errorf("ledger not initialized")
	}

	nodes, err := budget.GlobalLedger.GetPoolNodes(provider)
	if err != nil || len(nodes) == 0 {
		return "", fmt.Errorf("no keys in pool")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	idx := d.currentIndex[provider]
	if idx >= len(nodes) {
		idx = 0
	}

	// Try to find a valid key
	for i := 0; i < len(nodes); i++ {
		node := nodes[idx]
		d.currentIndex[provider] = (idx + 1) % len(nodes)

		plaintextKey, err := crypto.Decrypt(node.BlindedKeyHash)
		if err == nil {
			return plaintextKey, nil
		}
		idx = d.currentIndex[provider]
	}

	return "", fmt.Errorf("failed to decrypt any pool keys")
}
