package config

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Kostaaa1/loadbalancer/internal/models"
	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

const (
	defaultHealthCheckInterval = 5
)

type Config struct {
	Port                       string `json:"port" yaml:"port"`
	Strategy                   string `json:"strategy" yaml:"strategy"`
	HealthCheckIntervalSeconds int    `json:"health_check_interval_seconds" yaml:"health_check_interval_seconds"`

	RateLimit struct {
		RateLimiterEnabled       bool `json:"rate_limiter_enabled" yaml:"rate_limiter_enabled"`
		RateLimitTokens          int  `json:"rate_limit_tokens" yaml:"rate_limit_tokens"`
		RateLimitIntervalSeconds int  `json:"rate_limit_interval_seconds" yaml:"rate_limit_interval_seconds"`
	} `json:"rate_limit" yaml:"rate_limit"`

	Servers []*models.Server `json:"servers" yaml:"servers"`

	TLS struct {
		Enabled  bool   `json:"tls_enabled" yaml:"tls_enabled"`
		CertPath string `json:"tls_cert_file" yaml:"tls_cert_file"`
		KeyPath  string `json:"tls_key_file" yaml:"tls_key_file"`
	} `json:"tls" yaml:"tls"`

	Routing models.Routing `json:"routing" yaml:"routing"`

	configPath string `json:"-" yaml:"-"`
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

	cfg.configPath = p
	return &cfg, err
}

func (cfg *Config) Watch(ctx context.Context) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	var timer *time.Timer
	duration := 150 * time.Millisecond

	cooldownDur := 2 * time.Second
	var mu sync.Mutex
	var blocked bool

	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				mu.Lock()
				if blocked {
					mu.Unlock()
					continue
				}

				if timer != nil {
					timer.Stop()
				}

				timer = time.AfterFunc(duration, func() {
					mu.Lock()
					blocked = true
					mu.Unlock()

					if event.Has(fsnotify.Write) {
						p, err := os.FindProcess(os.Getpid())
						if err == nil {
							p.Signal(syscall.SIGHUP)
						}
					}

					time.AfterFunc(cooldownDur, func() {
						mu.Lock()
						blocked = false
						mu.Unlock()
					})
				})
				mu.Unlock()

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Println("File watch error:", err)
			}
		}
	}()

	if err := watcher.Add(cfg.configPath); err != nil {
		log.Fatal("error while adding config path:", cfg.configPath, "Error:", err)
	}

	<-ctx.Done()
}
