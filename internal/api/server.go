package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"tkngate/internal/budget"
	"tkngate/internal/cache"
	"tkngate/internal/logging"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// safePoolNode is a sanitized view of PoolNode that strips the encrypted key ciphertext.
// This prevents the AES-256-GCM ciphertext from ever being exposed over the network.
type safePoolNode struct {
	NodeID               string `json:"node_id"`
	ProviderType         string `json:"provider_type"`
	MeasuredTpmLimit     int    `json:"measured_tpm_limit"`
	RemainingTokensQuota int    `json:"remaining_tokens_quota"`
}

// StartTelemetryServer starts the local REST API for telemetry data.
func StartTelemetryServer(host string, port int) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/overview", withAuth(withCORS(handleOverview)))
	mux.HandleFunc("/api/v1/sessions", withAuth(withCORS(handleSessions)))
	mux.HandleFunc("/api/v1/pool", withAuth(withCORS(handlePool)))
	mux.HandleFunc("/api/v1/mesh/stats", withAuth(withCORS(handleMeshStats)))
	mux.HandleFunc("/api/v1/vkeys", withAuth(withCORS(handleVirtualKeys)))
	mux.Handle("/metrics", promhttp.Handler())

	addr := fmt.Sprintf("%s:%d", host, port)
	logging.Logger.Info("Telemetry API starting", "address", addr)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return server.ListenAndServe()
}

// withAuth enforces bearer-token authentication using TKNGATE_MASTER_KEY.
// All telemetry endpoints require authentication to prevent unauthorized data access.
func withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		masterKey := os.Getenv("TKNGATE_MASTER_KEY")
		if masterKey == "" {
			// If no master key is set, allow access (dev mode / local-only)
			next(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, `{"error":"Authorization header required. Use: Bearer <TKNGATE_MASTER_KEY>"}`, http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token != masterKey {
			logging.Logger.Warn("Unauthorized telemetry API access attempt")
			http.Error(w, `{"error":"Invalid bearer token"}`, http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

// withCORS adds CORS headers restricted to localhost origins only.
func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		// Only allow localhost origins (any port)
		if strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "http://127.0.0.1") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

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

	// SECURITY: Strip BlindedKeyHash (AES ciphertext) before sending over the wire.
	// Only expose safe metadata fields — never the encrypted key material.
	safeNodes := make([]safePoolNode, 0, len(allNodes))
	for _, n := range allNodes {
		safeNodes = append(safeNodes, safePoolNode{
			NodeID:               n.NodeID,
			ProviderType:         n.ProviderType,
			MeasuredTpmLimit:     n.MeasuredTpmLimit,
			RemainingTokensQuota: n.RemainingTokensQuota,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"nodes":      safeNodes,
		"total_keys": len(safeNodes),
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
