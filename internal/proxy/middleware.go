package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"tkngate/internal/budget"
	"tkngate/internal/compressor"
	"tkngate/internal/config"
	"tkngate/internal/logging"
	"tkngate/internal/pool"
	"tkngate/internal/tokenizer"
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

	sessionID := req.Header.Get("X-Tkngate-Session-ID")
	sessionZone := budget.ZoneGreen
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

	// 2. CAPTURE INPUT (For token counting & rewriting)
	var inputBody []byte
	var reqModel string
	if req.Body != nil {
		inputBody, _ = captureBody(req)
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

	// 3. EXECUTE REQUEST
	if pool.GlobalDRR != nil {
		dynamicKey, err := pool.GlobalDRR.GetNextKey(provider, 0.0)
		if err == nil && dynamicKey != "" {
			req.Header.Set("Authorization", "Bearer "+dynamicKey)
			logging.Logger.Info("DRR Engine rotated key", "provider", provider)
		}
	}

	res, err := t.Transport.RoundTrip(req)
	if err != nil {
		return res, err
	}

	// 4. CAPTURE OUTPUT (For token counting)
	var outputBody []byte
	if res.Body != nil {
		outputBody, _ = io.ReadAll(res.Body)
		res.Body = io.NopCloser(bytes.NewBuffer(outputBody))
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

	return res, nil
}

func blockRequest(message string) (*http.Response, error) {
	body := fmt.Sprintf(`{"error": "%s"}`, message)
	return &http.Response{
		Status:        "429 Too Many Requests",
		StatusCode:    429,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Body:          io.NopCloser(bytes.NewBufferString(body)),
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
