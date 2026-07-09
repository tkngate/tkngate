package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"tkngate/internal/budget"
	"tkngate/internal/cloud"
	"tkngate/internal/config"
	"tkngate/internal/logging"
	"tkngate/internal/tokenizer"
)

// streamingTransport wraps the standard proxyTransport to handle SSE (Server-Sent Events)
// streaming responses. When the upstream LLM returns `Transfer-Encoding: chunked` with
// `text/event-stream`, this transport intercepts each SSE chunk, counts tokens in real-time,
// and enforces budget limits mid-stream (cutting off the response if the budget is exhausted).
type streamingTransport struct {
	inner   *proxyTransport
	counter *tokenizer.Counter
}

// streamingResponseWriter wraps the response body to intercept SSE chunks for real-time token counting.
type streamingResponseBody struct {
	original       io.ReadCloser
	scanner        *bufio.Scanner
	counter        *tokenizer.Counter
	model          string
	provider       string
	sessionID      string
	virtualKeyHash string
	tokensSoFar    int
	buffer         bytes.Buffer
	done           bool
	startTime      time.Time
	firstTokenTime time.Time
}

func newStreamingResponseBody(body io.ReadCloser, counter *tokenizer.Counter, model, provider, sessionID, virtualKeyHash string, startTime time.Time) *streamingResponseBody {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024) // 1MB max line
	return &streamingResponseBody{
		original:       body,
		scanner:        scanner,
		counter:        counter,
		model:          model,
		provider:       provider,
		sessionID:      sessionID,
		virtualKeyHash: virtualKeyHash,
		startTime:      startTime,
	}
}

func (s *streamingResponseBody) Read(p []byte) (int, error) {
	// If we have buffered data from a previous scan, drain it first
	if s.buffer.Len() > 0 {
		return s.buffer.Read(p)
	}

	if s.done {
		return 0, io.EOF
	}

	// Scan the next line from the SSE stream
	if !s.scanner.Scan() {
		s.done = true
		// Record final streaming transaction
		s.recordTransaction()
		if err := s.scanner.Err(); err != nil {
			return 0, err
		}
		return 0, io.EOF
	}

	// Capture time to first token (TTFT) on first successful read of the stream
	if s.firstTokenTime.IsZero() {
		s.firstTokenTime = time.Now()
	}

	line := s.scanner.Text()

	// SSE data lines start with "data: "
	if strings.HasPrefix(line, "data: ") {
		data := strings.TrimPrefix(line, "data: ")

		// "[DONE]" is the standard OpenAI stream termination signal
		if data == "[DONE]" {
			s.done = true
			s.recordTransaction()
			s.buffer.WriteString(line)
			s.buffer.WriteString("\n\n")
			return s.buffer.Read(p)
		}

		// Parse the SSE JSON chunk to extract the token delta
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err == nil {
			for _, choice := range chunk.Choices {
				if choice.Delta.Content != "" {
					tokens := s.counter.Count(choice.Delta.Content, s.model)
					s.tokensSoFar += tokens
				}
			}
		}

		// v1.2.0 & v1.3.0: Mid-stream budget enforcement
		// If the streaming response has generated enough tokens to breach the session or virtual key budget, cut it off
		if s.virtualKeyHash != "" {
			vkStatus, err := budget.CheckVirtualKeyBudget(s.virtualKeyHash)
			if err == nil && vkStatus.Zone == budget.ZoneRed {
				logging.Logger.Warn("Budget Guard: cutting SSE stream mid-response due to Virtual Key budget exhaustion",
					"key_hash", s.virtualKeyHash, "tokens_so_far", s.tokensSoFar)
				s.done = true
				s.buffer.WriteString("data: [DONE]\n\n")
				s.recordTransaction()
				return s.buffer.Read(p)
			}
		} else if s.sessionID != "" {
			sessionStatus, err := budget.CheckSessionBudget(s.sessionID)
			if err == nil && sessionStatus.Zone == budget.ZoneRed {
				logging.Logger.Warn("Budget Guard: cutting SSE stream mid-response due to session budget exhaustion",
					"session", s.sessionID, "tokens_so_far", s.tokensSoFar)
				s.done = true
				// Send a graceful termination
				s.buffer.WriteString("data: [DONE]\n\n")
				s.recordTransaction()
				return s.buffer.Read(p)
			}
		}
	}

	// Pass through the line (including empty lines for SSE framing)
	s.buffer.WriteString(line)
	s.buffer.WriteByte('\n')
	return s.buffer.Read(p)
}

