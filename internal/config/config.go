package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig              `mapstructure:"server" json:"server"`
	Providers  map[string]ProviderConfig `mapstructure:"providers" json:"providers"`
	Budget     BudgetConfig              `mapstructure:"budget" json:"budget"`
	Compressor CompressorConfig          `mapstructure:"compressor" json:"compressor"`
	Cache      CacheConfig               `mapstructure:"cache" json:"cache"`
	Telemetry  TelemetryConfig           `mapstructure:"telemetry" json:"telemetry"`
	Shadow     ShadowConfig              `mapstructure:"shadow" json:"shadow"`
	RateLimit  RateLimitConfig           `mapstructure:"rate_limit" json:"rate_limit"`
	Mesh       MeshConfig                `mapstructure:"mesh" json:"mesh"`
	Cloud      CloudConfig               `mapstructure:"cloud" json:"cloud"`
	WAF        WAFConfig                 `mapstructure:"waf" json:"waf"`
}

type WAFConfig struct {
	Enabled   bool     `mapstructure:"enabled" json:"enabled"`
	Blocklist []string `mapstructure:"blocklist" json:"blocklist"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port" json:"port"`
	Host string `mapstructure:"host" json:"host"`
}

type ProviderConfig struct {
	APIKey       string `mapstructure:"api_key" json:"api_key"`
	BaseURL      string `mapstructure:"base_url" json:"base_url"`
	DefaultModel string `mapstructure:"default_model" json:"default_model"`
}

type BudgetConfig struct {
	GlobalLimitUSD    float64 `mapstructure:"global_limit_usd" json:"global_limit_usd"`
	MaxSessionCostUSD float64 `mapstructure:"max_session_cost_usd" json:"max_session_cost_usd"`
	FallbackModel     string  `mapstructure:"fallback_model" json:"fallback_model"`
	FallbackProvider  string  `mapstructure:"fallback_provider" json:"fallback_provider"`
	AmberThresholdPct int     `mapstructure:"amber_threshold_pct" json:"amber_threshold_pct"`
	RedThresholdPct   int     `mapstructure:"red_threshold_pct" json:"red_threshold_pct"`
	ResetInterval     string  `mapstructure:"reset_interval" json:"reset_interval"`
}

type CompressorConfig struct {
	Enabled        bool   `mapstructure:"enabled" json:"enabled"`
	SoftTokenLimit int    `mapstructure:"soft_token_limit" json:"soft_token_limit"`
	Strategy       string `mapstructure:"strategy" json:"strategy"`
}

type CacheConfig struct {
	Enabled    bool   `mapstructure:"enabled" json:"enabled"`
	MaxEntries int    `mapstructure:"max_entries" json:"max_entries"`
	TTLSeconds int    `mapstructure:"ttl_seconds" json:"ttl_seconds"`
	RedisURI   string `mapstructure:"redis_uri" json:"redis_uri"`
}

type TelemetryConfig struct {
	Enabled bool   `mapstructure:"enabled" json:"enabled"`
	Port    int    `mapstructure:"port" json:"port"`
	Host    string `mapstructure:"host" json:"host"`
}

type ShadowConfig struct {
	Enabled         bool    `mapstructure:"enabled" json:"enabled"`
	TargetProvider  string  `mapstructure:"target_provider" json:"target_provider"`
	TargetModel     string  `mapstructure:"target_model" json:"target_model"`
	TrafficFraction float64 `mapstructure:"traffic_fraction" json:"traffic_fraction"`
}

type RateLimitConfig struct {
	Enabled           bool `mapstructure:"enabled" json:"enabled"`
	RequestsPerMinute int  `mapstructure:"requests_per_minute" json:"requests_per_minute"`
	BurstSize         int  `mapstructure:"burst_size" json:"burst_size"`
}

type MeshConfig struct {
	ReputationEnabled   bool    `mapstructure:"reputation_enabled" json:"reputation_enabled"`
	PreflightModeration bool    `mapstructure:"preflight_moderation" json:"preflight_moderation"`
	ModerationAPIKey    string  `mapstructure:"moderation_api_key" json:"moderation_api_key"`
	InitialTrustScore   float64 `mapstructure:"initial_trust_score" json:"initial_trust_score"`
	SlashPenalty        float64 `mapstructure:"slash_penalty" json:"slash_penalty"`
	BlacklistThreshold  float64 `mapstructure:"blacklist_threshold" json:"blacklist_threshold"`
	PremiumTrustMinimum float64 `mapstructure:"premium_trust_minimum" json:"premium_trust_minimum"`
	FreeRiderLimit      int     `mapstructure:"free_rider_limit" json:"free_rider_limit"`
	StrictZKPMode       bool    `mapstructure:"strict_zkp_mode" json:"strict_zkp_mode"`
}

