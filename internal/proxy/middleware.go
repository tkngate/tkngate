package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"tkngate/internal/auth"
	"tkngate/internal/budget"
	"tkngate/internal/cache"
	"tkngate/internal/compressor"
	"tkngate/internal/config"
	"tkngate/internal/logging"
	"tkngate/internal/pool"
	"tkngate/internal/tokenizer"
	"tkngate/internal/waf"
)

// RoundTripper middleware that captures request/response to enforce budget and count tokens
type proxyTransport struct {
	Transport http.RoundTripper
	Counter   *tokenizer.Counter
}

func (t *proxyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	pathParts := strings.SplitN(strings.TrimPrefix(req.URL.Path, "/"), "/", 2)
	provider := "unknown"
	if len(pathParts) > 0 {
		provider = pathParts[0]
	}

	// 1. BUDGET GUARD (Global)
	status, err := budget.CheckBudget()
	if err != nil {
		logging.Logger.Error("budget check failed", "error", err)
	} else if status.Zone == budget.ZoneRed {
		logging.Logger.Error("Request blocked by global budget circuit breaker", "provider", provider)
		return blockRequest("Global Token Budget Exhausted")
	}

	// v1.3.0: VIRTUAL KEY AUTHENTICATION
	// Clients must provide `Authorization: Bearer tkngate-sk-...`
	authHeader := req.Header.Get("Authorization")
	var authenticatedKeyHash string
	var sessionID string
	sessionZone := budget.ZoneGreen
	
	if strings.HasPrefix(authHeader, "Bearer tkngate-sk-") {
		virtualKey := strings.TrimPrefix(authHeader, "Bearer ")
		
		keys, err := budget.GlobalLedger.GetVirtualKeys()
		if err == nil {
			for _, k := range keys {
				if auth.VerifyKey(virtualKey, k.KeyHash) {
					authenticatedKeyHash = k.KeyHash
					sessionID = k.Name // Map the virtual key name to the session ID for legacy tracking
					break
				}
			}
		}

		if authenticatedKeyHash == "" {
			logging.Logger.Error("Request blocked: Invalid Virtual Key provided", "provider", provider)
			return blockRequest("401 Unauthorized: Invalid Tkngate Virtual Key")
		}

		// Check the Virtual Key budget
		sStatus, err := budget.CheckVirtualKeyBudget(authenticatedKeyHash)
		if err != nil {
			logging.Logger.Error("virtual key budget check failed", "error", err)
		} else {
			sessionZone = sStatus.Zone
			if sessionZone == budget.ZoneRed {
				logging.Logger.Error("Request blocked by virtual key budget circuit breaker", "key", sessionID)
				return blockRequest(fmt.Sprintf("Virtual Key Budget Exhausted: %s", sessionID))
			}
		}
	} else {
		// Fallback to legacy Session ID behavior if Virtual Keys are not provided
		sessionID = req.Header.Get("X-Tkngate-Session-ID")
		if sessionID != "" {
			sStatus, err := budget.CheckSessionBudget(sessionID)
			if err != nil {
				logging.Logger.Error("session budget check failed", "error", err)
			} else {
				sessionZone = sStatus.Zone
				if sessionZone == budget.ZoneRed {
					logging.Logger.Error("Request blocked by session budget circuit breaker", "session", sessionID)
					return blockRequest(fmt.Sprintf("Session Token Budget Exhausted: %s", sessionID))
				}
			}
		}
	}

	// 2. CAPTURE INPUT (For token counting & rewriting)
	var inputBody []byte
	var reqModel string
	if req.Body != nil {
		inputBody, _ = captureBody(req)

		// 2.1. AI-WAF: Prompt Injection Firewall
		if err := waf.DetectJailbreak(inputBody); err != nil {
			logging.Logger.Error("WAF Blocked Request", "reason", err, "session", sessionID)
			return blockRequest(fmt.Sprintf("WAF Blocked Request: %v", err))
		}

		// 2.2. AI-WAF: Data Loss Prevention (PII Redaction)
		sanitizedBody := waf.RedactPII(inputBody)
		if len(sanitizedBody) != len(inputBody) || !bytes.Equal(sanitizedBody, inputBody) {
			logging.Logger.Info("WAF DLP Engine redacted sensitive PII from payload", "session", sessionID)
			inputBody = sanitizedBody
			req.Body = io.NopCloser(bytes.NewBuffer(inputBody))
			req.ContentLength = int64(len(inputBody))
		}

		reqModel = extractModel(inputBody)
		
		// Model Downgrade in Amber Zone
		if sessionZone == budget.ZoneAmber {
			fallback := config.Cfg.Budget.FallbackModel
			if fallback == "" {
				fallback = "gpt-4o-mini"
			}
			if reqModel != fallback {
				logging.Logger.Info("Session in Amber zone, downgrading model", "from", reqModel, "to", fallback, "session", sessionID)
				inputBody = replaceModel(inputBody, fallback)
				reqModel = fallback
				
				// Update req.Body and ContentLength to the new modified body
				req.Body = io.NopCloser(bytes.NewBuffer(inputBody))
				req.ContentLength = int64(len(inputBody))
			}
		}

		// 2.5. CONTEXT COMPRESSOR
		if config.Cfg.Compressor.Enabled {
			inTokens := t.Counter.Count(string(inputBody), reqModel)
			if inTokens > config.Cfg.Compressor.SoftTokenLimit {
				logging.Logger.Info("Payload exceeds soft limit, running context compressor", "tokens", inTokens, "limit", config.Cfg.Compressor.SoftTokenLimit)
				
				compressedBody := compressPayload(inputBody)
				if len(compressedBody) < len(inputBody) {
					logging.Logger.Info("Context compression successful", "original_bytes", len(inputBody), "new_bytes", len(compressedBody))
					inputBody = compressedBody
					req.Body = io.NopCloser(bytes.NewBuffer(inputBody))
					req.ContentLength = int64(len(inputBody))
				}
			}
		}
	}

	// 2.8. SEMANTIC CACHE CHECK
	if cache.GlobalCache != nil && len(inputBody) > 0 {
		if hit := cache.GlobalCache.Get(inputBody); hit != nil {
			logging.Logger.Info("Semantic cache HIT — returning cached response", "provider", provider, "cost_saved", "$0.00")
			return &http.Response{
				Status:        fmt.Sprintf("%d OK", hit.StatusCode),
				StatusCode:    hit.StatusCode,
				Proto:         "HTTP/1.1",
				ProtoMajor:    1,
				ProtoMinor:    1,
				Body:          io.NopCloser(bytes.NewBuffer(hit.ResponseBody)),
				ContentLength: int64(len(hit.ResponseBody)),
				Header:        http.Header{"X-Tkngate-Cache": []string{"HIT"}},
			}, nil
		}
	}

	// 3. EXECUTE REQUEST WITH AUTO-RETRY (v0.7.0)
	maxRetries := 3
	var res *http.Response
	var roundTripErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Reconstruct body safely for each attempt
		if len(inputBody) > 0 {
			req.Body = io.NopCloser(bytes.NewBuffer(inputBody))
			req.ContentLength = int64(len(inputBody))
		}

		if pool.GlobalDRR != nil {
			estimatedTokens := 1000
			if len(inputBody) > 0 {
				estimatedTokens = t.Counter.Count(string(inputBody), reqModel)
			}
			dynamicKey, err := pool.GlobalDRR.GetNextKey(provider, sessionID, estimatedTokens)
			if err == nil && dynamicKey != "" {
				if provider == "anthropic" {
					req.Header.Set("x-api-key", dynamicKey)
				} else {
					req.Header.Set("Authorization", "Bearer "+dynamicKey)
				}
				if attempt > 0 {
					logging.Logger.Info("DRR Engine attached new key for retry", "provider", provider, "attempt", attempt+1)
				} else {
					logging.Logger.Info("DRR Engine rotated key", "provider", provider)
				}
			}
		}

		res, roundTripErr = t.Transport.RoundTrip(req)
		if roundTripErr != nil {
			break // Break on hard network errors
		}

		// Intercept 429 Rate Limits
		if res.StatusCode == http.StatusTooManyRequests {
			if attempt < maxRetries {
				logging.Logger.Warn("Intercepted 429 Rate Limit. Auto-Retrying with new key...", "provider", provider, "attempt", attempt+1)
				// Drain and close the failing response to free the connection
				if res.Body != nil {
					io.Copy(io.Discard, res.Body)
					res.Body.Close()
				}
				continue
			} else {
				logging.Logger.Error("Max retries exhausted for 429 Rate Limit", "provider", provider)
			}
		}

		// v0.9.0: Universal API Router (Multi-AI Fallback)
		// Intercept severe upstream outages (500, 502, 503) and fallback to another provider seamlessly.
		// ONLY do this if the request is for chat completions, as we cannot safely translate embeddings/audio yet.
		if res.StatusCode >= 500 && attempt < maxRetries && strings.HasSuffix(req.URL.Path, "/chat/completions") {
			fallbackProvider := config.Cfg.Budget.FallbackProvider
			if fallbackProvider == "" {
				fallbackProvider = "deepseek" // Default fallback to deepseek if not configured
			}
			
			if provider != fallbackProvider {
				logging.Logger.Warn("Intercepted severe upstream outage. Engaging Universal API Router Fallback...", "from", provider, "to", fallbackProvider, "status", res.StatusCode)
				
				// 1. Swap Provider for the next loop iteration (so DRR grabs the new provider's key)
				provider = fallbackProvider
				
				// 2. Rewrite HTTP Destination and JSON payload model
				switch provider {
				case "anthropic":
					req.URL.Host = "api.anthropic.com"
					req.Host = "api.anthropic.com"
					req.URL.Path = "/v1/messages"
					inputBody = replaceModel(inputBody, "claude-sonnet-4-20250514")
					// Anthropic uses x-api-key header instead of Bearer token
					req.Header.Del("Authorization")
					if provCfg, ok := config.Cfg.Providers["anthropic"]; ok {
						req.Header.Set("x-api-key", provCfg.APIKey)
						req.Header.Set("anthropic-version", "2023-06-01")
					}
				case "openai":
					req.URL.Host = "api.openai.com"
					req.Host = "api.openai.com"
					req.URL.Path = "/v1/chat/completions"
					inputBody = replaceModel(inputBody, "gpt-4o")
				case "deepseek":
					req.URL.Host = "api.deepseek.com"
					req.Host = "api.deepseek.com"
					req.URL.Path = "/v1/chat/completions"
					inputBody = replaceModel(inputBody, "deepseek-chat")
				case "kimi":
					req.URL.Host = "api.moonshot.cn"
					req.Host = "api.moonshot.cn"
					req.URL.Path = "/v1/chat/completions"
					inputBody = replaceModel(inputBody, "moonshot-v1-8k")
				case "groq":
					req.URL.Host = "api.groq.com"
					req.Host = "api.groq.com"
					req.URL.Path = "/openai/v1/chat/completions"
					inputBody = replaceModel(inputBody, "llama3-8b-8192")
				}
				
				// Drain failing response
				if res.Body != nil {
					io.Copy(io.Discard, res.Body)
					res.Body.Close()
				}
				continue
			}
		}

		// Success or other terminal error code
		break
	}

	if roundTripErr != nil {
		return res, roundTripErr
	}

	// 4. HANDLE STREAMING vs BUFFERED RESPONSES
	// v1.2.0: If the response is an SSE stream, wrap it with our real-time token counter.
	// The streaming interceptor will count tokens per chunk and enforce budget limits mid-stream.
	if isStreamingResponse(res) {
		logging.Logger.Info("SSE streaming response detected — engaging real-time token interceptor", "provider", provider, "model", reqModel)
		wrapStreamingResponse(res, t.Counter, reqModel, provider, sessionID, authenticatedKeyHash)

		// Record input tokens only (output tokens are counted by the stream interceptor)
		go func() {
			if len(inputBody) > 0 {
				inTokens := t.Counter.Count(string(inputBody), reqModel)
				cost := tokenizer.EstimateCost(provider, reqModel, inTokens, 0)
				tx := budget.Transaction{
					SessionID:        sessionID,
					VirtualKeyHash:   authenticatedKeyHash,
					Provider:         provider,
					Model:            reqModel,
					InputTokens:      inTokens,
					OutputTokens:     0,
					EstimatedCostUSD: cost,
				}
				if err := budget.GlobalLedger.RecordTransaction(tx); err != nil {
					logging.Logger.Error("failed to record input transaction for stream", "error", err)
				}
			}
			latency := time.Since(start)
			logging.Logger.Info("SSE stream initiated",
				"provider", provider,
				"model", reqModel,
				"latency_ms", latency.Milliseconds(),
				"zone", status.Zone)
		}()
	} else {
		// Standard (non-streaming) response handling
		// 4. CAPTURE OUTPUT (For token counting)
		var outputBody []byte
		if res.Body != nil {
			// Limit response body to 10MB to prevent DoS via unbounded memory allocation
			outputBody, _ = io.ReadAll(io.LimitReader(res.Body, 10*1024*1024))
			res.Body = io.NopCloser(bytes.NewBuffer(outputBody))
		}

		// 4.5. STORE IN SEMANTIC CACHE (only on success)
		if cache.GlobalCache != nil && res.StatusCode >= 200 && res.StatusCode < 300 && len(inputBody) > 0 {
			cost := tokenizer.EstimateCost(provider, reqModel, t.Counter.Count(string(inputBody), reqModel), t.Counter.Count(string(outputBody), reqModel))
			cache.GlobalCache.Put(inputBody, outputBody, res.StatusCode, cost)
			res.Header.Set("X-Tkngate-Cache", "MISS")
		}

		// 5. TOKEN COUNTING & LEDGER UPDATE
		go func() {
			inTokens := 0
			outTokens := 0

			if len(inputBody) > 0 {
				inTokens = t.Counter.Count(string(inputBody), reqModel)
			}
			if len(outputBody) > 0 {
				outTokens = t.Counter.Count(string(outputBody), reqModel)
			}

			cost := tokenizer.EstimateCost(provider, reqModel, inTokens, outTokens)

			tx := budget.Transaction{
				SessionID:        sessionID,
				VirtualKeyHash:   authenticatedKeyHash,
				Provider:         provider,
				Model:            reqModel,
				InputTokens:      inTokens,
				OutputTokens:     outTokens,
				EstimatedCostUSD: cost,
			}

			if err := budget.GlobalLedger.RecordTransaction(tx); err != nil {
				logging.Logger.Error("failed to record transaction", "error", err)
			}

			latency := time.Since(start)
			logging.Logger.Info("Request handled",
				"provider", provider,
				"model", reqModel,
				"cost_usd", cost,
				"latency_ms", latency.Milliseconds(),
				"zone", status.Zone)
		}()
	}

	// 6. SHADOW MODE (v1.1.0) — Fire-and-forget mirror to a shadow provider
	if config.Cfg.Shadow.Enabled && len(inputBody) > 0 && strings.HasSuffix(req.URL.Path, "/chat/completions") {
		if rand.Float64() < config.Cfg.Shadow.TrafficFraction {
			go fireShadowRequest(inputBody, config.Cfg.Shadow.TargetProvider, config.Cfg.Shadow.TargetModel)
		}
	}

	return res, nil
}

