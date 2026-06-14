package mesh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"tkngate/internal/config"
	"tkngate/internal/logging"
)

type ModerationRequest struct {
	Input string `json:"input"`
}

type ModerationResponse struct {
	Results []struct {
		Flagged bool `json:"flagged"`
		Categories map[string]bool `json:"categories"`
	} `json:"results"`
}

// CheckModeration calls OpenAI's free moderation endpoint.
// Returns true if the prompt is safe, false if it is flagged or if an error occurs (fail-closed).
func CheckModeration(prompt string) (bool, error) {
	if !config.Cfg.Mesh.ReputationEnabled || !config.Cfg.Mesh.PreflightModeration {
		return true, nil // Moderation disabled, default to safe
	}

	apiKey := config.Cfg.Mesh.ModerationAPIKey
	if apiKey == "" {
		logging.Logger.Error("Preflight moderation enabled but no moderation_api_key provided. Failing closed for safety.")
		return false, fmt.Errorf("moderation_api_key is required when preflight_moderation is enabled")
	}

	reqBody := ModerationRequest{Input: prompt}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return false, fmt.Errorf("failed to marshal moderation request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/moderations", bytes.NewBuffer(jsonBody))
	if err != nil {
		return false, fmt.Errorf("failed to create moderation request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("moderation API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("moderation API returned status %d: %s", resp.StatusCode, string(body))
	}

	var modResp ModerationResponse
	if err := json.NewDecoder(resp.Body).Decode(&modResp); err != nil {
		return false, fmt.Errorf("failed to decode moderation response: %w", err)
	}

	if len(modResp.Results) == 0 {
		return false, fmt.Errorf("moderation API returned empty results")
	}

	if modResp.Results[0].Flagged {
		return false, nil // Unsafe prompt
	}

	return true, nil // Safe prompt
}
