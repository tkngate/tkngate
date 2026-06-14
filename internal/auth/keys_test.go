package auth

import (
	"strings"
	"testing"
)

func TestGenerateVirtualKey(t *testing.T) {
	key, err := GenerateVirtualKey()
	if err != nil {
		t.Fatalf("GenerateVirtualKey() returned error: %v", err)
	}

	// Verify the plaintext key has the correct prefix
	if !strings.HasPrefix(key.Plaintext, KeyPrefix) {
		t.Errorf("Expected key prefix '%s', got '%s'", KeyPrefix, key.Plaintext[:len(KeyPrefix)])
	}

	// Verify the hash is non-empty (bcrypt output)
	if key.Hash == "" {
		t.Error("Expected non-empty hash")
	}

	// Verify the hash starts with bcrypt identifier
	if !strings.HasPrefix(key.Hash, "$2a$") {
		t.Errorf("Expected bcrypt hash prefix '$2a$', got '%s'", key.Hash[:4])
	}

	t.Logf("Generated key: %s (hash length: %d)", key.Plaintext, len(key.Hash))
}

func TestVerifyKey_Valid(t *testing.T) {
	key, err := GenerateVirtualKey()
	if err != nil {
		t.Fatalf("GenerateVirtualKey() returned error: %v", err)
	}

	// The correct plaintext should verify against its own hash
	if !VerifyKey(key.Plaintext, key.Hash) {
		t.Error("VerifyKey() returned false for a valid key/hash pair")
	}
}

func TestVerifyKey_Invalid(t *testing.T) {
	key, err := GenerateVirtualKey()
	if err != nil {
		t.Fatalf("GenerateVirtualKey() returned error: %v", err)
	}

	// A random wrong key should NOT verify
	if VerifyKey("tkngate-sk-thisissomegarbage", key.Hash) {
		t.Error("VerifyKey() returned true for an invalid key")
	}
}

func TestGenerateVirtualKey_Uniqueness(t *testing.T) {
	key1, _ := GenerateVirtualKey()
	key2, _ := GenerateVirtualKey()

	if key1.Plaintext == key2.Plaintext {
		t.Error("Two generated keys should not have the same plaintext")
	}

	if key1.Hash == key2.Hash {
		t.Error("Two generated keys should not have the same hash")
	}
}
