package loadbalancer

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/Kostaaa1/loadbalancer/internal/config"
	"github.com/Kostaaa1/loadbalancer/internal/models"
	"github.com/Kostaaa1/loadbalancer/strategy"
)

var (
	RequestForwarded = "REQUEST_FORWARDED"
	RequestFailed    = "REQUEST_FAILED"
	RequestRetry     = "REQUEST_RETRY"
	RequestReceived  = "REQUEST_RECEIVED"
	RequestCompleted = "REQUEST_COMPLETED"

	LoadbalancerStarted = "LOADBALANCER_STARTED"
	MarkedHealthy       = "SERVER_MARKED_HEALTHY"
	MarkedUnhealthy     = "SERVER_MARKED_UNHEALTHY"
	NoBackendAvailable  = "NO_BACKEND_AVAILABLE"
	NoHealthyBackend    = "NO_HEALTHY_BACKEND"
	BackendError        = "BACKEND_ERROR"

	defaultTransport = &http.Transport{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
)

// type ILoadbalancer interface {
// 	http.Handler
// }

type loadBalancer struct {
	servers  []*models.Server
	strategy strategy.ILBStrategy
	logger   *slog.Logger
	sync.RWMutex
}

type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (l *loadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	l.logger.Info(RequestReceived, "method", r.Method, "url", r.URL.String(), "client_ip", r.RemoteAddr)

	srv := l.Next(w, r)
	if srv == nil {
		l.logger.Warn(NoBackendAvailable, "url", r.URL.String())
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("503 Service Unavailable - No backend servers available"))
		return
	}

	l.logger.Info(RequestForwarded, "server", srv.URL, "client_ip", r.RemoteAddr)

	parsed, _ := url.Parse(srv.URL)
	proxy := httputil.NewSingleHostReverseProxy(parsed)
	proxy.Transport = defaultTransport
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		l.logger.Error(BackendError, "server", srv.URL, "client_ip", r.UserAgent(), "error", err.Error())
	}

	statusRecorder := &statusResponseWriter{ResponseWriter: w}
	proxy.ServeHTTP(statusRecorder, r)
	l.logger.Info(RequestCompleted, "server", srv.URL, "status", statusRecorder.status, "latency_ms", time.Since(start).Milliseconds())
}

func (l *loadBalancer) Next(w http.ResponseWriter, r *http.Request) *models.Server {
	l.RLock()
	defer l.RUnlock()
	return l.strategy.Next(w, r)
}

func (l *loadBalancer) AddServer(srv *models.Server) {
	l.Lock()
	defer l.Unlock()
	l.servers = append(l.servers, srv)
}

func (l *loadBalancer) Remove(srv *models.Server) {
	l.Lock()
	defer l.Unlock()
	for i, server := range l.servers {
		if server.Name == srv.Name {
			l.servers = append(l.servers[:i], l.servers[i+1:]...)
			return
		}
	}
}

func New(cfg *config.Config, logger *slog.Logger) *loadBalancer {
	lbStrategy := strategy.GetLBStrategy(cfg.Strategy, cfg.Servers)

	ch := NewHealthchecker(cfg.Servers, logger)
	ch.Start(context.Background(), cfg.HealthCheckIntervalSeconds)

	return &loadBalancer{
		servers:  cfg.Servers,
		strategy: lbStrategy,
		logger:   logger,
	}
}
