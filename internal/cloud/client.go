package cloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"tkngate/internal/config"
	"tkngate/internal/logging"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// allowedCloudSchemes restricts outgoing cloud requests to safe protocols.
var allowedCloudSchemes = map[string]bool{
	"https": true,
}

// sanitizeCloudURL validates and sanitizes the cloud API URL to prevent SSRF.
// Only HTTPS URLs to non-internal hosts are allowed.
func sanitizeCloudURL(rawURL string, path string) (string, error) {
	if rawURL == "" {
		return "", fmt.Errorf("cloud API URL is not configured")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid cloud API URL: %w", err)
	}

	if !allowedCloudSchemes[strings.ToLower(u.Scheme)] {
		return "", fmt.Errorf("cloud API URL must use HTTPS, got '%s'", u.Scheme)
	}

	// Reconstruct safely to prevent path traversal
	u.Path = strings.TrimSuffix(u.Path, "/") + path
	u.RawQuery = ""
	u.Fragment = ""

	return u.String(), nil
}

// ValidationResponse represents the payload returned by /api/internal/validate
type ValidationResponse struct {
	Valid           bool    `json:"valid"`
	Error           string  `json:"error,omitempty"`
	KeyID           string  `json:"keyId,omitempty"`
	OrgID           string  `json:"orgId,omitempty"`
	OrgName         string  `json:"orgName,omitempty"`
	KeyName         string  `json:"keyName,omitempty"`
	BudgetAllocated float64 `json:"budgetAllocated,omitempty"`
	BudgetConsumed  float64 `json:"budgetConsumed,omitempty"`
	BudgetRemaining float64 `json:"budgetRemaining,omitempty"`
	Zone            string  `json:"zone,omitempty"` // GREEN, AMBER, RED
}

// ValidateKey makes an HTTP request to the Next.js control plane to check the Virtual Key.
func ValidateKey(rawKey string) (*ValidationResponse, error) {
	if !config.Cfg.Cloud.Enabled {
		return nil, fmt.Errorf("cloud mode is disabled")
	}

	baseURL, err := sanitizeCloudURL(config.Cfg.Cloud.APIURL, "/validate")
	if err != nil {
		return nil, fmt.Errorf("invalid cloud config: %w", err)
	}

	// Use url.Values to safely encode query parameters (prevents injection)
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("key", rawKey)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-tkngate-internal-secret", config.Cfg.Cloud.Secret)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call cloud API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("key not found in cloud")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var valRes ValidationResponse
	if err := json.Unmarshal(body, &valRes); err != nil {
		return nil, fmt.Errorf("failed to parse cloud response: %w", err)
	}

	// Even if it returns 403 Budget exhausted, the JSON contains valid=false which we handle upstream
	return &valRes, nil
}

// UsageReport represents a single transaction to report back to the SaaS.
type UsageReport struct {
	KeyID            string  `json:"keyId"`
	Provider         string  `json:"provider"`
	Model            string  `json:"model"`
	PromptTokens     int     `json:"promptTokens"`
	CompletionTokens int     `json:"completionTokens"`
	CostUSD          float64 `json:"costUsd"`
	LatencyMs        int     `json:"latencyMs"`
	TTFTMs           int     `json:"ttftMs"`
	FlaggedPII       bool    `json:"flaggedPii"`
	WasFailover      bool    `json:"wasFailover"`
	StatusCode       int     `json:"statusCode"`
}

// ReportUsage asynchronously posts a usage transaction to the cloud to drive DORA metrics.
func ReportUsage(report UsageReport) {
	if !config.Cfg.Cloud.Enabled {
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logging.Logger.Error("Recovered panic in ReportUsage", "error", r)
			}
		}()

		payload, err := json.Marshal(report)
		if err != nil {
			logging.Logger.Error("Failed to marshal usage report", "error", err)
			return
		}

		usageURL, err := sanitizeCloudURL(config.Cfg.Cloud.APIURL, "/usage")
		if err != nil {
			logging.Logger.Error("Invalid cloud URL for usage report", "error", err)
			return
		}

		req, err := http.NewRequest("POST", usageURL, bytes.NewBuffer(payload))
		if err != nil {
			logging.Logger.Error("Failed to create usage request", "error", err)
			return
		}

		req.Header.Set("x-tkngate-internal-secret", config.Cfg.Cloud.Secret)
		req.Header.Set("Content-Type", "application/json")

		resp, err := httpClient.Do(req)
		if err != nil {
			logging.Logger.Error("Failed to send usage report to cloud", "error", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			logging.Logger.Error("Cloud rejected usage report", "status", resp.StatusCode, "response", string(body))
		}
	}()
}
