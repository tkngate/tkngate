package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"tkngate/internal/logging"
)

// CacheEntry stores a cached API response along with metadata.
type CacheEntry struct {
	ResponseBody []byte
	StatusCode   int
	Headers      map[string]string
	CachedAt     time.Time
	HitCount     int
}

// Cache interface abstracts semantic caching engines.
type Cache interface {
	Get(payload []byte) *CacheEntry
	Put(payload []byte, responseBody []byte, statusCode int, estimatedCost float64)
	Stats() (hits int64, misses int64, size int, savingsUSD float64)
	Clear() error
}

// InMemoryCache is a thread-safe in-memory LRU cache keyed by a SHA-256
// hash of the normalised request payload (model + messages).
type InMemoryCache struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry
	maxSize int
	ttl     time.Duration
	hits    int64
	misses  int64
	savings float64 // estimated USD saved
}

// GlobalCache is the singleton cache instance initialised at boot.
var GlobalCache Cache

// InitCache creates the global semantic cache.
func InitCache(maxSize int, ttlSeconds int, redisURI string) {
	if maxSize <= 0 {
		maxSize = 512
	}
	if ttlSeconds <= 0 {
		ttlSeconds = 300 // 5 minutes default
	}

	if redisURI != "" {
		GlobalCache = NewRedisCache(redisURI, ttlSeconds)
		logging.Logger.Info("Distributed Semantic Redis cache initialised", "uri", redisURI, "ttl_seconds", ttlSeconds)
		return
	}

	GlobalCache = &InMemoryCache{
		entries: make(map[string]*CacheEntry),
		maxSize: maxSize,
		ttl:     time.Duration(ttlSeconds) * time.Second,
	}
	logging.Logger.Info("In-Memory Semantic cache initialised", "max_entries", maxSize, "ttl_seconds", ttlSeconds)
}

// computeKey generates a deterministic SHA-256 cache key from the request
// payload.  We hash only the "model" and "messages" fields so that
// identical prompts with different metadata (temperature, etc.) still hit
// the cache.  This is intentional: the user is paying for content, not for
// sampling variance.
func computeKey(payload []byte) string {
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		// Fallback: hash the entire payload.
		h := sha256.Sum256(payload)
		return hex.EncodeToString(h[:])
	}

	// Extract only the semantically meaningful fields.
	canonical := map[string]interface{}{}
	if m, ok := data["model"]; ok {
		canonical["model"] = m
	}
	if m, ok := data["messages"]; ok {
		canonical["messages"] = m
	}

	canonicalBytes, err := json.Marshal(canonical)
	if err != nil {
		h := sha256.Sum256(payload)
		return hex.EncodeToString(h[:])
	}

	h := sha256.Sum256(canonicalBytes)
	return hex.EncodeToString(h[:])
}

// Get looks up a cache entry.  Returns nil on miss or expiry.
func (c *InMemoryCache) Get(payload []byte) *CacheEntry {
	key := computeKey(payload)

	c.mu.RLock()
	entry, exists := c.entries[key]
	c.mu.RUnlock()

	if !exists {
		c.mu.Lock()
		c.misses++
		c.mu.Unlock()
		return nil
	}

	// TTL check
	if time.Since(entry.CachedAt) > c.ttl {
		c.mu.Lock()
		delete(c.entries, key)
		c.misses++
		c.mu.Unlock()
		return nil
	}

	c.mu.Lock()
	entry.HitCount++
	c.hits++
	c.mu.Unlock()

	return entry
}

// Put stores a response in the cache.
func (c *InMemoryCache) Put(payload []byte, responseBody []byte, statusCode int, estimatedCost float64) {
	key := computeKey(payload)

	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict oldest if at capacity.
	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	c.entries[key] = &CacheEntry{
		ResponseBody: responseBody,
		StatusCode:   statusCode,
		Headers:      map[string]string{"X-Tkngate-Cache": "STORED"},
		CachedAt:     time.Now(),
		HitCount:     0,
	}
	c.savings += estimatedCost
}

// evictOldest removes the entry with the oldest CachedAt timestamp.
// Must be called while holding c.mu write lock.
func (c *InMemoryCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for k, v := range c.entries {
		if oldestKey == "" || v.CachedAt.Before(oldestTime) {
			oldestKey = k
			oldestTime = v.CachedAt
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

// Stats returns cache statistics.
func (c *InMemoryCache) Stats() (hits int64, misses int64, size int, savingsUSD float64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hits, c.misses, len(c.entries), c.savings
}

// Clear flushes all entries from the in-memory cache.
func (c *InMemoryCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Reset the map and counters (except hits/misses to keep historical stats, but we reset them anyway for a clean slate)
	c.entries = make(map[string]*CacheEntry)
	c.hits = 0
	c.misses = 0
	c.savings = 0
	
	return nil
}
