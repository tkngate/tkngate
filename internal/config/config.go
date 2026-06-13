package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig              `mapstructure:"server"`
	Providers  map[string]ProviderConfig `mapstructure:"providers"`
	Budget     BudgetConfig              `mapstructure:"budget"`
	Compressor CompressorConfig          `mapstructure:"compressor"`
	Cache      CacheConfig               `mapstructure:"cache"`
	Telemetry  TelemetryConfig           `mapstructure:"telemetry"`
	Shadow     ShadowConfig              `mapstructure:"shadow"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type ProviderConfig struct {
	APIKey       string `mapstructure:"api_key"`
	BaseURL      string `mapstructure:"base_url"`
	DefaultModel string `mapstructure:"default_model"`
}

type BudgetConfig struct {
	GlobalLimitUSD    float64 `mapstructure:"global_limit_usd"`
	MaxSessionCostUSD float64 `mapstructure:"max_session_cost_usd"`
	FallbackModel     string  `mapstructure:"fallback_model"`
	FallbackProvider  string  `mapstructure:"fallback_provider"`
	AmberThresholdPct int     `mapstructure:"amber_threshold_pct"`
	RedThresholdPct   int     `mapstructure:"red_threshold_pct"`
	ResetInterval     string  `mapstructure:"reset_interval"`
}

type CompressorConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	SoftTokenLimit int    `mapstructure:"soft_token_limit"`
	Strategy       string `mapstructure:"strategy"`
}

type CacheConfig struct {
	Enabled    bool `mapstructure:"enabled"`
	MaxEntries int  `mapstructure:"max_entries"`
	TTLSeconds int  `mapstructure:"ttl_seconds"`
}

type TelemetryConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Port    int    `mapstructure:"port"`
	Host    string `mapstructure:"host"`
}

type ShadowConfig struct {
	Enabled         bool    `mapstructure:"enabled"`
	TargetProvider  string  `mapstructure:"target_provider"`
	TargetModel     string  `mapstructure:"target_model"`
	TrafficFraction float64 `mapstructure:"traffic_fraction"`
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

func validateConfig() error {
	if Cfg.Server.Port == 0 {
		Cfg.Server.Port = 7477
	}
	if Cfg.Server.Host == "" {
		Cfg.Server.Host = "127.0.0.1"
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
		Cfg.Cache.MaxEntries = 512
	}
	if Cfg.Cache.TTLSeconds == 0 {
		Cfg.Cache.TTLSeconds = 300
	}

	if Cfg.Telemetry.Port == 0 {
		Cfg.Telemetry.Port = 7478
	}
	if Cfg.Telemetry.Host == "" {
		Cfg.Telemetry.Host = "127.0.0.1"
	}

	return nil
}
