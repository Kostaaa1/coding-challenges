package loadbalancer

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Kostaaa1/loadbalancer/internal/models"
)

type Checker struct {
	servers    []*models.Server
	httpClient *http.Client
	logger     *slog.Logger
	isFirstLog atomic.Bool
	sync.RWMutex
}

func NewHealthchecker(servers []*models.Server, logger *slog.Logger) *Checker {
	httpClient := &http.Client{
		Timeout: time.Second * 3,
		Transport: &http.Transport{
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	return &Checker{
		servers:    servers,
		logger:     logger,
		httpClient: httpClient,
	}
}

func (h *Checker) UpdateServers(servers []*models.Server) {
	h.RLock()
	h.servers = servers
	h.isFirstLog.Store(false)
	h.RUnlock()
}

func (h *Checker) Start(interval int, ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Second * time.Duration(interval))
		defer ticker.Stop()

		h.run()

		for {
			select {
			case <-ticker.C:
				h.run()

			case <-ctx.Done():
				h.logger.Info("Health check stopped")
				return
			}
		}
	}()
}

func (h *Checker) run() {
	h.RLock()
	serversToCheck := make([]*models.Server, len(h.servers))
	copy(serversToCheck, h.servers)
	h.RUnlock()

	var wg sync.WaitGroup
	for _, srv := range serversToCheck {
		wg.Add(1)
		go func(s *models.Server) {
			defer wg.Done()
			h.check(s)
		}(srv)
	}
	wg.Wait()

	h.isFirstLog.Store(true)
}

func (h *Checker) check(srv *models.Server) {
	if !strings.HasPrefix(srv.HealthURL, "/") {
		srv.HealthURL = "/" + srv.HealthURL
	}
	u := fmt.Sprintf("%s%s", srv.URL, srv.HealthURL)

	resp, err := h.httpClient.Get(u)
	if err != nil {
		h.updateHealthStatus(srv, false)
		return
	}
	defer resp.Body.Close()

	h.updateHealthStatus(srv, resp.StatusCode == http.StatusOK)
}

func (h *Checker) updateHealthStatus(srv *models.Server, newStatus bool) {
	healthy := srv.IsHealthy()
	statusChanged := healthy != newStatus
	isFirstLog := !h.isFirstLog.Load()

	if statusChanged {
		srv.SetHealthy(newStatus)
	}

	var logStatus string

	if isFirstLog {
		if newStatus {
			logStatus = MarkedHealthy
		}
		if !newStatus {
			logStatus = MarkedUnhealthy
		}
	} else {
		if newStatus && !healthy {
			logStatus = MarkedHealthy
		}
		if !newStatus && healthy {
			logStatus = MarkedUnhealthy
		}
	}

	if logStatus != "" {
		h.logger.Info(
			logStatus,
			slog.Group("server", "address", srv.URL, "name", srv.Name, "healthy", newStatus),
		)
	}
}
