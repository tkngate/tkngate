package api

import (
	cryptoRand "crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"tkngate/internal/budget"
	"tkngate/internal/cache"
	"tkngate/internal/config"
	"tkngate/internal/crypto"
	"tkngate/internal/logging"
	"tkngate/internal/mesh"
	"tkngate/internal/telemetry"
	"tkngate/internal/validator"
	"tkngate/internal/waf"

	"github.com/google/uuid"
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

	mux.HandleFunc("/api/v1/overview", withCORS(withAuth(handleOverview)))
	mux.HandleFunc("/api/v1/sessions", withCORS(withAuth(handleSessions)))
	mux.HandleFunc("/api/v1/pool", withCORS(withAuth(handlePool)))
	mux.HandleFunc("/api/v1/mesh/stats", withCORS(withAuth(handleMeshStats)))
	mux.HandleFunc("/api/v1/mesh/reputation", withCORS(withAuth(handleMeshReputation)))
	mux.HandleFunc("/api/v1/vkeys", withCORS(withAuth(handleVirtualKeys)))
	mux.HandleFunc("/api/v1/orgs", withCORS(withAuth(handleOrgs)))
	mux.HandleFunc("/api/v1/security", withCORS(withAuth(handleSecurity)))
	mux.HandleFunc("/api/v1/budget/reset", withCORS(withAuth(handleBudgetReset)))
	mux.HandleFunc("/api/v1/cache/clear", withCORS(withAuth(handleCacheClear)))
	mux.HandleFunc("/api/v1/providers", withCORS(withAuth(handleProviders)))
	mux.HandleFunc("/api/v1/providers/test", withCORS(withAuth(handleProvidersTest)))
	mux.HandleFunc("/api/v1/pool/donate", withCORS(withAuth(handlePoolDonate)))
	mux.HandleFunc("/api/v1/config", withCORS(withAuth(handleConfig)))
	mux.HandleFunc("/api/v1/waf/rules", withCORS(withAuth(handleWafRules)))
	mux.Handle("/metrics", promhttp.Handler())

	// Serve the embedded React Dashboard
	subFS, err := fs.Sub(DashboardFS, "ui/dist")
	if err != nil {
		logging.Logger.Error("Failed to load embedded dashboard UI", "error", err)
	} else {
		fileServer := http.FileServer(http.FS(subFS))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// If it's an API request that fell through, return 404
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}
			
			// Try to open the requested file
			f, err := subFS.Open(strings.TrimPrefix(r.URL.Path, "/"))
			if err != nil {
				// File not found, fall back to index.html for React Router SPA
				r.URL.Path = "/"
			} else {
				f.Close()
			}
			fileServer.ServeHTTP(w, r)
		})
	}

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
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
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

func handleSecurity(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"waf_enabled":       config.Cfg.WAF.Enabled,
		"strict_zkp_mode":   config.Cfg.Mesh.StrictZKPMode,
		"dlp_redaction":     config.Cfg.WAF.Enabled,
		"prompt_injection":  config.Cfg.WAF.Enabled,
		"waf_blocks_total":  atomic.LoadInt64(&telemetry.RawWafBlocks),
		"zkp_verified_total": atomic.LoadInt64(&telemetry.RawZkpVerified),
		"zkp_failed_total":  atomic.LoadInt64(&telemetry.RawZkpFailed),
	})
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

func handleMeshReputation(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if mesh.GlobalReputation == nil {
		http.Error(w, `{"error":"mesh reputation system is disabled"}`, http.StatusServiceUnavailable)
		return
	}

	reputations := mesh.GlobalReputation.GetAllReputations()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"reputations": reputations,
		"timestamp":   time.Now(),
	})
}

