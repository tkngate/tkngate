package mesh

import (
	"database/sql"
	"testing"
	"time"
	"tkngate/internal/config"
	"tkngate/internal/logging"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}
	return db
}

func setupConfig() {
	config.Cfg = config.Config{}
	config.Cfg.Mesh.ReputationEnabled = true
	config.Cfg.Mesh.InitialTrustScore = 50.0
	config.Cfg.Mesh.SlashPenalty = 25.0
	config.Cfg.Mesh.BlacklistThreshold = 10.0
	config.Cfg.Mesh.PremiumTrustMinimum = 80.0
}

func TestReputationManager_GetOrCreate(t *testing.T) {
	logging.InitLogger()
	setupConfig()
	db := setupTestDB(t)
	InitReputation(db)

	nodeID := "node-1"
	rep := GlobalReputation.GetOrCreate(nodeID)

	if rep.TrustScore != 50.0 {
		t.Errorf("expected initial score 50.0, got %f", rep.TrustScore)
	}
	if rep.Tier != TierNew {
		t.Errorf("expected tier NEW, got %s", rep.Tier)
	}

	// Test persistence
	var score float64
	err := db.QueryRow(`SELECT trust_score FROM mesh_reputation WHERE node_id = ?`, nodeID).Scan(&score)
	if err != nil {
		t.Fatalf("failed to read from db: %v", err)
	}
	if score != 50.0 {
		t.Errorf("db score mismatch: got %f", score)
	}
}

func TestReputationManager_RecordSuccess(t *testing.T) {
	logging.InitLogger()
	setupConfig()
	InitReputation(nil) // test without DB

	nodeID := "node-success"
	rep := GlobalReputation.GetOrCreate(nodeID)

	GlobalReputation.RecordSuccess(nodeID)
	
	if rep.TotalRequests != 1 {
		t.Errorf("expected 1 request, got %d", rep.TotalRequests)
	}
	if rep.TrustScore <= 50.0 {
		t.Errorf("expected score > 50.0, got %f", rep.TrustScore)
	}
}

func TestReputationManager_SlashAndBlacklist(t *testing.T) {
	logging.InitLogger()
	setupConfig()
	InitReputation(nil)

	nodeID := "node-malicious"
	rep := GlobalReputation.GetOrCreate(nodeID)

	// Slash 1: 50 - 25 = 25
	blacklisted := GlobalReputation.Slash(nodeID, "spam")
	if blacklisted {
		t.Error("expected not to be blacklisted yet")
	}
	if rep.TrustScore != 25.0 {
		t.Errorf("expected 25.0, got %f", rep.TrustScore)
	}

	// Slash 2: 25 - 25 = 0 -> blacklisted (threshold is 10)
	blacklisted = GlobalReputation.Slash(nodeID, "fraud proof")
	if !blacklisted {
		t.Error("expected to be blacklisted")
	}
	if rep.TrustScore != 0.0 {
		t.Errorf("expected 0.0, got %f", rep.TrustScore)
	}
	if rep.Tier != TierUntrusted {
		t.Errorf("expected UNTRUSTED tier, got %s", rep.Tier)
	}
	if !GlobalReputation.IsBlacklisted(nodeID) {
		t.Error("IsBlacklisted should return true")
	}
}

func TestSubmitFraudProof(t *testing.T) {
	logging.InitLogger()
	setupConfig()
	InitReputation(nil)

	nodeID := "node-fraudster"
	GlobalReputation.GetOrCreate(nodeID)

	proof := FraudProof{
		OffenderNodeID: nodeID,
		VictimNodeID:   "victim-1",
		EvidenceHash:   "abc123hash",
		ReportedAt:     time.Now(),
	}

	// A single fraud proof slash of 25 drops them to 25. Another drops to 0.
	err := SubmitFraudProof(proof)
	if err != nil {
		t.Fatalf("SubmitFraudProof failed: %v", err)
	}
	
	rep := GlobalReputation.GetOrCreate(nodeID)
	if rep.TrustScore != 25.0 {
		t.Errorf("expected trust to drop to 25.0, got %f", rep.TrustScore)
	}
	
	SubmitFraudProof(proof) // Drop to 0
	
	if !GlobalReputation.IsBlacklisted(nodeID) {
		t.Error("Node should be blacklisted after two fraud proofs")
	}
}