func fireShadowRequest(body []byte, provider string, model string) {
	defer func() {
		if r := recover(); r != nil {
			logging.Logger.Error("Shadow mode panic recovered", "error", r)
		}
	}()

	providerCfg, ok := config.Cfg.Providers[provider]
	if !ok {
		logging.Logger.Warn("Shadow mode: target provider not configured", "provider", provider)
		return
	}

	shadowBody := replaceModel(body, model)

	baseURL := strings.TrimSuffix(providerCfg.BaseURL, "/")
	shadowURL := baseURL + "/chat/completions"

	shadowReq, err := http.NewRequest("POST", shadowURL, bytes.NewBuffer(shadowBody))
	if err != nil {
		logging.Logger.Error("Shadow mode: failed to create request", "error", err)
		return
	}

	shadowReq.Header.Set("Content-Type", "application/json")
	if provider == "anthropic" {
		shadowReq.Header.Set("x-api-key", providerCfg.APIKey)
		shadowReq.Header.Set("anthropic-version", "2023-06-01")
	} else {
		shadowReq.Header.Set("Authorization", "Bearer "+providerCfg.APIKey)
	}

	start := time.Now()
	client := &http.Client{Timeout: 30 * time.Second}
	shadowRes, err := client.Do(shadowReq)
	latency := time.Since(start)

	if err != nil {
		logging.Logger.Warn("Shadow mode: request failed", "provider", provider, "error", err)
		return
	}
	defer shadowRes.Body.Close()

	logging.Logger.Info("Shadow mode: mirror complete",
		"provider", provider,
		"model", model,
		"status", shadowRes.StatusCode,
		"latency_ms", latency.Milliseconds())
}

