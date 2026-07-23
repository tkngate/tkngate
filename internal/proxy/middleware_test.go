package proxy

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"tkngate/internal/budget"
	"tkngate/internal/config"
	"tkngate/internal/logging"
	"tkngate/internal/tokenizer"
)

// setupTestEnv sets up enough of the global environment to allow proxy tests to run
func setupTestEnv(t *testing.T) {
	logging.InitLogger()
	// Initialize memory ledger so budget checks don't panic
	err := budget.InitMemoryLedger()
	if err != nil {
		t.Fatalf("Failed to init memory ledger: %v", err)
	}
	
	// Create a dummy session with $100
	err = budget.GlobalLedger.EnsureSession("test-session", 100.0)
	if err != nil {
		t.Fatalf("Failed to create dummy session: %v", err)
	}

	config.Cfg = config.Config{}
	config.Cfg.Budget.GlobalLimitUSD = 999999
	config.Cfg.Budget.RedThresholdPct = 95
	config.Cfg.Budget.FallbackModel = "gpt-4o-mini"
}

func TestProxyRoundTrip_Success(t *testing.T) {
	setupTestEnv(t)

	// Mock upstream OpenAI server
	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices": [{"message": {"content": "mock response"}}]}`))
	}))
	defer mockUpstream.Close()

	counter, _ := tokenizer.NewCounter()
	transport := &proxyTransport{
		Transport: http.DefaultTransport,
		Counter:   counter,
	}

	req, _ := http.NewRequest("POST", mockUpstream.URL, bytes.NewBufferString(`{"model": "gpt-4o", "messages": [{"role": "user", "content": "hello"}]}`))
	req.Header.Set("X-Tkngate-Session-ID", "test-session")

	res, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", res.StatusCode)
	}

	body, _ := io.ReadAll(res.Body)
	if !bytes.Contains(body, []byte("mock response")) {
		t.Errorf("Expected mock response, got %s", string(body))
	}
}

func TestProxyRoundTrip_RateLimitRetry(t *testing.T) {
	setupTestEnv(t)

	requestCount := 0
	// Mock upstream that returns 429 once, then 200
	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("Rate limited"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices": [{"message": {"content": "success on retry"}}]}`))
	}))
	defer mockUpstream.Close()

	counter, _ := tokenizer.NewCounter()
	transport := &proxyTransport{
		Transport: http.DefaultTransport,
		Counter:   counter,
	}

	req, _ := http.NewRequest("POST", mockUpstream.URL, bytes.NewBufferString(`{"model": "gpt-4o", "messages": [{"role": "user", "content": "hello"}]}`))
	req.Header.Set("X-Tkngate-Session-ID", "test-session")

	res, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 after retry, got %d", res.StatusCode)
	}

	if requestCount != 2 {
		t.Errorf("Expected exactly 2 requests (1 fail, 1 retry), got %d", requestCount)
	}
}

func TestCanonicalizePayload(t *testing.T) {
	raw := []byte(`{"model": "gpt-4", "temperature": 0.8, "messages": [{"role": "user", "content": "hi"}]}`)
	
	canonical := canonicalizePayload(raw)
	
	if bytes.Contains(canonical, []byte("temperature")) {
		t.Errorf("Expected canonical payload to drop temperature, got %s", string(canonical))
	}
	
	if !bytes.Contains(canonical, []byte("gpt-4")) {
		t.Errorf("Expected canonical payload to keep model, got %s", string(canonical))
	}
}

func TestCanonicalizePayload_ToolCalling(t *testing.T) {
	// Payload 1: Base request with tool calls and a specific tool order
	payload1 := []byte(`{
		"model": "gpt-4o",
		"messages": [
			{"role": "user", "content": "Fetch the weather"},
			{
				"role": "assistant",
				"content": null,
				"tool_calls": [
					{
						"id": "call_abc123",
						"type": "function",
						"function": {"name": "get_weather", "arguments": "{\"location\": \"London\"}"}
					}
				]
			},
			{
				"role": "tool",
				"tool_call_id": "call_abc123",
				"content": "{\"temp\": 15}"
			}
		],
		"tools": [
			{"type": "function", "function": {"name": "get_weather", "description": "Get weather"}},
			{"type": "function", "function": {"name": "search_web", "description": "Search"}}
		],
		"response_format": {"type": "json_object"}
	}`)

	// Payload 2: Identical conversational tree, but different randomly generated tool call IDs and different tool array order
	payload2 := []byte(`{
		"model": "gpt-4o",
		"messages": [
			{"role": "user", "content": "Fetch the weather"},
			{
				"role": "assistant",
				"content": null,
				"tool_calls": [
					{
						"id": "call_xyz890",
						"type": "function",
						"function": {"name": "get_weather", "arguments": "{\"location\": \"London\"}"}
					}
				]
			},
			{
				"role": "tool",
				"tool_call_id": "call_xyz890",
				"content": "{\"temp\": 15}"
			}
		],
		"tools": [
			{"type": "function", "function": {"name": "search_web", "description": "Search"}},
			{"type": "function", "function": {"name": "get_weather", "description": "Get weather"}}
		],
		"response_format": {"type": "json_object"}
	}`)

	hash1 := canonicalizePayload(payload1)
	hash2 := canonicalizePayload(payload2)

	if !bytes.Equal(hash1, hash2) {
		t.Errorf("canonicalizePayload failed to normalize tool calls and schema structures.\nHash1: %s\nHash2: %s", string(hash1), string(hash2))
	}
}
