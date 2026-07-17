package cluster

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"tkngate/internal/config"
	"tkngate/internal/logging"
)

var (
	RedisClient *redis.Client
	IsLeader    bool
)

func InitCluster() error {
	if !config.Cfg.Cluster.Enabled {
		return nil
	}

	opt, err := redis.ParseURL(config.Cfg.Cluster.RedisURI)
	if err != nil {
		return err
	}

	RedisClient = redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := RedisClient.Ping(ctx).Err(); err != nil {
		return err
	}

	logging.Logger.Info("Cluster Mode enabled, connected to Redis", "node_id", config.Cfg.Cluster.NodeID)

	go electLeader()

	return nil
}

func electLeader() {
	nodeID := config.Cfg.Cluster.NodeID
	lockKey := "tkngate:leader"
	
	// Leader lock TTL is 10 seconds. We try to renew it every 5 seconds.
	ttl := 10 * time.Second
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		
		// Attempt to acquire or renew the lock
		// If the lock doesn't exist, we set it to our node ID.
		// If it exists but it's ours, we extend the TTL.
		
		// We can do this atomically with a small Lua script
		script := `
			if redis.call("get", KEYS[1]) == ARGV[1] then
				return redis.call("pexpire", KEYS[1], ARGV[2])
			elseif redis.call("set", KEYS[1], ARGV[1], "NX", "PX", ARGV[2]) then
				return 1
			else
				return 0
			end
		`
		
		res, err := RedisClient.Eval(ctx, script, []string{lockKey}, nodeID, ttl.Milliseconds()).Result()
		if err != nil {
			logging.Logger.Error("Leader election error", "error", err)
			IsLeader = false
			continue
		}

		success, ok := res.(int64)
		if ok && success == 1 {
			if !IsLeader {
				logging.Logger.Info("Node acquired Cluster Leadership", "node_id", nodeID)
				IsLeader = true
			}
		} else {
			if IsLeader {
				logging.Logger.Warn("Node lost Cluster Leadership", "node_id", nodeID)
				IsLeader = false
			}
		}
	}
}
