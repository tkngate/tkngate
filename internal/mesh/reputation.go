package mesh

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"tkngate/internal/config"
	"tkngate/internal/logging"
)

// TrustTier classifies nodes into access levels for key routing.
type TrustTier string

const (
	TierUntrusted TrustTier = "UNTRUSTED" // Blacklisted — no access
	TierNew       TrustTier = "NEW"       // Fresh node — cheap keys only
	TierTrusted   TrustTier = "TRUSTED"   // Established — standard keys
	TierPremium   TrustTier = "PREMIUM"   // High-trust — premium GPT-4/Opus keys
)

// NodeReputation tracks a mesh participant's trust state.
type NodeReputation struct {
	NodeID        string    `json:"node_id"`
	TrustScore    float64   `json:"trust_score"`
	TotalRequests int       `json:"total_requests"`
	Violations    int       `json:"violations"`
	Tier          TrustTier `json:"tier"`
	Blacklisted   bool      `json:"blacklisted"`
	LastActivity  time.Time `json:"last_activity"`
}

// ReputationManager maintains trust scores for all mesh participants.
type ReputationManager struct {
	mu    sync.RWMutex
	db    *sql.DB
	nodes map[string]*NodeReputation

	OnReputationChange func(nodeID string, change float64, reason string)
}

var GlobalReputation *ReputationManager

// InitReputation creates the reputation manager. If db is nil, runs in-memory only.
func InitReputation(db *sql.DB) error {
	mgr := &ReputationManager{
		nodes: make(map[string]*NodeReputation),
		db:    db,
	}

	if db != nil {
		query := `
		CREATE TABLE IF NOT EXISTS mesh_reputation (
			node_id TEXT PRIMARY KEY,
			trust_score REAL DEFAULT 50.0,
			total_requests INTEGER DEFAULT 0,
			violations INTEGER DEFAULT 0,
			blacklisted INTEGER DEFAULT 0,
			last_activity DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS mesh_fraud_proofs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			offender_node_id TEXT NOT NULL,
			victim_node_id TEXT NOT NULL,
			evidence TEXT NOT NULL,
			slash_amount REAL NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		`
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to create mesh reputation tables: %w", err)
		}

		// Load existing reputations from DB
		rows, err := db.Query(`SELECT node_id, trust_score, total_requests, violations, blacklisted, last_activity FROM mesh_reputation`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var n NodeReputation
				var blacklisted int
				var lastAct string
				if err := rows.Scan(&n.NodeID, &n.TrustScore, &n.TotalRequests, &n.Violations, &blacklisted, &lastAct); err == nil {
					n.Blacklisted = blacklisted == 1
					if parsedTime, err := time.Parse("2006-01-02 15:04:05", lastAct); err == nil {
						n.LastActivity = parsedTime
					} else {
						// fallback for RFC3339 or other formats
						if parsedTime, err := time.Parse(time.RFC3339, lastAct); err == nil {
							n.LastActivity = parsedTime
						}
					}
					n.Tier = computeTier(n.TrustScore, n.Blacklisted)
					mgr.nodes[n.NodeID] = &n
				} else {
					logging.Logger.Error("Failed to scan mesh_reputation row", "err", err)
				}
			}
		}
	}

	GlobalReputation = mgr
	logging.Logger.Info("Mesh Reputation System initialised", "loaded_nodes", len(mgr.nodes))

	// Start decay loop
	go mgr.StartDecayLoop()

	return nil
}

// StartDecayLoop penalizes idle nodes to prevent passive Sybil armies.
func (m *ReputationManager) StartDecayLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for id, rep := range m.nodes {
			// If inactive for > 24 hours
			if now.Sub(rep.LastActivity) > 24*time.Hour {
				if rep.TrustScore > 0 && !rep.Blacklisted {
					// Decay by 0.5 per check (so ~0.5 per minute after 24h, this is aggressive but good for demo)
					rep.TrustScore -= 0.5
					if rep.TrustScore < 0 {
						rep.TrustScore = 0
					}
					rep.Tier = computeTier(rep.TrustScore, rep.Blacklisted)
					
					if m.db != nil {
						m.db.Exec(`UPDATE mesh_reputation SET trust_score = ?, last_activity = ? WHERE node_id = ?`,
							rep.TrustScore, rep.LastActivity, id) // we don't update last_activity to now, it stays old
					}
				}
			}
		}
		m.mu.Unlock()
	}
}

