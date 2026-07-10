package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RawWafBlocks   int64
	RawZkpVerified int64
	RawZkpFailed   int64
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

	// MeshSlashesTotal tracks the number of times a node's reputation was slashed.
	MeshSlashesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tkngate_mesh_slashes_total",
			Help: "Total number of reputation slashes applied to nodes",
		},
		[]string{"reason"},
	)

	// MeshTrustScore tracks the current reputation score of a node.
	MeshTrustScore = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tkngate_mesh_trust_score",
			Help: "Current reputation trust score of a node",
		},
		[]string{"node_id"},
	)

	// ZkpVerifiedTotal tracks the number of valid ZK-SNARK attestations processed.
	ZkpVerifiedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tkngate_zkp_verified_total",
			Help: "Total number of valid ZK-SNARK proofs verified",
		},
	)

	// ZkpFailedTotal tracks the number of invalid or missing ZK-SNARK attestations rejected.
	ZkpFailedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tkngate_zkp_failed_total",
			Help: "Total number of invalid ZK-SNARK proofs rejected",
		},
	)
)
