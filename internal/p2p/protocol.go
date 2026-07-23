package p2p

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"tkngate/internal/config"
	"tkngate/internal/logging"
	"tkngate/internal/mesh"
	"tkngate/internal/pool"
	"tkngate/internal/zkp"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const (
	// Security: Maximum sizes to prevent OOM from malicious peers
	maxRouteRequestSize  = 10 * 1024 * 1024  // 10 MB max incoming route request
	maxRouteResponseSize = 25 * 1024 * 1024  // 25 MB max upstream LLM response
	maxPingSize          = 1024              // 1 KB max ping/pong payload
	streamDeadline       = 90 * time.Second  // Max time a stream can stay open

	// Security: Allowed providers whitelist to prevent provider injection
	allowedProviderList = "openai,anthropic,deepseek,mistral,ollama,kimi,groq"
)

const (
	PingProtocol  protocol.ID = "/tkngate/ping/1.0.0"
	RouteProtocol protocol.ID = "/tkngate/route/1.0.0"
)

// InitProtocols registers the stream handlers on the global host.
func InitProtocols() error {
	if GlobalHost == nil {
		return fmt.Errorf("p2p host not initialized")
	}

	GlobalHost.SetStreamHandler(PingProtocol, handlePingStream)
	GlobalHost.SetStreamHandler(RouteProtocol, handleRouteStream)

	return nil
}

func handlePingStream(s network.Stream) {
	defer s.Close()

	// Security: Set stream deadline to prevent slowloris
	_ = s.SetDeadline(time.Now().Add(10 * time.Second))

	// Security: Limit ping payload to 1KB to prevent OOM
	limitedReader := io.LimitReader(s, maxPingSize)
	decoder := json.NewDecoder(limitedReader)
	var ping Ping
	if err := decoder.Decode(&ping); err != nil {
		return
	}

	// Write Pong
	pong := Pong{Timestamp: time.Now().UnixMilli()}
	encoder := json.NewEncoder(s)
	_ = encoder.Encode(pong)
}

// isAllowedProvider checks the provider against a hardcoded whitelist.
func isAllowedProvider(provider string) bool {
	for _, p := range strings.Split(allowedProviderList, ",") {
		if strings.EqualFold(strings.TrimSpace(p), provider) {
			return true
		}
	}
	return false
}

