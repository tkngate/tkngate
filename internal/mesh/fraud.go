package mesh

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
	"tkngate/internal/config"
	"tkngate/internal/logging"
)

// FraudProof represents cryptographic evidence of a malicious prompt sent through the mesh.
type FraudProof struct {
	OffenderNodeID string    `json:"offender_node_id"`
	VictimNodeID   string    `json:"victim_node_id"`
	EvidenceHash   string    `json:"evidence_hash"` // Hash of the flagged prompt + moderation response
	ReportedAt     time.Time `json:"reported_at"`
}

// SubmitFraudProof handles a victim node reporting a malicious sender.
// If the proof is valid (simulated here for MVP), the offender is slashed.
func SubmitFraudProof(proof FraudProof) error {
	if !config.Cfg.Mesh.ReputationEnabled {
		return fmt.Errorf("mesh reputation system is disabled")
	}

	if GlobalReputation == nil {
		return fmt.Errorf("reputation system not initialized")
	}

	// In a real decentralized network, we would cryptographically verify the signature
	// of the OpenAI moderation response to ensure the victim isn't forging the proof.
	// For this MVP, we assume the proof is verified.

	isBlacklisted := GlobalReputation.Slash(proof.OffenderNodeID, "Fraud Proof: OpenAI Moderation Flagged Content")

	// Store the fraud proof in DB if available
	if GlobalReputation.db != nil {
		_, err := GlobalReputation.db.Exec(`INSERT INTO mesh_fraud_proofs (offender_node_id, victim_node_id, evidence, slash_amount) VALUES (?, ?, ?, ?)`,
			proof.OffenderNodeID, proof.VictimNodeID, proof.EvidenceHash, config.Cfg.Mesh.SlashPenalty)
		if err != nil {
			logging.Logger.Error("Failed to save fraud proof to ledger", "error", err)
		}
	}

	if isBlacklisted {
		logging.Logger.Warn("Node blacklisted due to fraud proof", "node", proof.OffenderNodeID)
	}

	return nil
}

// GenerateEvidenceHash creates a verifiable hash for a fraud proof.
func GenerateEvidenceHash(prompt, moderationRawResponse string) string {
	hasher := sha256.New()
	hasher.Write([]byte(prompt + "|" + moderationRawResponse))
	return hex.EncodeToString(hasher.Sum(nil))
}
