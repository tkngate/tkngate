package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"

	"tkngate/internal/tokenizer"
)

// NewProxy creates a new configured reverse proxy
func NewProxy() (*httputil.ReverseProxy, error) {
	counter, err := tokenizer.NewCounter()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize token counter: %w", err)
	}

	transport := &proxyTransport{
		Transport: http.DefaultTransport,
		Counter:   counter,
	}

	proxy := &httputil.ReverseProxy{
		Director:  Director,
		Transport: transport,
	}

	return proxy, nil
}
