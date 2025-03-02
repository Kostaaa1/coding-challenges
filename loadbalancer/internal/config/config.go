package config

import (
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
	configPath                 string
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

func (cfg *Config) Watch(done chan error) {
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
			case <-done:
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
						fmt.Println("Write event notice")
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

	<-done
}