type CloudConfig struct {
	Enabled bool   `mapstructure:"enabled" json:"enabled"`
	APIURL  string `mapstructure:"api_url" json:"api_url"`
	Secret  string `mapstructure:"secret" json:"secret"`
}

var Cfg Config

// LoadConfig reads configuration from file or environment variables.
func LoadConfig() error {
	viper.SetConfigName("tkngate")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/tkngate/")
	viper.AddConfigPath("$HOME/.tkngate")

	viper.SetEnvPrefix("tkngate")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %w", err)
		}
		// It's ok if config file is not found, we can rely on env vars or defaults
	}

	if err := viper.Unmarshal(&Cfg); err != nil {
		return fmt.Errorf("unable to decode into struct: %w", err)
	}

	return validateConfig()
}

func SaveConfig(updated Config) error {
	// Restore redacted values from the active configuration
	if Cfg.Providers != nil && updated.Providers != nil {
		for k, p := range updated.Providers {
			if p.APIKey == "[REDACTED]" {
				if existing, ok := Cfg.Providers[k]; ok {
					p.APIKey = existing.APIKey
					updated.Providers[k] = p
				}
			}
		}
	}
	if updated.Cloud.Secret == "[REDACTED]" {
		updated.Cloud.Secret = Cfg.Cloud.Secret
	}
	if updated.Mesh.ModerationAPIKey == "[REDACTED]" {
		updated.Mesh.ModerationAPIKey = Cfg.Mesh.ModerationAPIKey
	}

	Cfg = updated

	// Write to tkngate.yaml as pretty JSON (which is valid YAML)
	data, err := json.MarshalIndent(Cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	err = os.WriteFile("tkngate.yaml", data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write tkngate.yaml: %w", err)
	}

	return nil
}

func validateConfig() error {
	if Cfg.Server.Port == 0 {
		Cfg.Server.Port = 7477
	}
	if Cfg.Server.Host == "" {
		Cfg.Server.Host = "127.0.0.1"
	}

	// Seed all known providers so they always appear in the config/dashboard,
	// even if the user hasn't configured them yet.
	knownProviders := []string{"openai", "anthropic", "deepseek", "mistral", "kimi", "groq", "ollama"}
	if Cfg.Providers == nil {
		Cfg.Providers = make(map[string]ProviderConfig)
	}
	for _, p := range knownProviders {
		if _, exists := Cfg.Providers[p]; !exists {
			Cfg.Providers[p] = ProviderConfig{}
		}
	}

	if Cfg.Budget.AmberThresholdPct == 0 {
		Cfg.Budget.AmberThresholdPct = 70
	}
	if Cfg.Budget.RedThresholdPct == 0 {
		Cfg.Budget.RedThresholdPct = 95
	}

	if Cfg.Compressor.SoftTokenLimit == 0 {
		Cfg.Compressor.SoftTokenLimit = 4000 // Default 4k token soft limit
	}
	if Cfg.Compressor.Strategy == "" {
		Cfg.Compressor.Strategy = "go-ast"
	}

	if Cfg.Cache.MaxEntries == 0 {
		Cfg.Cache.MaxEntries = 1000
	}
	if Cfg.Cache.TTLSeconds == 0 {
		Cfg.Cache.TTLSeconds = 3600
	}

	if Cfg.Telemetry.Port == 0 {
		Cfg.Telemetry.Port = 7478
	}
	if Cfg.Telemetry.Host == "" {
		Cfg.Telemetry.Host = "127.0.0.1"
	}

	if Cfg.RateLimit.RequestsPerMinute == 0 {
		Cfg.RateLimit.RequestsPerMinute = 60
	}
	if Cfg.RateLimit.BurstSize == 0 {
		Cfg.RateLimit.BurstSize = 10
	}

	if Cfg.Mesh.InitialTrustScore == 0 {
		Cfg.Mesh.InitialTrustScore = 50.0
	}
	if Cfg.Mesh.SlashPenalty == 0 {
		Cfg.Mesh.SlashPenalty = 20.0
	}
	if Cfg.Mesh.BlacklistThreshold == 0 {
		Cfg.Mesh.BlacklistThreshold = -100.0
	}
	if Cfg.Mesh.PremiumTrustMinimum == 0 {
		Cfg.Mesh.PremiumTrustMinimum = 80.0 // Need 80+ for premium keys
	}
	if Cfg.Mesh.FreeRiderLimit == 0 {
		Cfg.Mesh.FreeRiderLimit = 10000 // Default 10k token limit for free riders
	}

	return nil
}

