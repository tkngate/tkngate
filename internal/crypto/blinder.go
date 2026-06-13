package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

var masterKey []byte


func InitCrypto() error {
	keyStr := os.Getenv("TKNGATE_MASTER_KEY")
	if keyStr == "" {
		return fmt.Errorf("FATAL SECURITY ERROR: TKNGATE_MASTER_KEY environment variable is missing. It must be exactly 32 characters to enable zero-knowledge encryption for the token mesh")
	}

	key := []byte(keyStr)
	if len(key) != 32 {
		return fmt.Errorf("FATAL SECURITY ERROR: TKNGATE_MASTER_KEY must be exactly 32 bytes (characters). Current length is %d", len(key))
	}

	masterKey = key
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
