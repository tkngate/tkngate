package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"time"

	"tkngate/internal/config"
	"tkngate/internal/tokenizer"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// NewProxy creates a new configured reverse proxy.
// A dedicated transport avoids the default Go HTTP client limit of 2 idle
// connections per upstream host, which otherwise causes socket churn under load.
func NewProxy() (*httputil.ReverseProxy, error) {
	counter, err := tokenizer.NewCounter()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize token counter: %w", err)
	}

	baseTransport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		// Keep a larger idle pool for upstream providers to reduce churn under
		// concurrent requests while still bounding resource usage.
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	if config.Cfg.OTEL.Enabled {
		baseTransport = otelhttp.NewTransport(baseTransport)
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
