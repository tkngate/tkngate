package p2p

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"io"

	"filippo.io/edwards25519"
	"github.com/libp2p/go-libp2p/core/crypto"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

// DeriveSharedSecret converts Ed25519 keys to X25519 and computes a Diffie-Hellman shared secret.
func DeriveSharedSecret(privKey crypto.PrivKey, pubKey crypto.PubKey) ([]byte, error) {
	// 1. Extract raw bytes from libp2p keys
	privBytes, err := privKey.Raw()
	if err != nil {
		return nil, fmt.Errorf("failed to extract private key bytes: %w", err)
	}
	pubBytes, err := pubKey.Raw()
	if err != nil {
		return nil, fmt.Errorf("failed to extract public key bytes: %w", err)
	}

	// 2. Convert Ed25519 Private Key to X25519
	// Libp2p uses standard 64-byte Ed25519 private keys where the first 32 bytes are the seed.
	if len(privBytes) != 64 {
		return nil, fmt.Errorf("expected 64-byte Ed25519 private key, got %d", len(privBytes))
	}
	seed := privBytes[:32]
	
	h := sha512.Sum512(seed)
	scalar := h[:32]
	scalar[0] &= 248
	scalar[31] &= 127
	scalar[31] |= 64

	// 3. Convert Ed25519 Public Key to X25519
	point, err := new(edwards25519.Point).SetBytes(pubBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid ed25519 public key: %w", err)
	}
	x25519PubKey := point.BytesMontgomery()

	// 4. Compute Diffie-Hellman
	sharedSecret, err := curve25519.X25519(scalar, x25519PubKey)
	if err != nil {
		return nil, fmt.Errorf("curve25519 dh failed: %w", err)
	}

	// 5. HKDF to derive 32-byte AES key
	hkdfReader := hkdf.New(sha256.New, sharedSecret, nil, []byte("tkngate-p2p-e2ee"))
	aesKey := make([]byte, 32)
	if _, err := io.ReadFull(hkdfReader, aesKey); err != nil {
		return nil, fmt.Errorf("hkdf failed: %w", err)
	}

	return aesKey, nil
}

func EncryptPayload(aesKey []byte, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func DecryptPayload(aesKey []byte, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertextBytes := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertextBytes, nil)
}
