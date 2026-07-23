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
		// Use the official models endpoint to validate the key
		req, err := http.NewRequest("GET", "https://api.anthropic.com/v1/models", nil)
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
		if resp.StatusCode != 200 {
			return fmt.Errorf("unexpected status code from Anthropic: %d", resp.StatusCode)
		}
		return nil

	case "deepseek":
		return validateOpenAICompatible(client, key, "https://api.deepseek.com/v1/models", "DeepSeek")
	case "mistral":
		return validateOpenAICompatible(client, key, "https://api.mistral.ai/v1/models", "Mistral")
	case "ollama":
		// Ollama runs locally without authentication. 
		// To prevent SSRF, we ignore any user-supplied URL and hardcode the validation endpoint
		// to localhost. If a user needs a remote Ollama instance, they must configure it via tkngate.yaml BaseURL.
		endpoint := "http://127.0.0.1:11434/api/tags"
		
		req, err := http.NewRequest("GET", endpoint, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("Ollama is not running locally (could not connect to port 11434)")
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return fmt.Errorf("unexpected status code from Ollama at %s: %d", endpoint, resp.StatusCode)
		}
		return nil
	case "kimi":
		return validateOpenAICompatible(client, key, "https://api.moonshot.cn/v1/models", "Kimi")
	case "groq":
		return validateOpenAICompatible(client, key, "https://api.groq.com/openai/v1/models", "Groq")
	case "gemini":
		req, err := http.NewRequest("GET", "https://generativelanguage.googleapis.com/v1beta/models", nil)
		if err != nil {
			return err
		}
		req.Header.Set("x-goog-api-key", key)

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("network error connecting to Gemini: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == 401 || resp.StatusCode == 400 || resp.StatusCode == 403 {
			return fmt.Errorf("invalid or revoked Gemini API key")
		}
		if resp.StatusCode != 200 {
			return fmt.Errorf("unexpected status code from Gemini: %d", resp.StatusCode)
		}
		return nil
	default:
		return fmt.Errorf("unsupported provider '%s' for validation. Supported providers: openai, anthropic, deepseek, mistral, kimi, groq, ollama, gemini", provider)
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