func handleRouteStream(s network.Stream) {
	defer s.Close()

	// Security: Set stream deadline to prevent slowloris / connection exhaustion
	_ = s.SetDeadline(time.Now().Add(streamDeadline))

	// Security: Limit incoming request to 10MB to prevent OOM
	limitedReader := io.LimitReader(s, maxRouteRequestSize)
	decoder := json.NewDecoder(limitedReader)
	var req RouteRequest
	if err := decoder.Decode(&req); err != nil {
		logging.Logger.Warn("Failed to decode RouteRequest from peer", "peer", s.Conn().RemotePeer())
		return
	}

	// Security: Validate provider against whitelist to prevent injection
	if !isAllowedProvider(req.Provider) {
		sendRouteError(s, "invalid provider")
		logging.Logger.Warn("Peer sent invalid provider", "peer", s.Conn().RemotePeer(), "provider", req.Provider)
		return
	}

	remotePeerID := s.Conn().RemotePeer().String()

	// 1. Validate the peer's reputation
	if mesh.GlobalReputation != nil {
		if mesh.GlobalReputation.GetTier(remotePeerID) == mesh.TierUntrusted || mesh.GlobalReputation.IsBlacklisted(remotePeerID) {
			sendRouteError(s, "peer is untrusted or blacklisted")
			return
		}
	}

	// 2. ZKP Verification & DRR Key Selection
	var dynamicKey string
	var err error
	
	if pool.GlobalDRR != nil {
		var nonce *big.Int
		if len(req.ZkpNonce) > 0 {
			nonce = new(big.Int).SetBytes(req.ZkpNonce)
		} else {
			nonce = big.NewInt(0)
		}
		
		var grothProof groth16.Proof
		if len(req.ZkpProof) > 0 {
			grothProof = groth16.NewProof(ecc.BN254)
			_, _ = grothProof.ReadFrom(bytes.NewReader(req.ZkpProof))
		}

		dynamicKey, err = pool.GlobalDRR.VerifyAndRoute(req.Provider, remotePeerID, req.EstimatedTokens, grothProof, nonce)
	}

	if err != nil || dynamicKey == "" {
		sendRouteError(s, fmt.Sprintf("failed to get route key: %v", err))
		return
	}

	// 3. Make the upstream HTTP request
	providerCfg, ok := config.Cfg.Providers[req.Provider]
	if !ok {
		sendRouteError(s, fmt.Sprintf("unsupported provider: %s", req.Provider))
		return
	}

	baseURL := strings.TrimSuffix(providerCfg.BaseURL, "/")
	// Decrypt payload using E2EE
	remotePubKey := s.Conn().RemotePublicKey()
	
	if GlobalIdentity == nil || remotePubKey == nil {
		sendRouteError(s, "e2ee crypto missing")
		return
	}
	
	aesKey, err := DeriveSharedSecret(GlobalIdentity.PrivKey, remotePubKey)
	if err != nil {
		sendRouteError(s, fmt.Sprintf("e2ee derivation failed: %v", err))
		return
	}
	
	plaintextPrompt, err := DecryptPayload(aesKey, req.EncryptedPrompt)
	if err != nil {
		sendRouteError(s, fmt.Sprintf("e2ee decryption failed: %v", err))
		return
	}

	targetURL := baseURL + "/chat/completions"

	httpReq, err := http.NewRequest("POST", targetURL, bytes.NewReader(plaintextPrompt))
	if err != nil {
		sendRouteError(s, "failed to create http request")
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if req.Provider == "anthropic" {
		httpReq.Header.Set("x-api-key", dynamicKey)
		httpReq.Header.Set("anthropic-version", "2023-06-01")
	} else {
		httpReq.Header.Set("Authorization", "Bearer "+dynamicKey)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	httpRes, err := client.Do(httpReq)
	if err != nil {
		sendRouteError(s, fmt.Sprintf("upstream request failed: %v", err))
		return
	}
	defer httpRes.Body.Close()

	// Security: Limit upstream response to 25MB to prevent OOM
	limitedBody := io.LimitReader(httpRes.Body, maxRouteResponseSize)
	bodyBytes, _ := io.ReadAll(limitedBody)

	if httpRes.StatusCode >= 400 {
		// Security: Do NOT leak upstream error body to the peer — it may
		// contain API key status info, rate-limit details, or account metadata.
		logging.Logger.Warn("Upstream error during P2P route", "status", httpRes.StatusCode, "body_len", len(bodyBytes))
		sendRouteError(s, fmt.Sprintf("upstream returned status %d", httpRes.StatusCode))
		return
	}

	// Calculate tokens used from the response
	var genericResp struct {
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	_ = json.Unmarshal(bodyBytes, &genericResp)

	encryptedBody, err := EncryptPayload(aesKey, bodyBytes)
	if err != nil {
		sendRouteError(s, fmt.Sprintf("failed to encrypt response: %v", err))
		return
	}

	resp := RouteResponse{
		Success:            true,
		ErrorMessage:       "",
		EncryptedResponse:  encryptedBody,
		InputTokensUsed:    genericResp.Usage.PromptTokens,
		OutputTokensUsed:   genericResp.Usage.CompletionTokens,
	}

	encoder := json.NewEncoder(s)
	_ = encoder.Encode(resp)
}

func sendRouteError(s network.Stream, msg string) {
	resp := RouteResponse{
		Success:      false,
		ErrorMessage: msg,
	}
	_ = json.NewEncoder(s).Encode(resp)
}

// SendPing sends a ping to a remote peer and returns the round trip latency.
func SendPing(ctx context.Context, peerID peer.ID) (time.Duration, error) {
	if GlobalHost == nil {
		return 0, fmt.Errorf("p2p host not initialized")
	}

	s, err := GlobalHost.NewStream(ctx, peerID, PingProtocol)
	if err != nil {
		return 0, err
	}
	defer s.Close()

	start := time.Now()
	
	ping := Ping{Timestamp: start.UnixMilli()}
	encoder := json.NewEncoder(s)
	if err := encoder.Encode(ping); err != nil {
		return 0, err
	}

	var pong Pong
	decoder := json.NewDecoder(s)
	if err := decoder.Decode(&pong); err != nil {
		return 0, err
	}

	return time.Since(start), nil
}

// SendRouteRequest forwards an AI prompt to a remote peer for processing.
func SendRouteRequest(ctx context.Context, peerID peer.ID, req RouteRequest) (*RouteResponse, error) {
	if GlobalHost == nil {
		return nil, fmt.Errorf("p2p host not initialized")
	}

	s, err := GlobalHost.NewStream(ctx, peerID, RouteProtocol)
	if err != nil {
		return nil, err
	}
	defer s.Close()

	// Security: Set stream deadline
	_ = s.SetDeadline(time.Now().Add(streamDeadline))

	encoder := json.NewEncoder(s)
	if err := encoder.Encode(req); err != nil {
		return nil, err
	}

	// Security: Limit response from peer to prevent OOM
	limitedReader := io.LimitReader(s, maxRouteResponseSize)
	var resp RouteResponse
	decoder := json.NewDecoder(limitedReader)
	if err := decoder.Decode(&resp); err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("peer closed connection unexpectedly")
		}
		return nil, err
	}

	return &resp, nil
}

// OffloadRequest attempts to route an LLM prompt to a trusted peer on the mesh.
func OffloadRequest(ctx context.Context, provider string, model string, sessionID string, estimatedTokens int, inputBody []byte) (*RouteResponse, error) {
	if GlobalHost == nil {
		return nil, fmt.Errorf("p2p host not initialized")
	}

	peers := GlobalHost.Network().Peers()
	if len(peers) == 0 {
		return nil, fmt.Errorf("no peers connected")
	}

	// Find all trusted peers that support our PingProtocol
	var trustedPeers []peer.ID
	for _, p := range peers {
		// Only try to offload to peers that explicitly support our protocol
		supports, err := GlobalHost.Peerstore().SupportsProtocols(p, PingProtocol)
		if err != nil || len(supports) == 0 {
			continue
		}

		if mesh.GlobalReputation != nil {
			tier := mesh.GlobalReputation.GetTier(p.String())
			if tier != mesh.TierUntrusted && !mesh.GlobalReputation.IsBlacklisted(p.String()) {
				trustedPeers = append(trustedPeers, p)
			}
		} else {
			trustedPeers = append(trustedPeers, p)
		}
	}

	if len(trustedPeers) == 0 {
		return nil, fmt.Errorf("no trusted peers available for offloading")
	}

	// Ping peers concurrently to measure latency
	type peerLatency struct {
		id      peer.ID
		latency time.Duration
	}
	var results []peerLatency
	var mu sync.Mutex
	var wg sync.WaitGroup

	pingCtx, cancel := context.WithTimeout(ctx, 1000*time.Millisecond)
	defer cancel()

	for _, p := range trustedPeers {
		wg.Add(1)
		go func(pid peer.ID) {
			defer wg.Done()
			lat, err := SendPing(pingCtx, pid)
			if err == nil {
				mu.Lock()
				results = append(results, peerLatency{id: pid, latency: lat})
				mu.Unlock()
			}
		}(p)
	}
	wg.Wait()

	if len(results) == 0 {
		return nil, fmt.Errorf("all trusted peers failed to respond to ping")
	}

	// Sort by lowest latency
	sort.Slice(results, func(i, j int) bool {
		return results[i].latency < results[j].latency
	})

	// Generate E2EE Encrypted Payload for the specific peer
	// Since payload encryption is per-peer (different shared secret), we do it inside the retry loop.
	
	var lastErr error
	for _, peerRes := range results {
		selectedPeer := peerRes.id
		
		remotePubKey := GlobalHost.Peerstore().PubKey(selectedPeer)
		if remotePubKey == nil {
			lastErr = fmt.Errorf("missing public key for peer %s", selectedPeer)
			continue
		}
		
		aesKey, err := DeriveSharedSecret(GlobalIdentity.PrivKey, remotePubKey)
		if err != nil {
			lastErr = fmt.Errorf("e2ee derivation failed: %v", err)
			continue
		}
		
		ciphertext, err := EncryptPayload(aesKey, inputBody)
		if err != nil {
			lastErr = fmt.Errorf("e2ee encryption failed: %v", err)
			continue
		}

		req := RouteRequest{
			SessionID:       sessionID,
			Provider:        provider,
			Model:           model,
			EstimatedTokens: estimatedTokens,
			EncryptedPrompt: ciphertext,
		}

		if config.Cfg.Mesh.StrictZKPMode {
			if zkp.GlobalZKP != nil {
				proof, nonce, err := zkp.GlobalZKP.GenerateProof(string(inputBody))
				if err != nil {
					lastErr = fmt.Errorf("zkp generation failed: %v", err)
					continue
				}
				
				var proofBuf bytes.Buffer
				if _, err := proof.WriteTo(&proofBuf); err == nil {
					req.ZkpProof = proofBuf.Bytes()
				}
				if nonce != nil {
					req.ZkpNonce = nonce.Bytes()
				}
			} else {
				req.ZkpProof = []byte("dummy_proof")
			}
		}

		logging.Logger.Info("Offloading prompt to P2P peer", "peer", selectedPeer.String(), "latency", peerRes.latency, "provider", provider)
		resp, err := SendRouteRequest(ctx, selectedPeer, req)
		if err != nil {
			logging.Logger.Debug("P2P offload failed, retrying next peer", "failed_peer", selectedPeer.String(), "error", err)
			lastErr = err
			continue
		}
		if !resp.Success {
			logging.Logger.Debug("P2P offload peer returned error, retrying next peer", "failed_peer", selectedPeer.String(), "error", resp.ErrorMessage)
			lastErr = fmt.Errorf("peer error: %s", resp.ErrorMessage)
			continue
		}
		
		decryptedResp, err := DecryptPayload(aesKey, resp.EncryptedResponse)
		if err != nil {
			lastErr = fmt.Errorf("failed to decrypt response from peer %s: %v", selectedPeer.String(), err)
			continue
		}
		resp.EncryptedResponse = decryptedResp
		
		return resp, nil
	}

	return nil, fmt.Errorf("all P2P offload attempts failed, last error: %v", lastErr)
}
