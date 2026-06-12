package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var masterKey []byte

// InitCrypto loads or generates the master key
func InitCrypto() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	keyPath := filepath.Join(homeDir, ".tkngate", "master.key")

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		// Generate new 32-byte key
		key := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, key); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
			return err
		}
		if err := os.WriteFile(keyPath, key, 0600); err != nil {
			return err
		}
		masterKey = key
	} else {
		key, err := os.ReadFile(keyPath)
		if err != nil {
			return err
		}
		if len(key) != 32 {
			return fmt.Errorf("invalid master key length in %s", keyPath)
		}
		masterKey = key
	}
	return nil
}

func Encrypt(plaintext string) (string, error) {
	if masterKey == nil {
		return "", fmt.Errorf("crypto not initialized")
	}

	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

func Decrypt(ciphertextHex string) (string, error) {
	if masterKey == nil {
		return "", fmt.Errorf("crypto not initialized")
	}

	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
