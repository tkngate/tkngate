package p2p

// FraudAlert represents a cryptographic proof that a node sent malicious traffic.
type FraudAlert struct {
	OffenderNodeID string `json:"offender_node_id"`
	VictimNodeID   string `json:"victim_node_id"`
	EvidenceHash   string `json:"evidence_hash"`
	ReportedAt     int64  `json:"reported_at"`
	Signature      []byte `json:"signature"` // Signature of the above fields by the victim
}

// ReputationUpdate is broadcast when a node's score changes.
type ReputationUpdate struct {
	SubjectNodeID    string  `json:"subject_node_id"` // The node whose score is changing
	ReporterNodeID   string  `json:"reporter_node_id"` // The node reporting the change
	TrustScoreChange float64 `json:"trust_score_change"` // Amount changed (e.g. -20 for slash)
	Reason           string  `json:"reason"`
	Timestamp        int64   `json:"timestamp"`
	Signature        []byte  `json:"signature"` // Signature of the above by the reporter
}

// RouteRequest is sent to a peer to route a prompt.
type RouteRequest struct {
	SessionID       string `json:"session_id"`
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	EstimatedTokens int    `json:"estimated_tokens"`
	EncryptedPrompt []byte `json:"encrypted_prompt"`
	
	// ZKP fields (Groth16 BN254)
	ZkpProof []byte `json:"zkp_proof"` // serialized proof
	ZkpNonce []byte `json:"zkp_nonce"` // nonce used in proof
}

// RouteResponse is sent back to the requester.
type RouteResponse struct {
	Success            bool   `json:"success"`
	ErrorMessage       string `json:"error_message"`
	EncryptedResponse  []byte `json:"encrypted_response"`
	InputTokensUsed    int    `json:"input_tokens_used"`
	OutputTokensUsed   int    `json:"output_tokens_used"`
}

// Ping/Pong for latency checks.
type Ping struct {
	Timestamp int64 `json:"timestamp"`
}

type Pong struct {
	Timestamp int64 `json:"timestamp"`
}
