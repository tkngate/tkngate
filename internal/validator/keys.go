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
		return validateOpenAICompatible(client, key, "https://api.openai.com/v1/models", "OpenAI")

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

	case "deepseek":
		return validateOpenAICompatible(client, key, "https://api.deepseek.com/v1/models", "DeepSeek")
	case "kimi":
		return validateOpenAICompatible(client, key, "https://api.moonshot.cn/v1/models", "Kimi")
	case "groq":
		return validateOpenAICompatible(client, key, "https://api.groq.com/openai/v1/models", "Groq")
	default:
		return fmt.Errorf("unsupported provider '%s' for validation. Supported providers: openai, anthropic, deepseek, kimi, groq", provider)
	}
}

func validateOpenAICompatible(client *http.Client, key, endpoint, name string) error {
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+key)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("network error connecting to %s: %v", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("invalid or revoked %s API key", name)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code from %s: %d", name, resp.StatusCode)
	}
	return nil
}
