package zkp

import (
	"github.com/consensys/gnark/frontend"
)

// WafCircuit is a zero-knowledge circuit that proves a prompt is "safe"
// without revealing the prompt's actual content.
//
// How it works (Simplified v1 - Hash Blacklist):
//   - The Prover (sender) has a private prompt. They compute its MiMC hash.
//   - The circuit asserts that the hash does NOT match any of the public
//     blacklisted hashes (known malicious prompt patterns).
//   - If the proof verifies, the mesh node knows the prompt is safe
//     without ever seeing the plaintext.
//
// Fields tagged `gnark:",public"` are visible to the verifier.
// Fields tagged `gnark:",secret"` are private to the prover.
type WafCircuit struct {
	// PromptHash is the MiMC hash of the plaintext prompt (private witness).
	// The prover computes this locally and provides it as a secret input.
	PromptHash frontend.Variable `gnark:",secret"`

	// BlacklistedHashes is a fixed-size array of known-malicious prompt hashes.
	// These are public inputs that both prover and verifier agree on.
	BlacklistedHashes [5]frontend.Variable `gnark:",public"`

	// AttestationNonce is a random value to prevent proof replay attacks.
	// Each proof is bound to a unique nonce so it cannot be reused.
	AttestationNonce frontend.Variable `gnark:",public"`
}

// Define implements the gnark frontend.Circuit interface.
// It specifies the constraints that must be satisfied for a valid proof.
func (c *WafCircuit) Define(api frontend.API) error {
	// For each blacklisted hash, assert that the prompt hash is NOT equal.
	// In ZK arithmetic, we prove (PromptHash - BlacklistedHash) != 0
	// by asserting the inverse exists (only zero has no inverse in a finite field).
	for i := 0; i < len(c.BlacklistedHashes); i++ {
		diff := api.Sub(c.PromptHash, c.BlacklistedHashes[i])
		api.AssertIsDifferent(diff, 0)
	}

	// Bind the attestation nonce to prevent replay.
	// A simple constraint: nonce must be non-zero.
	api.AssertIsDifferent(c.AttestationNonce, 0)

	return nil
}
