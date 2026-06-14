package auth

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	// KeyPrefix is the standard prefix for Tkngate Virtual Keys
	KeyPrefix = "tkngate-sk-"
	// KeyByteLength is the number of random bytes generated before base32 encoding
	KeyByteLength = 32
)

// VirtualKey represents an active Tkngate virtual API key and its plaintext value
type VirtualKey struct {
	Plaintext string
	Hash      string
}

// GenerateVirtualKey creates a new cryptographically secure virtual key.
// It returns the plaintext key (to show the user ONCE) and the bcrypt hash (to store in SQLite).
func GenerateVirtualKey() (*VirtualKey, error) {
	// Generate random bytes
	randomBytes := make([]byte, KeyByteLength)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Base32 encode without padding for cleaner API keys
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	plaintext := KeyPrefix + strings.ToLower(encoded)

	// Hash the plaintext key for secure storage (like passwords)
	// Cost 10 is a good balance between security and validation latency
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(plaintext), 10)
	if err != nil {
		return nil, fmt.Errorf("failed to hash virtual key: %w", err)
	}

	return &VirtualKey{
		Plaintext: plaintext,
		Hash:      string(hashBytes),
	}, nil
}

// VerifyKey checks if the provided plaintext key matches the stored hash
func VerifyKey(plaintext string, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext))
	return err == nil
}
