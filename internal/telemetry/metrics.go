package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RequestsTotal tracks all incoming LLM proxy requests.
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tkngate_requests_total",
			Help: "The total number of processed requests",
		},
		[]string{"provider", "status"},
	)

	// TokensConsumedTotal tracks the overall sum of prompt + completion tokens.
	TokensConsumedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tkngate_tokens_consumed_total",
			Help: "The total number of tokens consumed (prompt + completion)",
		},
	)

	// CacheHitsTotal tracks successful semantic cache hits.
	CacheHitsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tkngate_cache_hits_total",
			Help: "The total number of semantic cache hits",
		},
	)

	// WafInterceptsTotal tracks requests blocked by the AI WAF.
	WafInterceptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tkngate_waf_intercepts_total",
			Help: "The total number of requests blocked by the AI WAF",
		},
		[]string{"reason"},
	)

	// BudgetSpentTotal tracks the estimated total USD cost of all requests.
	BudgetSpentTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tkngate_budget_spent_usd_total",
			Help: "The total estimated cost of all tokens consumed in USD",
		},
	)

	// VirtualKeySpend tracks the estimated USD cost per Virtual Key.
	VirtualKeySpend = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tkngate_virtual_key_spend_usd_total",
			Help: "The total estimated cost in USD per Virtual Key",
		},
		[]string{"virtual_key"},
	)

	// ActiveConnections tracks the number of currently in-flight requests.
	ActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "tkngate_active_connections",
			Help: "The number of currently active in-flight requests to the proxy",
		},
	)
)
