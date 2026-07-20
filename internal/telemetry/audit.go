// Package telemetry provides centralized audit logging for all proxy decisions.
// AuditRecords are batched and shipped asynchronously to a configurable HTTP sink.
package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"tkngate/internal/config"
	"tkngate/internal/logging"
)

// AuditRecord represents a single proxy decision event.
type AuditRecord struct {
	Timestamp   time.Time `json:"timestamp"`
	NodeID      string    `json:"node_id"`
	RequestID   string    `json:"request_id"`
	Provider    string    `json:"provider"`
	Model       string    `json:"model"`
	SessionID   string    `json:"session_id,omitempty"`
	InputTokens int       `json:"input_tokens"`
	OutputTokens int      `json:"output_tokens"`
	CostUSD     float64   `json:"cost_usd"`
	LatencyMs   int64     `json:"latency_ms"`
	StatusCode  int       `json:"status_code"`
	Action      string    `json:"action"`  // "ALLOW", "BLOCK_BUDGET", "BLOCK_WAF", "BLOCK_RATE_LIMIT", "CACHE_HIT"
	Reason      string    `json:"reason,omitempty"`
}

var (
	auditMu       sync.Mutex
	auditBuffer   []AuditRecord
	uiAuditBuffer [200]AuditRecord
	uiAuditIndex  int
	uiAuditCount  int
	auditOnce     sync.Once
)

// InitAuditShipper starts the background goroutine that flushes the audit buffer.
func InitAuditShipper() {
	if !config.Cfg.Audit.Enabled {
		return
	}
	auditOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				flushAuditBuffer()
			}
		}()
		logging.Logger.Info("Centralized Audit Log shipper started", "sink", config.Cfg.Audit.SinkURL)
	})
}

// EmitAuditRecord adds an AuditRecord to the async batch buffer.
func EmitAuditRecord(rec AuditRecord) {
	if !config.Cfg.Audit.Enabled {
		return
	}
	rec.Timestamp = time.Now()
	rec.NodeID = config.Cfg.Cluster.NodeID
	if rec.NodeID == "" {
		rec.NodeID = "standalone"
	}

	auditMu.Lock()
	auditBuffer = append(auditBuffer, rec)
	
	uiAuditBuffer[uiAuditIndex] = rec
	uiAuditIndex = (uiAuditIndex + 1) % 200
	if uiAuditCount < 200 {
		uiAuditCount++
	}

	// If buffer exceeds 500 records, flush immediately to prevent OOM.
	shouldFlush := len(auditBuffer) >= 500
	auditMu.Unlock()

	if shouldFlush {
		go flushAuditBuffer()
	}
}

// GetRecentAuditRecords returns the last N records for the dashboard UI.
func GetRecentAuditRecords() []AuditRecord {
	auditMu.Lock()
	defer auditMu.Unlock()
	
	result := make([]AuditRecord, 0, uiAuditCount)
	if uiAuditCount == 0 {
		return result
	}
	
	if uiAuditCount < 200 {
		for i := 0; i < uiAuditCount; i++ {
			result = append(result, uiAuditBuffer[i])
		}
	} else {
		for i := 0; i < 200; i++ {
			idx := (uiAuditIndex + i) % 200
			result = append(result, uiAuditBuffer[idx])
		}
	}
	return result
}

func flushAuditBuffer() {
	auditMu.Lock()
	if len(auditBuffer) == 0 {
		auditMu.Unlock()
		return
	}
	// Swap buffer — ship the current batch, new records go to fresh buffer.
	batch := auditBuffer
	auditBuffer = nil
	auditMu.Unlock()

	sinkURL := config.Cfg.Audit.SinkURL
	if sinkURL == "" {
		return
	}

	payload, err := json.Marshal(batch)
	if err != nil {
		logging.Logger.Error("Audit shipper: failed to serialize batch", "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sinkURL, bytes.NewBuffer(payload))
	if err != nil {
		logging.Logger.Error("Audit shipper: failed to create request", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logging.Logger.Error("Audit shipper: failed to ship batch", "error", err, "records", len(batch))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		logging.Logger.Error("Audit shipper: sink returned error", "status", resp.StatusCode, "records", len(batch))
		return
	}

	logging.Logger.Debug("Audit shipper: batch shipped successfully", "records", len(batch))
}
