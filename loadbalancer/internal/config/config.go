package config

import (
	"encoding/json"
	"os"
	"path"
	"strings"

	"github.com/Kostaaa1/loadbalancer/internal/models"
	"gopkg.in/yaml.v3"
)

const (
	defaultHealthCheckInterval = 5
)

type Config struct {
	Port                       string           `json:"port" yaml:"port"`
	Strategy                   string           `json:"strategy" yaml:"strategy"`
	Servers                    []*models.Server `json:"servers" yaml:"servers"`
	HealthCheckIntervalSeconds int              `json:"health_check_interval_seconds" yaml:"health_check_interval_seconds"`
	RateLimiterEnabled         bool             `json:"rate_limiter_enabled" yaml:"rate_limiter_enabled"`
	RateLimitTokens            int              `json:"rate_limit_tokens" yaml:"rate_limit_tokens"`
	RateLimitIntervalSeconds   int              `json:"rate_limit_interval_seconds" yaml:"rate_limit_interval_seconds"`
	TLSEnabled                 bool             `json:"tls_enabled" yaml:"tls_enabled"`
	TLSCertPath                string           `json:"tls_cert_path" yaml:"tls_cert_path"`
	TLSKeyPath                 string           `json:"tls_key_path" yaml:"tls_key_path"`
}

func Load(p string) (*Config, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}

	ext := path.Ext(p)

	var cfg Config

	if ext == ".json" {
		if err := json.Unmarshal(b, &cfg); err != nil {
			return nil, err
		}
	}

	if ext == ".yml" || ext == ".yaml" {
		if err := yaml.Unmarshal(b, &cfg); err != nil {
			return nil, err
		}
	}

	if !strings.HasPrefix(cfg.Port, ":") {
		cfg.Port = ":" + cfg.Port
	}

	if cfg.HealthCheckIntervalSeconds == 0 {
		cfg.HealthCheckIntervalSeconds = defaultHealthCheckInterval
	}

	return &cfg, err
}
