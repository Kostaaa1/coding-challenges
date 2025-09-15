package healthcheck

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Kostaaa1/loadbalancer/internal/config"
	"github.com/Kostaaa1/loadbalancer/internal/models"
)

type Checker struct {
	servers    []*models.Server
	cfg        config.HealthcheckConfig
	httpClient *http.Client
	logger     *slog.Logger

	parentCtx context.Context
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup

	isFirstLog atomic.Bool
	sync.RWMutex
}

func New(ctx context.Context, cfg *config.Config, logger *slog.Logger) *Checker {
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
		parentCtx:  ctx,
		servers:    cfg.Servers,
		cfg:        cfg.Healthcheck,
		logger:     logger,
		httpClient: httpClient,
	}
}

func (h *Checker) Restart(cfg *config.Config) {
	h.Lock()
	defer h.Unlock()

	h.Stop()
	h.wg.Wait()

	h.cfg = cfg.Healthcheck
	h.servers = cfg.Servers
	h.isFirstLog.Store(false)

	h.Start()
}

func (h *Checker) Start() {
	h.ctx, h.cancel = context.WithCancel(h.parentCtx)

	h.wg.Add(1)
	go func() {
		defer h.wg.Done()

		ticker := time.NewTicker(time.Second * time.Duration(h.cfg.Interval))
		defer ticker.Stop()

		h.run()

		for {
			select {
			case <-ticker.C:
				h.run()
			case <-h.ctx.Done():
				h.logger.Info("Health check stopped")
				return
			}
		}
	}()
}

func (h *Checker) Stop() {
	if h.cancel != nil {
		h.cancel()
	}
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
	urlPath, err := url.JoinPath(srv.URL, srv.HealthURL)
	if err != nil {
		h.logger.Error("failed to create url for healthchecking", "", srv.URL, "", srv.HealthURL, "msg", err)
		return
	}

	resp, err := h.httpClient.Get(urlPath)
	if err != nil {
		h.updateHealthStatus(srv, false)
		return
	}
	defer resp.Body.Close()

	h.updateHealthStatus(srv, resp.StatusCode == http.StatusOK)
}

func (h *Checker) updateHealthStatus(srv *models.Server, newStatus bool) {
	currentStatus := srv.IsHealthy()
	statusChanged := currentStatus != newStatus
	isFirstLog := !h.isFirstLog.Load()

	var logStatus string
	if isFirstLog {
		if newStatus {
			logStatus = "SERVER_HEALTHY"
		}
		if !newStatus {
			logStatus = "SERVER_UNHEALTHY"
		}
	} else {
		if newStatus && !currentStatus {
			logStatus = "SERVER_HEALTHY"
		}
		if !newStatus && currentStatus {
			logStatus = "SERVER_UNHEALTHY"
		}
	}

	if statusChanged {
		if !newStatus {
			srv.ActiveFailures++
			if srv.ActiveFailures >= h.cfg.UnhealthyThreshold {
				srv.SetHealthy(false)
			} else {
				logStatus = ""
			}
		} else {
			srv.ActiveSuccesses++
			if srv.ActiveSuccesses >= h.cfg.HealthyThreshold {
				srv.SetHealthy(true)
			} else {
				logStatus = ""
			}
		}
	}

	if logStatus != "" {
		h.logger.Info(
			logStatus,
			slog.Group("server", "address", srv.URL, "name", srv.Name, "healthy", newStatus),
		)
	}
}
