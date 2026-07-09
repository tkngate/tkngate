package cache

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"tkngate/internal/logging"
)

type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisCache(uri string, ttlSeconds int) *RedisCache {
	opt, err := redis.ParseURL(uri)
	if err != nil {
		logging.Logger.Error("Failed to parse Redis URI", "error", err, "uri", uri)
		// Fallback to simple localhost default if parse fails
		opt = &redis.Options{
			Addr: "localhost:6379",
		}
	}

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		logging.Logger.Error("Failed to connect to Redis", "error", err)
	}

	return &RedisCache{
		client: client,
		ttl:    time.Duration(ttlSeconds) * time.Second,
	}
}

func (r *RedisCache) Get(payload []byte) *CacheEntry {
	key := computeKey(payload)
	ctx := context.Background()

	val, err := r.client.Get(ctx, "tkngate:cache:"+key).Result()
	if err == redis.Nil {
		r.client.Incr(ctx, "tkngate:stats:misses")
		return nil
	} else if err != nil {
		logging.Logger.Error("Redis GET error", "error", err)
		return nil
	}

	var entry CacheEntry
	if err := json.Unmarshal([]byte(val), &entry); err != nil {
		logging.Logger.Error("Failed to unmarshal cache entry from Redis", "error", err)
		return nil
	}

	r.client.Incr(ctx, "tkngate:stats:hits")
	return &entry
}

func (r *RedisCache) Put(payload []byte, responseBody []byte, statusCode int, estimatedCost float64) {
	key := computeKey(payload)
	ctx := context.Background()

	entry := CacheEntry{
		ResponseBody: responseBody,
		StatusCode:   statusCode,
		Headers:      map[string]string{"X-Tkngate-Cache": "STORED-REDIS"},
		CachedAt:     time.Now(),
		HitCount:     0,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		logging.Logger.Error("Failed to marshal cache entry for Redis", "error", err)
		return
	}

	err = r.client.Set(ctx, "tkngate:cache:"+key, data, r.ttl).Err()
	if err != nil {
		logging.Logger.Error("Redis SET error", "error", err)
		return
	}

	// Update savings stat using INCRBYFLOAT
	r.client.IncrByFloat(ctx, "tkngate:stats:savings", estimatedCost)
}

func (r *RedisCache) Stats() (hits int64, misses int64, size int, savingsUSD float64) {
	ctx := context.Background()

	// Fetch hits
	if h, err := r.client.Get(ctx, "tkngate:stats:hits").Result(); err == nil {
		hits, _ = strconv.ParseInt(h, 10, 64)
	}

	// Fetch misses
	if m, err := r.client.Get(ctx, "tkngate:stats:misses").Result(); err == nil {
		misses, _ = strconv.ParseInt(m, 10, 64)
	}

	// Fetch savings
	if s, err := r.client.Get(ctx, "tkngate:stats:savings").Result(); err == nil {
		savingsUSD, _ = strconv.ParseFloat(s, 64)
	}

	// For size, we can fetch DBSIZE or just scan keys starting with tkngate:cache:
	// For performance, we'll just use DBSIZE as a global approximation, or 0.
	size = int(r.client.DBSize(ctx).Val())

	return hits, misses, size, savingsUSD
}

// Clear flushes all tkngate related keys from Redis.
func (r *RedisCache) Clear() error {
	ctx := context.Background()
	
	// Delete stats
	r.client.Del(ctx, "tkngate:stats:hits", "tkngate:stats:misses", "tkngate:stats:savings")
	
	// Delete all tkngate:cache:* keys
	// This uses a scan loop to avoid blocking Redis with a KEYS command
	iter := r.client.Scan(ctx, 0, "tkngate:cache:*", 0).Iterator()
	for iter.Next(ctx) {
		r.client.Del(ctx, iter.Val())
	}
	
	if err := iter.Err(); err != nil {
		logging.Logger.Error("Redis scan error during Clear", "error", err)
		return err
	}
	
	return nil
}
