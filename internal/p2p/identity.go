package p2p

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"tkngate/internal/logging"
)

// Identity represents the cryptographic identity of the node.
type Identity struct {
	PrivKey libp2pcrypto.PrivKey
	PubKey  libp2pcrypto.PubKey
	PeerID  peer.ID
}

var GlobalIdentity *Identity

// LoadOrGenerateIdentity loads the identity from ~/.tkngate/identity.key or generates a new one.
func LoadOrGenerateIdentity() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get home dir: %w", err)
	}

	configDir := filepath.Join(homeDir, ".tkngate")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	keyPath := filepath.Join(configDir, "identity.key")

	var privKey libp2pcrypto.PrivKey

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		// Generate new key
		logging.Logger.Info("Generating new cryptographic identity...")
		
		// generate standard ed25519 key
		_, rawPriv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return fmt.Errorf("failed to generate ed25519 key: %w", err)
		}

		privKey, err = libp2pcrypto.UnmarshalEd25519PrivateKey(rawPriv)
		if err != nil {
			return fmt.Errorf("failed to unmarshal libp2p key: %w", err)
		}

		// Save to file
		rawKeyBytes, err := libp2pcrypto.MarshalPrivateKey(privKey)
		if err != nil {
			return fmt.Errorf("failed to marshal private key: %w", err)
		}
		
		hexKey := hex.EncodeToString(rawKeyBytes)
		if err := os.WriteFile(keyPath, []byte(hexKey), 0600); err != nil {
			return fmt.Errorf("failed to write identity key to disk: %w", err)
		}
	} else {
		// Load existing key
		logging.Logger.Info("Loading existing identity...")
		hexKeyBytes, err := os.ReadFile(keyPath)
		if err != nil {
			return fmt.Errorf("failed to read identity key: %w", err)
		}
		
		rawKeyBytes, err := hex.DecodeString(string(hexKeyBytes))
		if err != nil {
			return fmt.Errorf("failed to decode identity key hex: %w", err)
		}

		privKey, err = libp2pcrypto.UnmarshalPrivateKey(rawKeyBytes)
		if err != nil {
			return fmt.Errorf("failed to unmarshal loaded private key: %w", err)
		}
	}

	pubKey := privKey.GetPublic()
	peerID, err := peer.IDFromPublicKey(pubKey)
	if err != nil {
		return fmt.Errorf("failed to extract peer ID: %w", err)
	}

	GlobalIdentity = &Identity{
		PrivKey: privKey,
		PubKey:  pubKey,
		PeerID:  peerID,
	}

	logging.Logger.Info("P2P Identity loaded", "peer_id", peerID.String())
	return nil
}
