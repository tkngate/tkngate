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
	"tkngate/internal/logging"
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

	// 1. BUDGET GUARD
	status, err := budget.CheckBudget()
	if err != nil {
		logging.Logger.Error("budget check failed", "error", err)
	} else if status.Zone == budget.ZoneRed {
		// Circuit Breaker triggered
		logging.Logger.Error("Request blocked by budget circuit breaker", "provider", provider)
		return blockRequest("Token Budget Exhausted")
	}

	// 2. CAPTURE INPUT (For token counting)
	var inputBody []byte
	var reqModel string
	if req.Body != nil {
		inputBody, _ = captureBody(req)
		reqModel = extractModel(inputBody)
	}

	// 3. EXECUTE REQUEST
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
