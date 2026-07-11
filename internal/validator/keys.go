package validator

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// allowedOllamaHosts restricts Ollama URLs to only local addresses to prevent SSRF.
var allowedOllamaHosts = map[string]bool{
	"localhost": true,
	"127.0.0.1": true,
	"::1":       true,
}

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
		// We perform a simple health check. Users might pass the host URL as the 'key'.
		endpoint := "http://localhost:11434/api/tags"
		if strings.HasPrefix(key, "http") {
			u, err := url.Parse(key)
			if err != nil {
				return fmt.Errorf("invalid ollama URL format: %v", err)
			}
			if u.Scheme != "http" && u.Scheme != "https" {
				return fmt.Errorf("ollama URL must use http or https")
			}

			// Extract hostname without port for validation
			hostname := u.Hostname()

			// Resolve the hostname to check for internal IPs (SSRF protection)
			if !allowedOllamaHosts[hostname] {
				// Also resolve DNS to block rebinding attacks targeting internal networks
				ips, err := net.LookupIP(hostname)
				if err != nil {
					return fmt.Errorf("cannot resolve ollama hostname '%s': %v", hostname, err)
				}
				allLocal := true
				for _, ip := range ips {
					if !ip.IsLoopback() {
						allLocal = false
						break
					}
				}
				if !allLocal {
					return fmt.Errorf("ollama URL must point to localhost (127.0.0.1 or ::1), got '%s'", hostname)
				}
			}

			// Reconstruct URL safely to prevent SSRF path injection
			u.Path = "/api/tags"
			u.RawQuery = ""
			u.Fragment = ""
			endpoint = u.String()
		}
		
		req, err := http.NewRequest("GET", endpoint, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("network error connecting to Ollama: %v", err)
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
