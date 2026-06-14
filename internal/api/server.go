package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"tkngate/internal/budget"
	"tkngate/internal/cache"
	"tkngate/internal/logging"
)

// StartTelemetryServer starts the local REST API for telemetry data.
func StartTelemetryServer(host string, port int) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/overview", withCORS(handleOverview))
	mux.HandleFunc("/api/v1/sessions", withCORS(handleSessions))
	mux.HandleFunc("/api/v1/pool", withCORS(handlePool))
	mux.HandleFunc("/api/v1/mesh/stats", withCORS(handleMeshStats))
	mux.HandleFunc("/api/v1/vkeys", withCORS(handleVirtualKeys))

	addr := fmt.Sprintf("%s:%d", host, port)
	logging.Logger.Info("Telemetry API starting", "address", addr)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return server.ListenAndServe()
}

// withCORS adds open CORS headers so local dashboards can fetch the data.
func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func handleOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	totalSpent, _ := budget.GlobalLedger.GetTotalSpend()
	txCount, _ := budget.GlobalLedger.GetTransactionCount()
	
	status, _ := budget.CheckBudget()

	var cacheHits, cacheMisses int64
	var cacheSize int
	var cacheSavings float64

	if cache.GlobalCache != nil {
		cacheHits, cacheMisses, cacheSize, cacheSavings = cache.GlobalCache.Stats()
	}

	resp := map[string]interface{}{
		"total_spent_usd": totalSpent,
		"global_limit":    status.LimitUSD,
		"global_zone":     status.Zone,
		"total_requests":  txCount,
		"cache_stats": map[string]interface{}{
			"hits":    cacheHits,
			"misses":  cacheMisses,
			"entries": cacheSize,
			"savings": cacheSavings,
		},
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessions, err := budget.GlobalLedger.GetSessions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sessions": sessions,
	})
}

func handlePool(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// For the dashboard, we might want to see both openai and anthropic nodes.
	openaiNodes, _ := budget.GlobalLedger.GetPoolNodes("openai")
	anthropicNodes, _ := budget.GlobalLedger.GetPoolNodes("anthropic")

	var allNodes []budget.PoolNode
	allNodes = append(allNodes, openaiNodes...)
	allNodes = append(allNodes, anthropicNodes...)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"nodes":      allNodes,
		"total_keys": len(allNodes),
	})
}

func handleMeshStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	providers := []string{"openai", "anthropic", "deepseek", "kimi", "groq"}
	var allNodes []budget.PoolNode
	
	for _, p := range providers {
		nodes, _ := budget.GlobalLedger.GetPoolNodes(p)
		allNodes = append(allNodes, nodes...)
	}

	var totalCapacity int
	activeNodes := 0

	for _, node := range allNodes {
		if node.RemainingTokensQuota > 0 {
			totalCapacity += node.RemainingTokensQuota
			activeNodes++
		}
	}

	health := "HEALTHY"
	if activeNodes == 0 {
		health = "CRITICAL - NO CAPACITY"
	} else if totalCapacity < 50000 {
		health = "DEGRADED"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_donated_capacity": totalCapacity,
		"active_nodes":           activeNodes,
		"network_health":         health,
		"timestamp":              time.Now(),
	})
}

func handleVirtualKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	keys, err := budget.GlobalLedger.GetVirtualKeys()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"virtual_keys": keys,
	})
}
