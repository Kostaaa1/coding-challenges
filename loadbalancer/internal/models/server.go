package models

import (
	"sync"
	"time"
)

type Server struct {
	Name           string `json:"name" yaml:"name"`
	URL            string `json:"url" yaml:"url"`
	HealthURL      string `json:"health_url" yaml:"health_url"`
	Weight         int    `json:"weight"`
	MaxConnections int64  `json:"max_connections" yaml:"max_connections"`
	SlowStart      int    `json:"slow_start" yaml:"slow_start"`
	// LastFailed       time.Time
	// FailTimeout      time.Duration `json:"fail_timeout" yaml:"fail_timeout"`
	// MaxFails         int `json:"max_fails" yaml:"max_fails"`
	// PassiveFailures  int
	// PassiveSuccesses int // TODO
	ActiveFailures  int
	ActiveSuccesses int
	// LastActiveCheck  time.Time
	// LastPassiveCheck time.Time
	ActiveConns  int64
	TotalConns   int64
	ResponseTime time.Duration `json:"response_time"`
	healthy      bool
	sync.RWMutex
}

func (srv *Server) IsHealthy() bool {
	srv.Lock()
	defer srv.Unlock()
	return srv.healthy
}

func (srv *Server) SetHealthy(status bool) {
	srv.Lock()
	defer srv.Unlock()

	if status {
		srv.ActiveSuccesses = 0
		// srv.PassiveSuccesses = 0
	} else {
		srv.ActiveFailures = 0
		// srv.PassiveFailures = 0
	}
	srv.healthy = status
}
