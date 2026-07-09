package cache

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// setupTestRedis attempts to connect to a local Redis instance for integration testing.
// It skips the test if Redis is not running.
func setupTestRedis(t *testing.T) *RedisCache {
	uri := "redis://localhost:6379"
	
	opt, err := redis.ParseURL(uri)
	if err != nil {
		t.Fatalf("Failed to parse URI: %v", err)
	}

	client := redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Skipping integration test: Redis not running on %s", uri)
	}

	// Make sure we have a clean slate for the test
	cache := &RedisCache{
		client: client,
		ttl:    30 * time.Second,
	}
	cache.Clear()
	
	return cache
}

func TestRedisCache_PutAndGet(t *testing.T) {
	cache := setupTestRedis(t)
	defer cache.Clear()

	payload := []byte(`{"model": "gpt-4", "messages": [{"role": "user", "content": "hello"}]}`)
	responseBody := []byte(`{"choices": [{"message": {"content": "world"}}]}`)

	// 1. Initial Get should be a miss
	entry := cache.Get(payload)
	if entry != nil {
		t.Fatal("Expected cache miss, got hit")
	}

	// 2. Put into cache
	cache.Put(payload, responseBody, 200, 0.05)

	// 3. Get should now hit
	entry = cache.Get(payload)
	if entry == nil {
		t.Fatal("Expected cache hit, got miss")
	}

	if string(entry.ResponseBody) != string(responseBody) {
		t.Errorf("Expected response body %s, got %s", responseBody, entry.ResponseBody)
	}
	if entry.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", entry.StatusCode)
	}
	if entry.Headers["X-Tkngate-Cache"] != "STORED-REDIS" {
		t.Errorf("Expected redis cache header, got %s", entry.Headers["X-Tkngate-Cache"])
	}
}

func TestRedisCache_Stats(t *testing.T) {
	cache := setupTestRedis(t)
	defer cache.Clear()

	payload1 := []byte(`{"model": "gpt-4", "messages": [{"role": "user", "content": "1"}]}`)
	payload2 := []byte(`{"model": "gpt-4", "messages": [{"role": "user", "content": "2"}]}`)

	// Misses
	cache.Get(payload1)
	cache.Get(payload2)

	// Puts
	cache.Put(payload1, []byte("res1"), 200, 0.10)
	cache.Put(payload2, []byte("res2"), 200, 0.20)

	// Hits
	cache.Get(payload1)
	cache.Get(payload1)
	cache.Get(payload2)

	hits, misses, size, savings := cache.Stats()

	if misses != 2 {
		t.Errorf("Expected 2 misses, got %d", misses)
	}
	if hits != 3 {
		t.Errorf("Expected 3 hits, got %d", hits)
	}
	if size < 2 { // Size might be larger if other keys exist in DB, but should be at least 2
		t.Errorf("Expected size >= 2, got %d", size)
	}
	// Floating point comparison
	if savings < 0.29 || savings > 0.31 {
		t.Errorf("Expected savings ~0.30, got %f", savings)
	}
}

func TestRedisCache_Clear(t *testing.T) {
	cache := setupTestRedis(t)

	payload := []byte(`{"model": "gpt-4", "messages": [{"role": "user", "content": "clear_me"}]}`)
	cache.Put(payload, []byte("res"), 200, 0.50)
	
	cache.Clear()

	// Should be a miss after clear
	entry := cache.Get(payload)
	if entry != nil {
		t.Fatal("Expected miss after cache clear, got hit")
	}

	hits, misses, _, savings := cache.Stats()
	if hits != 0 || misses != 0 || savings != 0 {
		t.Errorf("Expected all stats to be 0 after clear, got hits=%d misses=%d savings=%f", hits, misses, savings)
	}
}
