package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"tkngate/internal/logging"
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

	// Read Ping
	decoder := json.NewDecoder(s)
	var ping Ping
	if err := decoder.Decode(&ping); err != nil {
		return
	}

	// Write Pong
	pong := Pong{Timestamp: time.Now().UnixMilli()}
	encoder := json.NewEncoder(s)
	_ = encoder.Encode(pong)
}

func handleRouteStream(s network.Stream) {
	defer s.Close()

	// Read RouteRequest
	decoder := json.NewDecoder(s)
	var req RouteRequest
	if err := decoder.Decode(&req); err != nil {
		logging.Logger.Warn("Failed to decode RouteRequest from peer", "peer", s.Conn().RemotePeer())
		return
	}

	// TODO: Actually handle the routing via the local DRR engine and API Keys.
	// For now, we return a simulated success response.
	// This would involve:
	// 1. ZKP Verification (drr.VerifyAndRoute)
	// 2. Making the upstream HTTP request
	// 3. Encrypting the response
	
	resp := RouteResponse{
		Success:            true,
		ErrorMessage:       "",
		EncryptedResponse:  []byte("simulated_encrypted_response"),
		InputTokensUsed:    10,
		OutputTokensUsed:   20,
	}

	encoder := json.NewEncoder(s)
	_ = encoder.Encode(resp)
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

	encoder := json.NewEncoder(s)
	if err := encoder.Encode(req); err != nil {
		return nil, err
	}

	var resp RouteResponse
	decoder := json.NewDecoder(s)
	if err := decoder.Decode(&resp); err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("peer closed connection unexpectedly")
		}
		return nil, err
	}

	return &resp, nil
}
