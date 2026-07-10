package pool

import (
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"tkngate/internal/budget"
	"tkngate/internal/config"
	"tkngate/internal/crypto"
	"tkngate/internal/mesh"
	"tkngate/internal/telemetry"
	"tkngate/internal/zkp"

	"github.com/consensys/gnark/backend/groth16"
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

// VerifyAndRoute is the ZK-SNARK-aware entry point for the mesh.
// If StrictZKPMode is enabled in config, this method verifies the
// zero-knowledge proof before allowing key routing. If disabled,
// it falls through to the standard GetNextKey.
func (d *DRREngine) VerifyAndRoute(provider, sessionID string, estimatedTokens int, proof groth16.Proof, nonce *big.Int) (string, error) {
	if config.Cfg.Mesh.StrictZKPMode {
		if zkp.GlobalZKP == nil {
			telemetry.ZkpFailedTotal.Inc()
			atomic.AddInt64(&telemetry.RawZkpFailed, 1)
			return "", fmt.Errorf("ZKP engine not initialized but strict_zkp_mode is enabled")
		}
		if proof == nil {
			telemetry.ZkpFailedTotal.Inc()
			atomic.AddInt64(&telemetry.RawZkpFailed, 1)
			return "", fmt.Errorf("ZKP: strict mode enabled — proof required but not provided")
		}
		if err := zkp.GlobalZKP.VerifyProof(proof, nonce); err != nil {
			telemetry.ZkpFailedTotal.Inc()
			atomic.AddInt64(&telemetry.RawZkpFailed, 1)
			// Slash the sender's reputation for submitting an invalid proof.
			if mesh.GlobalReputation != nil && sessionID != "" {
				mesh.GlobalReputation.Slash(sessionID, "invalid ZKP proof")
			}
			return "", fmt.Errorf("ZKP: proof verification failed for session %s — request blocked", sessionID)
		}
		telemetry.ZkpVerifiedTotal.Inc()
		atomic.AddInt64(&telemetry.RawZkpVerified, 1)
	}

	return d.GetNextKey(provider, sessionID, estimatedTokens)
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
	if d.sessionUsage[sessionID] > config.Cfg.Mesh.FreeRiderLimit {
		return "", fmt.Errorf("Fairness Engine: Token bucket exhausted for free-rider session %s. Please donate keys to increase your priority limit", sessionID)
	}

	// 2. Mesh Reputation System (v1.6.0)
	if mesh.GlobalReputation != nil {
		if mesh.GlobalReputation.IsBlacklisted(sessionID) {
			return "", fmt.Errorf("node %s is blacklisted from the mesh due to fraud", sessionID)
		}

		tier := mesh.GlobalReputation.GetTier(sessionID)
		if tier == mesh.TierUntrusted {
			return "", fmt.Errorf("node %s is untrusted", sessionID)
		}
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
			// 3. Mesh Premium Gate: Prevent NEW/TRUSTED from draining PREMIUM keys
			if mesh.GlobalReputation != nil && sessionID != "" {
				tier := mesh.GlobalReputation.GetTier(sessionID)
				
				// A simple heuristic: if a node has > 10M TPM, we consider it a "premium" key.
				// Untrusted users shouldn't route through enterprise-grade keys.
				isPremiumKey := node.MeasuredTpmLimit > 10000000
				
				if isPremiumKey && tier != mesh.TierPremium {
					// Skip this key, the requester doesn't have enough trust
					idx = d.currentIndex[provider]
					continue
				}
			}

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

