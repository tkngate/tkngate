package tkngate

import (
	"net/http"
	"os"
)

// Transport is an http.RoundTripper middleware that injects Tkngate headers
type Transport struct {
	Base       http.RoundTripper
	VirtualKey string
	Provider   string
	SessionID  string
}

// RoundTrip executes a single HTTP transaction, injecting necessary Tkngate headers
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	key := t.VirtualKey
	if key == "" {
		key = os.Getenv("TKNGATE_VIRTUAL_KEY")
	}

	// Clone the request to avoid modifying the original one
	newReq := req.Clone(req.Context())

	if key != "" {
		newReq.Header.Set("Authorization", "Bearer "+key)
	}

	if t.Provider != "" {
		newReq.Header.Set("X-Tkngate-Provider", t.Provider)
	}

	if t.SessionID != "" {
		newReq.Header.Set("X-Tkngate-Session-ID", t.SessionID)
	}

	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}

	return base.RoundTrip(newReq)
}

// WrapClient wraps an existing *http.Client with Tkngate middleware
func WrapClient(client *http.Client, virtualKey, provider, sessionID string) *http.Client {
	if client == nil {
		client = http.DefaultClient
	}

	transport := &Transport{
		Base:       client.Transport,
		VirtualKey: virtualKey,
		Provider:   provider,
		SessionID:  sessionID,
	}

	return &http.Client{
		Transport:     transport,
		CheckRedirect: client.CheckRedirect,
		Jar:           client.Jar,
		Timeout:       client.Timeout,
	}
}
