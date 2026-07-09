package cache

import (
	"testing"
	"tkngate/internal/logging"
)

func TestInMemoryCache_PutAndGet(t *testing.T) {
	logging.InitLogger()
	InitCache(10, 60, "") // In-memory, max 10 entries, 60s TTL

	payload := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}]}`)
	response := []byte(`{"choices":[{"message":{"content":"hi!"}}]}`)

	// Put a response into the cache
	GlobalCache.Put(payload, response, 200, 0.01)

	// Get should return a hit
	hit := GlobalCache.Get(payload)
	if hit == nil {
		t.Fatal("Expected cache HIT, got nil")
	}

	if string(hit.ResponseBody) != string(response) {
		t.Errorf("Cached response body mismatch: got %s", string(hit.ResponseBody))
	}

	if hit.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", hit.StatusCode)
	}

	t.Log("Cache HIT verified successfully")
}

func TestInMemoryCache_Miss(t *testing.T) {
	logging.InitLogger()
	InitCache(10, 60, "")

	payload := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"unique prompt never cached"}]}`)

	hit := GlobalCache.Get(payload)
	if hit != nil {
		t.Error("Expected cache MISS, got a hit")
	}
}

func TestInMemoryCache_Stats(t *testing.T) {
	logging.InitLogger()
	InitCache(10, 60, "")

	payload := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"stats test"}]}`)
	response := []byte(`{"choices":[{"message":{"content":"ok"}}]}`)

	// 1 miss
	GlobalCache.Get(payload)
	// Store it
	GlobalCache.Put(payload, response, 200, 0.05)
	// 1 hit
	GlobalCache.Get(payload)

	hits, misses, size, savings := GlobalCache.Stats()

	if hits != 1 {
		t.Errorf("Expected 1 hit, got %d", hits)
	}
	if misses != 1 {
		t.Errorf("Expected 1 miss, got %d", misses)
	}
	if size != 1 {
		t.Errorf("Expected cache size 1, got %d", size)
	}
	if savings < 0.04 {
		t.Errorf("Expected savings >= $0.04, got $%.4f", savings)
	}

	t.Logf("Stats: hits=%d, misses=%d, size=%d, savings=$%.4f", hits, misses, size, savings)
}

func TestInMemoryCache_Eviction(t *testing.T) {
	logging.InitLogger()
	InitCache(2, 60, "") // Max 2 entries

	p1 := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"prompt 1"}]}`)
	p2 := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"prompt 2"}]}`)
	p3 := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"prompt 3"}]}`)
	resp := []byte(`{"ok":true}`)

	GlobalCache.Put(p1, resp, 200, 0.01)
	GlobalCache.Put(p2, resp, 200, 0.01)
	GlobalCache.Put(p3, resp, 200, 0.01) // This should evict p1

	// p1 should be evicted
	if GlobalCache.Get(p1) != nil {
		t.Error("Expected p1 to be evicted, but got a hit")
	}

	// p3 should still be there
	if GlobalCache.Get(p3) == nil {
		t.Error("Expected p3 to be cached, but got a miss")
	}

	t.Log("LRU eviction verified successfully")
}

func TestCacheInterface_InMemory(t *testing.T) {
	logging.InitLogger()
	InitCache(10, 60, "") // No Redis URI → InMemoryCache

	// Verify GlobalCache implements the Cache interface
	var _ Cache = GlobalCache
	t.Log("GlobalCache correctly implements Cache interface (InMemoryCache)")
}

func TestInMemoryCache_ToolCalling(t *testing.T) {
	logging.InitLogger()
	InitCache(10, 60, "")

	plainPayload := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"What is the weather?"}]}`)
	toolPayload := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"What is the weather?"}],"tools":[{"type":"function","function":{"name":"get_weather"}}]}`)

	plainResponse := []byte(`{"choices":[{"message":{"content":"I don't know the weather."}}]}`)

	// Put plain response
	GlobalCache.Put(plainPayload, plainResponse, 200, 0.01)

	// Tool payload should NOT get a hit from the plain cache
	hit := GlobalCache.Get(toolPayload)
	if hit != nil {
		t.Error("CRITICAL BUG: Tool payload got a cache hit from a plain payload! Semantic key bleeding detected.")
	}

	t.Log("Tool-calling payloads are correctly isolated from plain text payloads")
}
