package lb

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/Kostaaa1/loadbalancer/internal/models"
)

type Checker struct {
	servers    []*models.Server
	httpClient *http.Client
	logger     *slog.Logger
	// sync.RWMutex
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

func (h *Checker) Start(ctx context.Context, interval int) {
	go func() {
		ticker := time.NewTicker(time.Second * time.Duration(interval))
		defer ticker.Stop()

		h.run()

		for {
			select {
			case <-ticker.C:
				h.run()
			case <-ctx.Done():
				fmt.Println("Health check stopped")
				return
			}
		}
	}()
}

func (h *Checker) run() {
	var wg sync.WaitGroup

	for _, srv := range h.servers {
		wg.Add(1)
		go func(srv *models.Server) {
			defer wg.Done()
			h.check(srv)
		}(srv)
	}

	wg.Wait()
}

// TODO: reduce unnecessary checks for the servers that are unhealthy
func (h *Checker) check(srv *models.Server) {
	u := fmt.Sprintf("%s%s", srv.URL, srv.HealthURL)

	resp, err := h.httpClient.Get(u)
	if err != nil {
		h.updateHealthStatus(srv, false)
		return
	}
	defer resp.Body.Close()

	h.updateHealthStatus(srv, resp.StatusCode == http.StatusOK)
}

func (h *Checker) updateHealthStatus(srv *models.Server, status bool) {
	srv.Lock()
	defer srv.Unlock()

	if srv.Healthy == status {
		return
	}

	var logStatus string
	if status && !srv.Healthy {
		logStatus = MarkedHealthy
	}

	if !status && srv.Healthy {
		logStatus = MarkedUnhealthy
	}

	srv.Healthy = status

	h.logger.Info(
		logStatus,
		slog.Group("server", "address", srv.URL, "name", srv.Name, "healthy", srv.Healthy),
	)
}
