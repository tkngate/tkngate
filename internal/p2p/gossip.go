package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	
	"tkngate/internal/config"
	"tkngate/internal/logging"
	"tkngate/internal/mesh"
)

var (
	GlobalPubSub        *pubsub.PubSub
	ReputationTopic     *pubsub.Topic
	FraudTopic          *pubsub.Topic
	reputationSub       *pubsub.Subscription
	fraudSub            *pubsub.Subscription
)

const (
	ReputationTopicName = "/tkngate/reputation/1.0"
	FraudTopicName      = "/tkngate/fraud/1.0"
)

// InitGossip starts the GossipSub routing and joins topics.
func InitGossip(ctx context.Context) error {
	if GlobalHost == nil {
		return fmt.Errorf("p2p host not initialized")
	}

	if !config.Cfg.Mesh.ReputationEnabled {
		return nil
	}

	ps, err := pubsub.NewGossipSub(ctx, GlobalHost)
	if err != nil {
		return fmt.Errorf("failed to init gossipsub: %w", err)
	}
	GlobalPubSub = ps

	// Join Reputation Topic
	ReputationTopic, err = ps.Join(ReputationTopicName)
	if err != nil {
		return fmt.Errorf("failed to join reputation topic: %w", err)
	}

	reputationSub, err = ReputationTopic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to reputation topic: %w", err)
	}

	// Join Fraud Topic
	FraudTopic, err = ps.Join(FraudTopicName)
	if err != nil {
		return fmt.Errorf("failed to join fraud topic: %w", err)
	}

	fraudSub, err = FraudTopic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to fraud topic: %w", err)
	}

	go handleReputationMessages(ctx)
	go handleFraudMessages(ctx)

	logging.Logger.Info("GossipSub topics subscribed")
	return nil
}

func handleReputationMessages(ctx context.Context) {
	for {
		msg, err := reputationSub.Next(ctx)
		if err != nil {
			logging.Logger.Error("reputation sub error", "error", err)
			return
		}

		// Don't process our own messages
		if msg.ReceivedFrom == GlobalHost.ID() {
			continue
		}

		var update ReputationUpdate
		if err := json.Unmarshal(msg.Data, &update); err != nil {
			logging.Logger.Debug("Failed to unmarshal reputation update", "error", err)
			continue
		}

		// Verify signature
		pubKey, err := msg.GetFrom().ExtractPublicKey()
		if err != nil {
			continue
		}
		
		sigData := fmt.Sprintf("%s|%s|%f|%s|%d", update.SubjectNodeID, update.ReporterNodeID, update.TrustScoreChange, update.Reason, update.Timestamp)
		ok, err := pubKey.Verify([]byte(sigData), update.Signature)
		if err != nil || !ok {
			logging.Logger.Warn("Invalid signature on reputation update", "from", msg.ReceivedFrom)
			continue
		}

		// Apply update to local mesh reputation
		if mesh.GlobalReputation != nil {
			if update.TrustScoreChange < 0 {
				mesh.GlobalReputation.Slash(update.SubjectNodeID, fmt.Sprintf("P2P Slash from %s: %s", update.ReporterNodeID, update.Reason))
			} else if update.TrustScoreChange > 0 {
				mesh.GlobalReputation.Boost(update.SubjectNodeID, update.TrustScoreChange)
			}
		}
	}
}

func handleFraudMessages(ctx context.Context) {
	for {
		msg, err := fraudSub.Next(ctx)
		if err != nil {
			logging.Logger.Error("fraud sub error", "error", err)
			return
		}

		if msg.ReceivedFrom == GlobalHost.ID() {
			continue
		}

		var alert FraudAlert
		if err := json.Unmarshal(msg.Data, &alert); err != nil {
			continue
		}

		// Verify signature
		pubKey, err := msg.GetFrom().ExtractPublicKey()
		if err != nil {
			continue
		}

		sigData := fmt.Sprintf("%s|%s|%s|%d", alert.OffenderNodeID, alert.VictimNodeID, alert.EvidenceHash, alert.ReportedAt)
		ok, err := pubKey.Verify([]byte(sigData), alert.Signature)
		if err != nil || !ok {
			logging.Logger.Warn("Invalid signature on fraud alert", "from", msg.ReceivedFrom)
			continue
		}

		// Save the fraud proof to the ledger and slash
		if mesh.GlobalReputation != nil {
			proof := mesh.FraudProof{
				OffenderNodeID: alert.OffenderNodeID,
				VictimNodeID:   alert.VictimNodeID,
				EvidenceHash:   alert.EvidenceHash,
			}
			_ = mesh.SubmitFraudProof(proof)
		}
	}
}

// BroadcastReputationUpdate sends a signed reputation update to the mesh
func BroadcastReputationUpdate(ctx context.Context, subjectID string, change float64, reason string) {
	if ReputationTopic == nil || GlobalIdentity == nil {
		return
	}

	update := ReputationUpdate{
		SubjectNodeID:    subjectID,
		ReporterNodeID:   GlobalIdentity.PeerID.String(),
		TrustScoreChange: change,
		Reason:           reason,
		Timestamp:        time.Now().Unix(),
	}

	sigData := fmt.Sprintf("%s|%s|%f|%s|%d", update.SubjectNodeID, update.ReporterNodeID, update.TrustScoreChange, update.Reason, update.Timestamp)
	sig, err := GlobalIdentity.PrivKey.Sign([]byte(sigData))
	if err != nil {
		return
	}
	update.Signature = sig

	data, err := json.Marshal(update)
	if err != nil {
		return
	}

	_ = ReputationTopic.Publish(ctx, data)
}

// BroadcastFraudAlert sends a signed fraud alert to the mesh
func BroadcastFraudAlert(ctx context.Context, offenderID string, evidenceHash string) {
	if FraudTopic == nil || GlobalIdentity == nil {
		return
	}

	alert := FraudAlert{
		OffenderNodeID: offenderID,
		VictimNodeID:   GlobalIdentity.PeerID.String(),
		EvidenceHash:   evidenceHash,
		ReportedAt:     time.Now().Unix(),
	}

	sigData := fmt.Sprintf("%s|%s|%s|%d", alert.OffenderNodeID, alert.VictimNodeID, alert.EvidenceHash, alert.ReportedAt)
	sig, err := GlobalIdentity.PrivKey.Sign([]byte(sigData))
	if err != nil {
		return
	}
	alert.Signature = sig

	data, err := json.Marshal(alert)
	if err != nil {
		return
	}

	_ = FraudTopic.Publish(ctx, data)
}
