package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"tkngate/internal/config"
)

func getDefaultBaseURL(provider string) string {
	switch provider {
	case "openai":
		return "https://api.openai.com"
	case "anthropic":
		return "https://api.anthropic.com"
	case "deepseek":
		return "https://api.deepseek.com"
	case "mistral":
		return "https://api.mistral.ai"
	case "kimi":
		return "https://api.moonshot.cn"
	case "groq":
		return "https://api.groq.com/openai"
	case "ollama":
		return "http://127.0.0.1:11434"
	case "gemini":
		return "https://generativelanguage.googleapis.com"
	default:
		return "https://api.openai.com"
	}
}

// Director modifies the request before it is sent to the backend.
func Director(req *http.Request) {
	// Example path: /openai/v1/chat/completions -> Provider = openai
	// Or standard OpenAI SDK: /v1/chat/completions -> Provider defaults to openai
	cleanPath := strings.TrimPrefix(req.URL.Path, "/")
	pathParts := strings.SplitN(cleanPath, "/", 2)
	
	if len(pathParts) < 1 || cleanPath == "" {
		return
	}

	providerKey := pathParts[0]
	providerCfg, ok := config.Cfg.Providers[providerKey]
	
	var newPath string
	
	if !ok {
		// If the first segment isn't a known provider (e.g. "v1"), default to OpenAI
		// This makes Tkngate a 100% drop-in replacement for the OpenAI SDK without URL hacks.
		providerKey = "openai"
		providerCfg = config.Cfg.Providers[providerKey]
		newPath = "/" + cleanPath
	} else {
		// We matched a provider prefix. Strip it from the forwarded path.
		if len(pathParts) > 1 {
			newPath = "/" + pathParts[1]
		} else {
			newPath = "/"
		}
	}

	// Determine backend URL
	baseURL := strings.TrimSuffix(providerCfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = getDefaultBaseURL(providerKey)
	}

	// Update request URL
	req.URL.Scheme = "https"
	if strings.HasPrefix(baseURL, "http://") {
		req.URL.Scheme = "http"
		baseURL = strings.TrimPrefix(baseURL, "http://")
	} else {
		baseURL = strings.TrimPrefix(baseURL, "https://")
	}

	hostPath := strings.SplitN(baseURL, "/", 2)
	req.URL.Host = hostPath[0]
	if len(hostPath) > 1 {
		req.URL.Path = "/" + hostPath[1] + newPath
	} else {
		req.URL.Path = newPath
	}

	req.Host = req.URL.Host

	// Save original auth header before overwriting so middleware can extract the virtual key
	originalAuth := req.Header.Get("Authorization")
	if originalAuth != "" {
		req.Header.Set("X-Tkngate-Original-Auth", originalAuth)
	}

	// Inject authentication
	if providerKey == "anthropic" {
		req.Header.Set("x-api-key", providerCfg.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01") // typically required
	} else if providerKey == "gemini" {
		req.Header.Set("x-goog-api-key", providerCfg.APIKey)
	} else {
		req.Header.Set("Authorization", "Bearer "+providerCfg.APIKey)
	}

	// Tell the middleware which provider we resolved
	req.Header.Set("X-Tkngate-Provider", providerKey)

	// If there's a body, we might need to peek at it later for token counting.
	// But `Director` isn't the best place for it because we can't easily capture the response here.
	// We handle that in our custom RoundTripper / Middleware.
}

// captureBody reads a body, returns the bytes, and replaces the body so it can be read again.
func captureBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	// Limit request body to 10MB to prevent DoS via unbounded memory allocation
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 10*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	return bodyBytes, nil
}
