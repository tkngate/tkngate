package validator

import (
	"fmt"
	"net/http"
	"time"
)

// ValidateKey performs a lightweight ping to the provider's API to ensure the key is real and active.
func ValidateKey(provider, key string) error {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	switch provider {
	case "openai":
		req, err := http.NewRequest("GET", "https://api.openai.com/v1/models", nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+key)

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("network error connecting to OpenAI: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == 401 {
			return fmt.Errorf("invalid or revoked OpenAI API key")
		}
		if resp.StatusCode != 200 {
			return fmt.Errorf("unexpected status code from OpenAI: %d", resp.StatusCode)
		}
		return nil

	case "anthropic":
		// Anthropic does not have a widely used /models GET endpoint, 
		// but checking /v1/messages with GET will yield a 405 Method Not Allowed 
		// if the key is valid, and 401 Unauthorized if the key is fake.
		req, err := http.NewRequest("GET", "https://api.anthropic.com/v1/messages", nil)
		if err != nil {
			return err
		}
		req.Header.Set("x-api-key", key)
		req.Header.Set("anthropic-version", "2023-06-01")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("network error connecting to Anthropic: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == 401 {
			return fmt.Errorf("invalid or revoked Anthropic API key")
		}
		// If we get 405 Method Not Allowed, the auth succeeded but the method is wrong, which is fine!
		if resp.StatusCode != 405 && resp.StatusCode != 200 {
			return fmt.Errorf("unexpected status code from Anthropic: %d", resp.StatusCode)
		}
		return nil

	default:
		return fmt.Errorf("unsupported provider '%s' for validation. Supported providers: openai, anthropic", provider)
	}
}
