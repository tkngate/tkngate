package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"

	"tkngate/internal/config"
	"tkngate/internal/tokenizer"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// NewProxy creates a new configured reverse proxy
func NewProxy() (*httputil.ReverseProxy, error) {
	counter, err := tokenizer.NewCounter()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize token counter: %w", err)
	}

	var baseTransport http.RoundTripper = http.DefaultTransport
	if config.Cfg.OTEL.Enabled {
		baseTransport = otelhttp.NewTransport(http.DefaultTransport)
	}

	transport := &proxyTransport{
		Transport: baseTransport,
		Counter:   counter,
	}

	proxy := &httputil.ReverseProxy{
		Director:  Director,
		Transport: transport,
	}

	return proxy, nil
}
