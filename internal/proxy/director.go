package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"tkngate/internal/config"
)

// Director modifies the request before it is sent to the backend.
func Director(req *http.Request) {
	// Example path: /openai/chat/completions -> Provider = openai
	pathParts := strings.SplitN(strings.TrimPrefix(req.URL.Path, "/"), "/", 2)
	if len(pathParts) < 1 {
		return
	}

	providerKey := pathParts[0]
	providerCfg, ok := config.Cfg.Providers[providerKey]
	if !ok {
		return
	}

	// Determine backend URL
	baseURL := strings.TrimSuffix(providerCfg.BaseURL, "/")

	// Reconstruct the intended path
	newPath := ""
	if len(pathParts) > 1 {
		newPath = "/" + pathParts[1]
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

	// Inject authentication
	if providerKey == "anthropic" {
		req.Header.Set("x-api-key", providerCfg.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01") // typically required
	} else {
		req.Header.Set("Authorization", "Bearer "+providerCfg.APIKey)
	}

	// If there's a body, we might need to peek at it later for token counting.
	// But `Director` isn't the best place for it because we can't easily capture the response here.
	// We handle that in our custom RoundTripper / Middleware.
}

// captureBody reads a body, returns the bytes, and replaces the body so it can be read again.
func captureBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	return bodyBytes, nil
}