func blockRequest(message string) (*http.Response, error) {
	resp := map[string]string{"error": message}
	body, _ := json.Marshal(resp)
	return &http.Response{
		Status:        "429 Too Many Requests",
		StatusCode:    429,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Body:          io.NopCloser(bytes.NewBuffer(body)),
		ContentLength: int64(len(body)),
		Header:        make(http.Header),
	}, nil
}

// Simple extractor for the "model" field in standard JSON payloads
func extractModel(payload []byte) string {
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err == nil {
		if m, ok := data["model"].(string); ok {
			return m
		}
	}
	return "unknown"
}

func replaceModel(payload []byte, newModel string) []byte {
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err == nil {
		data["model"] = newModel
		if modified, err := json.Marshal(data); err == nil {
			return modified
		}
	}
	return payload
}

func compressPayload(payload []byte) []byte {
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return payload
	}

	messages, ok := data["messages"].([]interface{})
	if !ok {
		return payload
	}

	modified := false
	for i, msg := range messages {
		msgMap, ok := msg.(map[string]interface{})
		if !ok {
			continue
		}
		
		content, ok := msgMap["content"].(string)
		if ok && len(content) > 100 { 
			compressed := compressor.Compress(content)
			if len(compressed) < len(content) {
				msgMap["content"] = compressed
				messages[i] = msgMap
				modified = true
			}
		}
	}

	if modified {
		data["messages"] = messages
		if newPayload, err := json.Marshal(data); err == nil {
			return newPayload
		}
	}

	return payload
}