func handleOrgs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		orgs, err := budget.GlobalLedger.GetOrganizations()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"organizations": orgs,
		})

	case "POST":
		var req struct {
			Name     string  `json:"name"`
			LimitUSD float64 `json:"limit_usd"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
			http.Error(w, `{"error":"name and limit_usd required"}`, http.StatusBadRequest)
			return
		}
		if req.LimitUSD <= 0 {
			req.LimitUSD = 100.0
		}
		if err := budget.GlobalLedger.CreateOrganization(req.Name, req.LimitUSD); err != nil {
			http.Error(w, `{"error":"organization already exists or db error"}`, http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "created",
			"name":   req.Name,
		})

	case "DELETE":
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			http.Error(w, `{"error":"id is required"}`, http.StatusBadRequest)
			return
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, `{"error":"invalid id format"}`, http.StatusBadRequest)
			return
		}
		if err := budget.GlobalLedger.DeleteOrganization(id); err != nil {
			http.Error(w, `{"error":"failed to delete organization"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "deleted",
			"id":     id,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleVirtualKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		keys, err := budget.GlobalLedger.GetVirtualKeys()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"virtual_keys": keys,
		})

	case "POST":
		var req struct {
			Name             string  `json:"name"`
			Budget           float64 `json:"budget_usd"`
			OrgID            int     `json:"org_id"`
			AllowedProviders string  `json:"allowed_providers"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
			http.Error(w, `{"error":"name and budget_usd required"}`, http.StatusBadRequest)
			return
		}
		if req.Budget <= 0 {
			req.Budget = 10.0
		}
		// Generate a secure random key: tkngate-sk-<16 hex bytes>
		buf := make([]byte, 16)
		if _, err := cryptoRand.Read(buf); err != nil {
			http.Error(w, `{"error":"key generation failed"}`, http.StatusInternalServerError)
			return
		}
		plainKey := fmt.Sprintf("tkngate-sk-%x", buf)
		keyHash := fmt.Sprintf("%x", sha256Sum([]byte(plainKey)))
		if err := budget.GlobalLedger.RegisterVirtualKey(keyHash, req.Name, req.Budget, req.OrgID, req.AllowedProviders); err != nil {
			http.Error(w, `{"error":"key already exists or db error"}`, http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"key":        plainKey,
			"name":       req.Name,
			"budget_usd": req.Budget,
			"note":       "Store this key securely — it will not be shown again.",
		})

	case "DELETE":
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
			http.Error(w, `{"error":"name required"}`, http.StatusBadRequest)
			return
		}
		if err := budget.GlobalLedger.RevokeVirtualKey(req.Name); err != nil {
			http.Error(w, `{"error":"key not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "revoked", "name": req.Name})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func sha256Sum(b []byte) []byte {
	h := sha256.Sum256(b)
	return h[:]
}

func handleBudgetReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := budget.GlobalLedger.Reset(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func handleCacheClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if cache.GlobalCache != nil {
		if err := cache.GlobalCache.Clear(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

func handleProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	safeProviders := make(map[string]interface{})
	for name, p := range config.Cfg.Providers {
		masked := "NOT SET"
		if p.APIKey != "" {
			if len(p.APIKey) <= 8 {
				masked = strings.Repeat("•", len(p.APIKey))
			} else {
				masked = p.APIKey[:4] + strings.Repeat("•", len(p.APIKey)-8) + p.APIKey[len(p.APIKey)-4:]
			}
		}
		endpoint := p.BaseURL
		if endpoint == "" {
			endpoint = "(default)"
		}
		safeProviders[name] = map[string]interface{}{
			"default_model": p.DefaultModel,
			"endpoint":      endpoint,
			"key_status":    masked,
			"has_key":       p.APIKey != "",
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"providers": safeProviders,
	})
}

func handleProvidersTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	results := make(map[string]interface{})
	for name, p := range config.Cfg.Providers {
		if p.APIKey == "" && name != "ollama" {
			results[name] = map[string]interface{}{"status": "skipped", "error": "No API key"}
			continue
		}
		
		err := validator.ValidateKey(name, p.APIKey)
		if err != nil {
			results[name] = map[string]interface{}{"status": "failed", "error": err.Error()}
		} else {
			results[name] = map[string]interface{}{"status": "passed", "error": ""}
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": results,
	})
}

func handlePoolDonate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Provider string `json:"provider"`
		Key      string `json:"key"`
		Limit    int    `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Key == "" || req.Provider == "" {
		http.Error(w, `{"error":"provider and key required"}`, http.StatusBadRequest)
		return
	}
	
	if err := validator.ValidateKey(req.Provider, req.Key); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"validation failed: %v"}`, err), http.StatusBadRequest)
		return
	}
	
	encryptedKey, err := crypto.Encrypt(req.Key)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"encryption failed: %v"}`, err), http.StatusInternalServerError)
		return
	}
	
	node := budget.PoolNode{
		NodeID:               uuid.New().String(),
		ProviderType:         req.Provider,
		BlindedKeyHash:       encryptedKey,
		MeasuredTpmLimit:     req.Limit,
		RemainingTokensQuota: req.Limit,
	}
	
	if err := budget.GlobalLedger.AddPoolNode(node); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to save node: %v"}`, err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "donated",
		"node_id": node.NodeID,
	})
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == "PUT" {
		var updated config.Config
		if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to decode config: %v"}`, err), http.StatusBadRequest)
			return
		}

		// Security (SSRF/DoS protection): Do not allow API updates to sensitive infrastructure settings
		updated.Cloud = config.Cfg.Cloud
		updated.Server = config.Cfg.Server
		updated.Telemetry = config.Cfg.Telemetry

		// Validate any newly provided API keys before saving
		for providerName, providerConfig := range updated.Providers {
			key := providerConfig.APIKey
			if key != "" && key != "[REDACTED]" {
				if err := validator.ValidateKey(providerName, key); err != nil {
					http.Error(w, fmt.Sprintf(`{"error":"Invalid API Key for %s: %v"}`, providerName, err), http.StatusBadRequest)
					return
				}
			}
		}

		if err := config.SaveConfig(updated); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to save config: %v"}`, err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	safeConfig := config.Cfg
	
	// Deep copy the Providers map to avoid mutating the global state!
	safeProviders := make(map[string]config.ProviderConfig)
	for k, p := range config.Cfg.Providers {
		if p.APIKey != "" {
			p.APIKey = "[REDACTED]"
		}
		safeProviders[k] = p
	}
	safeConfig.Providers = safeProviders
	if safeConfig.Cloud.Secret != "" {
		safeConfig.Cloud.Secret = "[REDACTED]"
	}
	if safeConfig.Mesh.ModerationAPIKey != "" {
		safeConfig.Mesh.ModerationAPIKey = "[REDACTED]"
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(safeConfig)
}

func handleWafRules(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"injections": waf.KnownPromptInjections,
		"blocklist": config.Cfg.WAF.Blocklist,
	})
}
