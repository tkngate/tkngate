package telemetry

import (
	"encoding/json"
	"sync"
)

// DiffEvent represents a real-time diff between the primary and shadow models.
type DiffEvent struct {
	PrimaryProvider string `json:"primary_provider"`
	PrimaryModel    string `json:"primary_model"`
	PrimaryText     string `json:"primary_text"`
	ShadowProvider  string `json:"shadow_provider"`
	ShadowModel     string `json:"shadow_model"`
	ShadowText      string `json:"shadow_text"`
}

// DiffBroadcaster manages SSE clients for live diff viewing.
type DiffBroadcaster struct {
	clients map[chan []byte]bool
	mu      sync.Mutex
}

var GlobalDiffBroadcaster = &DiffBroadcaster{
	clients: make(map[chan []byte]bool),
}

// AddClient adds a new SSE channel.
func (b *DiffBroadcaster) AddClient(ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.clients[ch] = true
}

// RemoveClient removes an SSE channel.
func (b *DiffBroadcaster) RemoveClient(ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.clients, ch)
	close(ch)
}

// Broadcast sends a DiffEvent to all connected clients.
func (b *DiffBroadcaster) Broadcast(event DiffEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.clients {
		select {
		case ch <- data:
		default:
			// If client is blocked, drop the event to avoid blocking the proxy.
		}
	}
}
