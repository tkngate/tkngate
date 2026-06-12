package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig              `mapstructure:"server"`
	Providers map[string]ProviderConfig `mapstructure:"providers"`
	Budget    BudgetConfig              `mapstructure:"budget"`
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
	AmberThresholdPct int     `mapstructure:"amber_threshold_pct"`
	RedThresholdPct   int     `mapstructure:"red_threshold_pct"`
	ResetInterval     string  `mapstructure:"reset_interval"`
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

	return nil
}