func (s *streamingResponseBody) Close() error {
	return s.original.Close()
}

// recordTransaction logs the final token count for streaming responses
func (s *streamingResponseBody) recordTransaction() {
	if s.tokensSoFar == 0 {
		return
	}

	cost := tokenizer.EstimateCost(s.provider, s.model, 0, s.tokensSoFar)
	
	// Calculate TTFT and full stream Latency
	latencyMs := int(time.Since(s.startTime).Milliseconds())
	ttftMs := latencyMs
	if !s.firstTokenTime.IsZero() {
		ttftMs = int(s.firstTokenTime.Sub(s.startTime).Milliseconds())
	}

	if config.Cfg.Cloud.Enabled {
		// Cloud telemetry for streams only reports the output tokens and cost
		cloud.ReportUsage(cloud.UsageReport{
			KeyID:            s.virtualKeyHash, // In cloud mode, this was set to the Cloud KeyID in middleware
			Provider:         s.provider,
			Model:            s.model,
			PromptTokens:     0,
			CompletionTokens: s.tokensSoFar,
			CostUSD:          cost,
			LatencyMs:        latencyMs,
			TTFTMs:           ttftMs,
			FlaggedPII:       false,
			WasFailover:      false,
			StatusCode:       200,
		})
	} else {
		tx := budget.Transaction{
			SessionID:        s.sessionID,
			VirtualKeyHash:   s.virtualKeyHash,
			Provider:         s.provider,
			Model:            s.model,
			InputTokens:      0, // Input tokens are counted before the stream starts
			OutputTokens:     s.tokensSoFar,
			EstimatedCostUSD: cost,
		}

		if err := budget.GlobalLedger.RecordTransaction(tx); err != nil {
			logging.Logger.Error("failed to record streaming transaction", "error", err)
		}
	}

	logging.Logger.Info("SSE stream complete",
		"provider", s.provider,
		"model", s.model,
		"output_tokens", s.tokensSoFar,
		"cost_usd", cost,
		"latency_ms", latencyMs,
		"ttft_ms", ttftMs)
}

// isStreamingRequest checks if the request payload has `"stream": true`
func isStreamingRequest(body []byte) bool {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return false
	}
	if stream, ok := data["stream"].(bool); ok {
		return stream
	}
	return false
}

// isStreamingResponse checks if the response is an SSE stream
func isStreamingResponse(res *http.Response) bool {
	ct := res.Header.Get("Content-Type")
	return strings.Contains(ct, "text/event-stream")
}

// wrapStreamingResponse wraps the response body with our SSE interceptor
func wrapStreamingResponse(res *http.Response, counter *tokenizer.Counter, model, provider, sessionID, virtualKeyHash string, startTime time.Time) {
	if !isStreamingResponse(res) {
		return
	}

	// Add streaming header for the dashboard
	res.Header.Set("X-Tkngate-Stream", "true")

	// Only set Cache-Control if not already set
	if res.Header.Get("Cache-Control") == "" {
		res.Header.Set("Cache-Control", "no-cache")
	}

	streamBody := newStreamingResponseBody(res.Body, counter, model, provider, sessionID, virtualKeyHash, startTime)
	res.Body = streamBody

	// Disable content length since we're streaming
	res.ContentLength = -1
	res.Header.Del("Content-Length")

	// Ensure Transfer-Encoding is chunked
	if !strings.Contains(res.Header.Get("Transfer-Encoding"), "chunked") {
		res.Header.Set("Transfer-Encoding", "chunked")
	}

	_ = config.Cfg // Ensure config is accessible (prevents unused import)
}