// GetOrCreate returns the reputation for a node, creating a fresh entry if new.
func (m *ReputationManager) GetOrCreate(nodeID string) *NodeReputation {
	m.mu.Lock()
	defer m.mu.Unlock()

	if rep, ok := m.nodes[nodeID]; ok {
		return rep
	}

	// New node joins the mesh
	rep := &NodeReputation{
		NodeID:        nodeID,
		TrustScore:    config.Cfg.Mesh.InitialTrustScore,
		TotalRequests: 0,
		Violations:    0,
		Tier:          TierNew,
		Blacklisted:   false,
		LastActivity:  time.Now(),
	}
	m.nodes[nodeID] = rep

	// Persist if DB is available
	if m.db != nil {
		m.db.Exec(`INSERT OR IGNORE INTO mesh_reputation (node_id, trust_score) VALUES (?, ?)`,
			nodeID, rep.TrustScore)
	}

	logging.Logger.Info("New mesh node registered", "node_id", nodeID, "initial_trust", rep.TrustScore)
	return rep
}

// RecordSuccess increments trust for a node after a clean request.
func (m *ReputationManager) RecordSuccess(nodeID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rep, ok := m.nodes[nodeID]
	if !ok {
		return
	}

	rep.TotalRequests++
	rep.LastActivity = time.Now()

	// Gradual trust increase: +0.1 per clean request, capped at 100
	if rep.TrustScore < 100.0 {
		rep.TrustScore += 0.1
		if rep.TrustScore > 100.0 {
			rep.TrustScore = 100.0
		}
	}
	rep.Tier = computeTier(rep.TrustScore, rep.Blacklisted)

	if m.db != nil {
		m.db.Exec(`UPDATE mesh_reputation SET trust_score = ?, total_requests = ?, last_activity = ? WHERE node_id = ?`,
			rep.TrustScore, rep.TotalRequests, rep.LastActivity, nodeID)
	}

	if m.OnReputationChange != nil {
		// Run in goroutine to prevent blocking
		go m.OnReputationChange(nodeID, 0.1, "Clean request")
	}
}

// Slash penalises a node for a ToS violation. Returns true if the node is now blacklisted.
func (m *ReputationManager) Slash(nodeID string, reason string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	rep, ok := m.nodes[nodeID]
	if !ok {
		return false
	}

	penalty := config.Cfg.Mesh.SlashPenalty
	rep.TrustScore -= penalty
	rep.Violations++
	rep.LastActivity = time.Now()

	if rep.TrustScore <= config.Cfg.Mesh.BlacklistThreshold {
		rep.TrustScore = 0
		rep.Blacklisted = true
		rep.Tier = TierUntrusted
		logging.Logger.Error("MESH NODE BLACKLISTED",
			"node_id", nodeID,
			"reason", reason,
			"violations", rep.Violations,
		)
	} else {
		rep.Tier = computeTier(rep.TrustScore, rep.Blacklisted)
		logging.Logger.Warn("Mesh node slashed",
			"node_id", nodeID,
			"penalty", penalty,
			"new_trust", rep.TrustScore,
			"reason", reason,
		)
	}

	if m.db != nil {
		blacklisted := 0
		if rep.Blacklisted {
			blacklisted = 1
		}
		m.db.Exec(`UPDATE mesh_reputation SET trust_score = ?, violations = ?, blacklisted = ?, last_activity = ? WHERE node_id = ?`,
			rep.TrustScore, rep.Violations, blacklisted, rep.LastActivity, nodeID)
	}

	if m.OnReputationChange != nil {
		go m.OnReputationChange(nodeID, -penalty, reason)
	}

	return rep.Blacklisted
}

// Boost applies a trust score increase (used by P2P Gossip).
func (m *ReputationManager) Boost(nodeID string, amount float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rep, ok := m.nodes[nodeID]
	if !ok {
		return
	}

	if !rep.Blacklisted {
		rep.TrustScore += amount
		if rep.TrustScore > 100.0 {
			rep.TrustScore = 100.0
		}
		rep.Tier = computeTier(rep.TrustScore, rep.Blacklisted)
		if m.db != nil {
			m.db.Exec(`UPDATE mesh_reputation SET trust_score = ? WHERE node_id = ?`, rep.TrustScore, nodeID)
		}
	}
}

// IsBlacklisted returns whether a node is banned from the mesh.
func (m *ReputationManager) IsBlacklisted(nodeID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if rep, ok := m.nodes[nodeID]; ok {
		return rep.Blacklisted
	}
	return false
}

// GetTier returns the trust tier for key routing decisions.
func (m *ReputationManager) GetTier(nodeID string) TrustTier {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if rep, ok := m.nodes[nodeID]; ok {
		return rep.Tier
	}
	return TierNew
}

// GetAllReputations returns a snapshot of all node reputations (for telemetry).
func (m *ReputationManager) GetAllReputations() []NodeReputation {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]NodeReputation, 0, len(m.nodes))
	for _, rep := range m.nodes {
		result = append(result, *rep)
	}
	return result
}

func computeTier(score float64, blacklisted bool) TrustTier {
	if blacklisted || score <= config.Cfg.Mesh.BlacklistThreshold {
		return TierUntrusted
	}
	if score >= config.Cfg.Mesh.PremiumTrustMinimum {
		return TierPremium
	}
	if score >= 40.0 {
		return TierTrusted
	}
	return TierNew
}
