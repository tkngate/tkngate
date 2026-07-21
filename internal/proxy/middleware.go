package proxy

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"tkngate/internal/auth"
	"tkngate/internal/budget"
	"tkngate/internal/cache"
	"tkngate/internal/cloud"
	"tkngate/internal/compressor"
	"tkngate/internal/config"
	"tkngate/internal/limiter"
	"tkngate/internal/logging"
	"tkngate/internal/mesh"
	"tkngate/internal/p2p"
	"tkngate/internal/pool"
	"tkngate/internal/telemetry"
	"tkngate/internal/tokenizer"
	"tkngate/internal/waf"
)

// RoundTripper middleware that captures request/response to enforce budget and count tokens
type proxyTransport struct {
	Transport http.RoundTripper
	Counter   *tokenizer.Counter
}

func (t *proxyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	telemetry.ActiveConnections.Inc()
	defer telemetry.ActiveConnections.Dec()

	start := time.Now()

	provider := req.Header.Get("X-Tkngate-Provider")
	if provider == "" {
		provider = "openai" // fallback
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
	// Director saves the original Auth header to X-Tkngate-Original-Auth before injecting the upstream API key.
	authHeader := req.Header.Get("X-Tkngate-Original-Auth")
	if authHeader == "" {
		authHeader = req.Header.Get("Authorization")
	}

	var authenticatedKeyHash string
	var sessionID string
	var piiFlagged bool
	var wasFailover bool
	sessionZone := budget.ZoneGreen

	if strings.HasPrefix(authHeader, "Bearer tkngate-sk-") {
		virtualKey := strings.TrimPrefix(authHeader, "Bearer ")

		if config.Cfg.Cloud.Enabled {
			// CLOUD MODE: Call Next.js API for validation
			cloudRes, err := cloud.ValidateKey(virtualKey)
			if err != nil || cloudRes == nil {
				logging.Logger.Error("Request blocked: Cloud Validation Failed", "error", err)
				return blockRequest("401 Unauthorized: Invalid Tkngate Virtual Key")
			}
			
			if !cloudRes.Valid {
				logging.Logger.Error("Request blocked by cloud budget circuit breaker", "keyName", cloudRes.KeyName)
				return blockRequest(fmt.Sprintf("Virtual Key Budget Exhausted: %s", cloudRes.KeyName))
			}

			// Validated by Cloud
			authenticatedKeyHash = cloudRes.KeyID // We use the database ID as the identifier in cloud mode
			sessionID = cloudRes.KeyName
			
			switch cloudRes.Zone {
			case "RED":
				sessionZone = budget.ZoneRed
			case "AMBER":
				sessionZone = budget.ZoneAmber
			default:
				sessionZone = budget.ZoneGreen
			}
			
		} else {
			// LOCAL MODE: Use SQLite Ledger
			keys, err := budget.GlobalLedger.GetVirtualKeys()
			if err == nil {
				for _, k := range keys {
					// Support both SHA-256 (dashboard API) and bcrypt (CLI) key hashes
					sha := sha256.Sum256([]byte(virtualKey))
					shaHex := hex.EncodeToString(sha[:])
					if shaHex == k.KeyHash || auth.VerifyKey(virtualKey, k.KeyHash) {
						authenticatedKeyHash = k.KeyHash
						sessionID = k.Name // Map the virtual key name to the session ID for legacy tracking
						
						// RBAC Check (v2.0.0)
						if k.AllowedProviders != "" {
							allowedList := strings.Split(k.AllowedProviders, ",")
							isAllowed := false
							for _, allowed := range allowedList {
								if strings.TrimSpace(allowed) == provider {
									isAllowed = true
									break
								}
							}
							if !isAllowed {
								logging.Logger.Error("Request blocked by RBAC: Provider not allowed for this key", "key", k.Name, "requested", provider)
								return blockRequest(fmt.Sprintf("403 Forbidden: RBAC Policy Violation. Key '%s' cannot access provider '%s'", k.Name, provider))
							}
						}

						// Org Budget Check (v2.0.0)
						if k.OrgID > 0 {
							orgs, orgErr := budget.GlobalLedger.GetOrganizations()
							if orgErr == nil {
								for _, o := range orgs {
									if o.ID == k.OrgID {
										if o.ConsumedUSD >= o.BudgetLimitUSD {
											logging.Logger.Error("Request blocked: Organization budget exhausted", "org", o.Name)
											return blockRequest(fmt.Sprintf("403 Forbidden: Organization Budget Exhausted (%s)", o.Name))
										}
										break
									}
								}
							}
						}

						if k.ConsumedBudget >= k.AllocatedBudget {
							sessionZone = budget.ZoneRed
						} else if k.ConsumedBudget >= k.AllocatedBudget*0.75 {
							sessionZone = budget.ZoneAmber
						}
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

	// v1.5.0: PRE-EMPTIVE RATE LIMITING (Token Bucket)
	identifier := authenticatedKeyHash
	if identifier == "" {
		identifier = sessionID
	}
	if identifier != "" {
		if !limiter.GlobalManager.Allow(identifier) {
			logging.Logger.Warn("Request blocked: Rate limit exceeded", "identifier", identifier, "provider", provider)
			return blockRequest(fmt.Sprintf("429 Too Many Requests: Rate limit exceeded for %s", identifier))
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
			if mesh.GlobalReputation != nil && sessionID != "" {
				mesh.GlobalReputation.Slash(sessionID, "WAF Jailbreak Detected")
				if telemetry.MeshSlashesTotal != nil {
					telemetry.MeshSlashesTotal.WithLabelValues("waf_jailbreak").Inc()
				}
			}
			telemetry.WafInterceptsTotal.WithLabelValues("jailbreak").Inc()
			atomic.AddInt64(&telemetry.RawWafBlocks, 1)
			telemetry.RequestsTotal.WithLabelValues(provider, "403").Inc()
			return blockRequest(fmt.Sprintf("WAF Blocked Request: %v", err))
		}

		// 2.2. AI-WAF: Data Loss Prevention (PII Redaction)
		sanitizedBody := waf.RedactPII(inputBody)
		if len(sanitizedBody) != len(inputBody) || !bytes.Equal(sanitizedBody, inputBody) {
			logging.Logger.Info("WAF DLP Engine redacted sensitive PII from payload", "session", sessionID)
			piiFlagged = true
			inputBody = sanitizedBody
			req.Body = io.NopCloser(bytes.NewBuffer(inputBody))
			req.ContentLength = int64(len(inputBody))
		}

		// 2.3. Preflight Moderation Hook
		if config.Cfg.Mesh.ReputationEnabled && config.Cfg.Mesh.PreflightModeration {
			promptStr := extractPromptString(inputBody)
			if promptStr != "" {
				safe, err := mesh.CheckModeration(promptStr)
				if err != nil {
					// Fail closed for safety
					logging.Logger.Error("Preflight moderation failed", "error", err)
					return blockRequest(fmt.Sprintf("Moderation API Error: %v", err))
				}
				if !safe {
					logging.Logger.Error("Moderation Flagged Request", "session", sessionID)
					if mesh.GlobalReputation != nil && sessionID != "" {
						mesh.GlobalReputation.Slash(sessionID, "OpenAI Moderation Flagged")
						if telemetry.MeshSlashesTotal != nil {
							telemetry.MeshSlashesTotal.WithLabelValues("openai_moderation").Inc()
						}
					}
					telemetry.WafInterceptsTotal.WithLabelValues("moderation").Inc()
					atomic.AddInt64(&telemetry.RawWafBlocks, 1)
					telemetry.RequestsTotal.WithLabelValues(provider, "403").Inc()
					return blockRequest("Request blocked by Preflight Moderation Engine")
				}
			}
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
		if hit := cache.GlobalCache.Get(canonicalizePayload(inputBody)); hit != nil {
			logging.Logger.Info("Semantic cache HIT — returning cached response", "provider", provider, "cost_saved", "$0.00")
			telemetry.CacheHitsTotal.Inc()
			telemetry.RequestsTotal.WithLabelValues(provider, fmt.Sprintf("%d", hit.StatusCode)).Inc()
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
			} else if config.Cfg.P2P.Enabled {
				// No local keys available! Attempt P2P offloading.
				p2pResp, p2pErr := p2p.OffloadRequest(req.Context(), provider, reqModel, sessionID, estimatedTokens, inputBody)
				if p2pErr == nil && p2pResp != nil && p2pResp.Success {
					mockRes := &http.Response{
						Status:        "200 OK",
						StatusCode:    200,
						Proto:         "HTTP/1.1",
						ProtoMajor:    1,
						ProtoMinor:    1,
						Body:          io.NopCloser(bytes.NewReader(p2pResp.EncryptedResponse)),
						ContentLength: int64(len(p2pResp.EncryptedResponse)),
						Header:        make(http.Header),
					}
					mockRes.Header.Set("Content-Type", "application/json")
					res = mockRes
					roundTripErr = nil
					atomic.AddInt64(&telemetry.RawPromptsOffloaded, 1)
					logging.Logger.Info("Successfully offloaded prompt to P2P mesh", "provider", provider)
					break // Break out of retry loop, we got our response!
				} else {
					if p2pErr != nil {
						logging.Logger.Debug("P2P offload failed", "err", p2pErr)
					} else if p2pResp != nil {
						logging.Logger.Warn("P2P peer rejected prompt", "peer_error", p2pResp.ErrorMessage)
					}
				}
			}
		}

		// Only do actual RoundTrip if we didn't already get a P2P mock response
		if res == nil {
			res, roundTripErr = t.Transport.RoundTrip(req)
		}
		if roundTripErr != nil {
			break // Break on hard network errors
		}

		// Intercept 429 Rate Limits
		if res.StatusCode == http.StatusTooManyRequests {
			if attempt < maxRetries {
				logging.Logger.Warn("Intercepted 429 Rate Limit. Auto-Retrying with new key...", "provider", provider, "attempt", attempt+1)
				// Close the failing response body to free the FD. 
				// We intentionally don't drain it via io.Copy to avoid goroutine leaks if the upstream stalls.
				// This prevents connection reuse, but safety is more important during a 429/500.
				if res.Body != nil {
					res.Body.Close()
				}
				res = nil // Reset so next iteration actually calls RoundTrip
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
				wasFailover = true
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

				// Close failing response
				if res.Body != nil {
					res.Body.Close()
				}
				res = nil
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
		wrapStreamingResponse(res, t.Counter, reqModel, provider, sessionID, authenticatedKeyHash, start)

		// Record input tokens only (output tokens are counted by the stream interceptor)
		go func() {
			latency := time.Since(start)
			if len(inputBody) > 0 {
				inTokens := t.Counter.Count(string(inputBody), reqModel)
				cost := tokenizer.EstimateCost(provider, reqModel, inTokens, 0)
				
				if config.Cfg.Cloud.Enabled {
					// We don't report streaming TTFT here because the stream interceptor does it.
					// We'll just let the stream interceptor report the full cost.
				} else {
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
			}
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
			cache.GlobalCache.Put(canonicalizePayload(inputBody), outputBody, res.StatusCode, cost)
			res.Header.Set("X-Tkngate-Cache", "MISS")
		}

		// 5. TOKEN COUNTING & LEDGER UPDATE (synchronous to prevent race conditions)
		// Extract real token counts from the provider's response JSON when available.
		// This ensures our cost tracking exactly matches the provider's dashboard.
		inTokens := 0
		outTokens := 0
		var realModel string

		if len(outputBody) > 0 && res.StatusCode >= 200 && res.StatusCode < 300 {
			var respJSON struct {
				Model string `json:"model"`
				Usage struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
					TotalTokens      int `json:"total_tokens"`
				} `json:"usage"`
			}
			if jsonErr := json.Unmarshal(outputBody, &respJSON); jsonErr == nil && respJSON.Usage.TotalTokens > 0 {
				inTokens = respJSON.Usage.PromptTokens
				outTokens = respJSON.Usage.CompletionTokens
				realModel = respJSON.Model
			}
		}

		// Fallback to heuristic counting if the response didn't include usage data
		if inTokens == 0 && outTokens == 0 {
			if len(inputBody) > 0 {
				inTokens = t.Counter.Count(string(inputBody), reqModel)
			}
			if len(outputBody) > 0 {
				outTokens = t.Counter.Count(string(outputBody), reqModel)
			}
		}

		// Use the model name from the response if available (e.g., "deepseek-v4-flash")
		if realModel != "" {
			reqModel = realModel
		}

		cost := tokenizer.EstimateCost(provider, reqModel, inTokens, outTokens)
		latency := time.Since(start)

		// Record transaction SYNCHRONOUSLY so the next request's budget check
		// sees the updated spend immediately — prevents parallel overspend.
		if config.Cfg.Cloud.Enabled {
			cloud.ReportUsage(cloud.UsageReport{
				KeyID:            authenticatedKeyHash,
				Provider:         provider,
				Model:            reqModel,
				PromptTokens:     inTokens,
				CompletionTokens: outTokens,
				CostUSD:          cost,
				LatencyMs:        int(latency.Milliseconds()),
				TTFTMs:           int(latency.Milliseconds()),
				FlaggedPII:       piiFlagged,
				WasFailover:      wasFailover,
				StatusCode:       res.StatusCode,
			})
		} else {
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
		}

		telemetry.RequestsTotal.WithLabelValues(provider, fmt.Sprintf("%d", res.StatusCode)).Inc()
		telemetry.TokensConsumedTotal.Add(float64(inTokens + outTokens))
		telemetry.BudgetSpentTotal.Add(cost)
		if authenticatedKeyHash != "" {
			telemetry.VirtualKeySpend.WithLabelValues(authenticatedKeyHash).Add(cost)
		}

		if res.StatusCode >= 200 && res.StatusCode < 300 && mesh.GlobalReputation != nil && sessionID != "" {
			mesh.GlobalReputation.RecordSuccess(sessionID)
		}

		telemetry.EmitAuditRecord(telemetry.AuditRecord{
			Provider:     provider,
			Model:        reqModel,
			SessionID:    sessionID,
			InputTokens:  inTokens,
			OutputTokens: outTokens,
			CostUSD:      cost,
			LatencyMs:    latency.Milliseconds(),
			StatusCode:   res.StatusCode,
			Action:       "ALLOW",
		})

		logging.Logger.Info("Request handled",
			"provider", provider,
			"model", reqModel,
			"cost_usd", cost,
			"latency_ms", latency.Milliseconds(),
			"zone", status.Zone)
	}

	// 6. SHADOW MODE (v1.1.0) — Fire-and-forget mirror to a shadow provider
	if config.Cfg.Shadow.Enabled && len(inputBody) > 0 && strings.HasSuffix(req.URL.Path, "/chat/completions") {
		if rand.Float64() < config.Cfg.Shadow.TrafficFraction {
			go fireShadowRequest(inputBody, provider, config.Cfg.Shadow.TargetProvider, config.Cfg.Shadow.TargetModel)
		}
	}

	return res, nil
}

func fireShadowRequest(body []byte, primaryProvider string, shadowProvider string, shadowModel string) {
	defer func() {
		if r := recover(); r != nil {
			logging.Logger.Error("Shadow mode panic recovered", "error", r)
		}
	}()

	providerCfg, ok := config.Cfg.Providers[shadowProvider]
	if !ok {
		logging.Logger.Warn("Shadow mode: target provider not configured", "provider", shadowProvider)
		return
	}

	shadowBody := replaceModel(body, shadowModel)

	baseURL := strings.TrimSuffix(providerCfg.BaseURL, "/")

	// SSRF protection: validate the BaseURL before making a request
	u, err := url.Parse(baseURL)
	if err != nil {
		logging.Logger.Error("Shadow mode: invalid base URL", "url", baseURL, "error", err)
		return
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		logging.Logger.Error("Shadow mode: base URL must use http(s)", "scheme", u.Scheme)
		return
	}
	// Block requests to private/internal IPs
	hostname := u.Hostname()
	if hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1" ||
		strings.HasPrefix(hostname, "10.") || strings.HasPrefix(hostname, "192.168.") || strings.HasPrefix(hostname, "169.254.") {
		// Allow localhost only for ollama provider
		if shadowProvider != "ollama" {
			logging.Logger.Error("Shadow mode: refusing to send requests to internal network", "host", hostname)
			return
		}
	}

	shadowURL := u.String() + "/chat/completions"

	shadowReq, err := http.NewRequest("POST", shadowURL, bytes.NewBuffer(shadowBody))
	if err != nil {
		logging.Logger.Error("Shadow mode: failed to create request", "error", err)
		return
	}

	shadowReq.Header.Set("Content-Type", "application/json")
	if shadowProvider == "anthropic" {
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
		logging.Logger.Warn("Shadow mode: request failed", "provider", shadowProvider, "error", err)
		return
	}
	bodyBytes, _ := io.ReadAll(shadowRes.Body)

	var shadowText string
	var respData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &respData); err == nil {
		if choices, ok := respData["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if msg, ok := choice["message"].(map[string]interface{}); ok {
					shadowText, _ = msg["content"].(string)
				}
			}
		}
	}

	if telemetry.GlobalDiffBroadcaster != nil && shadowText != "" {
		telemetry.GlobalDiffBroadcaster.Broadcast(telemetry.DiffEvent{
			PrimaryProvider: primaryProvider,
			PrimaryModel:    extractModel(body),
			PrimaryText:     "(Check primary stream for live text)",
			ShadowProvider:  shadowProvider,
			ShadowModel:     shadowModel,
			ShadowText:      shadowText,
		})
	}

	logging.Logger.Info("Shadow mode: mirror complete",
		"provider", shadowProvider,
		"model", shadowModel,
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

// canonicalizePayload produces a deterministic, tool-call-aware representation
// of the request payload for semantic cache keying.
//
// v2.7.0 — Native Tool-Calling Support:
//   - Scrubs random `tool_call_id` / `id` fields from messages so that
//     identical agent conversation trees with different randomly generated
//     IDs produce the same cache hash.
//   - Sorts the `tools` array alphabetically by `function.name` so that
//     clients sending the same tool set in a different order still hit cache.
//   - Preserves `response_format` for Structured Outputs cache keying.
func canonicalizePayload(payload []byte) []byte {
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return payload
	}

	canonical := make(map[string]interface{})
	if model, ok := data["model"]; ok {
		canonical["model"] = model
	}

	// Canonicalize messages: strip random tool_call_id fields
	if messages, ok := data["messages"].([]interface{}); ok {
		canonical["messages"] = scrubToolCallIDs(messages)
	} else if messages, ok := data["messages"]; ok {
		canonical["messages"] = messages
	}

	// Sort tools array deterministically by function.name
	if tools, ok := data["tools"].([]interface{}); ok {
		canonical["tools"] = sortToolsByName(tools)
	} else if tools, ok := data["tools"]; ok {
		canonical["tools"] = tools
	}

	if toolChoice, ok := data["tool_choice"]; ok {
		canonical["tool_choice"] = toolChoice
	}
	if responseFormat, ok := data["response_format"]; ok {
		canonical["response_format"] = responseFormat
	}

	if newPayload, err := json.Marshal(canonical); err == nil {
		return newPayload
	}
	return payload
}

// scrubToolCallIDs removes randomly generated `tool_call_id` and `id` fields
// from tool-call messages so that identical conversational trees always
// produce the same cache hash regardless of the LLM's random ID generation.
func scrubToolCallIDs(messages []interface{}) []interface{} {
	cleaned := make([]interface{}, len(messages))
	for i, msg := range messages {
		msgMap, ok := msg.(map[string]interface{})
		if !ok {
			cleaned[i] = msg
			continue
		}

		// Deep-copy the message map so we don't mutate the original payload
		scrubbed := make(map[string]interface{})
		for k, v := range msgMap {
			scrubbed[k] = v
		}

		// Strip `tool_call_id` from role:"tool" messages
		delete(scrubbed, "tool_call_id")

		// Strip `id` from each item inside `tool_calls` arrays (role:"assistant")
		if toolCalls, ok := scrubbed["tool_calls"].([]interface{}); ok {
			cleanedCalls := make([]interface{}, len(toolCalls))
			for j, tc := range toolCalls {
				tcMap, ok := tc.(map[string]interface{})
				if !ok {
					cleanedCalls[j] = tc
					continue
				}
				tcCopy := make(map[string]interface{})
				for k, v := range tcMap {
					tcCopy[k] = v
				}
				delete(tcCopy, "id")
				cleanedCalls[j] = tcCopy
			}
			scrubbed["tool_calls"] = cleanedCalls
		}

		cleaned[i] = scrubbed
	}
	return cleaned
}

// sortToolsByName sorts a tools array alphabetically by each tool's
// function.name field.  This ensures that clients sending the same set of
// tools in a different order still produce the same cache key.
func sortToolsByName(tools []interface{}) []interface{} {
	sorted := make([]interface{}, len(tools))
	copy(sorted, tools)

	// Simple insertion sort — tool arrays are typically < 20 items
	for i := 1; i < len(sorted); i++ {
		key := sorted[i]
		keyName := extractToolFunctionName(key)
		j := i - 1
		for j >= 0 && extractToolFunctionName(sorted[j]) > keyName {
			sorted[j+1] = sorted[j]
			j--
		}
		sorted[j+1] = key
	}
	return sorted
}

// extractToolFunctionName extracts the function.name from an OpenAI tool object.
func extractToolFunctionName(tool interface{}) string {
	toolMap, ok := tool.(map[string]interface{})
	if !ok {
		return ""
	}
	fn, ok := toolMap["function"].(map[string]interface{})
	if !ok {
		return ""
	}
	name, _ := fn["name"].(string)
	return name
}

func extractPromptString(payload []byte) string {
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return ""
	}

	messages, ok := data["messages"].([]interface{})
	if !ok || len(messages) == 0 {
		return ""
	}

	lastMsg, ok := messages[len(messages)-1].(map[string]interface{})
	if !ok {
		return ""
	}

	content, ok := lastMsg["content"].(string)
	if !ok {
		return ""
	}

	return content
}
