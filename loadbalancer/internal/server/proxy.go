package loadbalancer

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/Kostaaa1/loadbalancer/internal/balancer"
	"github.com/Kostaaa1/loadbalancer/internal/config"
	"github.com/Kostaaa1/loadbalancer/internal/models"
)

var (
	RequestForwarded    = "REQUEST_FORWARDED"
	RequestFailed       = "REQUEST_FAILED"
	RequestRetry        = "REQUEST_RETRY"
	RequestReceived     = "REQUEST_RECEIVED"
	RequestCompleted    = "REQUEST_COMPLETED"
	LoadbalancerStarted = "LOADBALANCER_STARTED"
	MarkedHealthy       = "SERVER_MARKED_HEALTHY"
	MarkedUnhealthy     = "SERVER_MARKED_UNHEALTHY"
	NoBackendAvailable  = "NO_BACKEND_AVAILABLE"
	NoHealthyBackend    = "NO_HEALTHY_BACKEND"
	BackendError        = "BACKEND_ERROR"
	ConfigReload        = "CONFIG_RELOAD"

	defaultTransport = &http.Transport{
		MaxIdleConns:          1000,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    true,
	}
)

type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

type LB struct {
	proxies  map[string]*httputil.ReverseProxy
	strategy balancer.ILBStrategy
	checker  *Checker
	cfg      *config.Config
	logger   *slog.Logger
	sync.Mutex
}

func (l *LB) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	l.logger.Info(RequestReceived, "method", r.Method, "url", r.URL.String(), "client_ip", r.RemoteAddr)

	srv := l.Next(w, r)

	if srv == nil {
		l.logger.Warn(NoBackendAvailable, "url", r.URL.String())
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("503 Service Unavailable - No backend servers available"))
		return
	}

	if l.cfg.Strategy == balancer.LeastConnectionsStrategy {
		srv.ConnCount.Add(1)
		defer srv.ConnCount.Add(-1)
	}

	l.logger.Info(RequestForwarded, "server", srv.URL, "client_ip", r.RemoteAddr)

	statusRecorder := &statusResponseWriter{ResponseWriter: w}
	proxy := l.proxies[srv.URL]
	proxy.ServeHTTP(statusRecorder, r)

	l.logger.Info(RequestCompleted, "server", srv.Name, "server_url", srv.URL, "status", statusRecorder.status, "latency_ms", time.Since(start).Milliseconds())
}

func (l *LB) Next(w http.ResponseWriter, r *http.Request) *models.Server {
	l.Lock()
	defer l.Unlock()
	return l.strategy.Next(w, r)
}

func (l *LB) SetConfig(cfg *config.Config) error {
	l.Lock()
	defer l.Unlock()

	l.cfg = cfg
	lbStrategy, err := balancer.GetLBStrategy(cfg.Strategy, cfg.Servers)
	if err != nil {
		return err
	}
	l.strategy = lbStrategy
	l.strategy.UpdateServers(cfg.Servers)
	l.checker.UpdateServers(cfg.Servers)

	proxies := make(map[string]*httputil.ReverseProxy, len(cfg.Servers))
	for _, srv := range cfg.Servers {
		l.addProxy(proxies, srv)
	}
	l.proxies = proxies

	return nil
}

func (l *LB) addProxy(proxies map[string]*httputil.ReverseProxy, srv *models.Server) {
	parsed, _ := url.Parse(srv.URL)
	proxy := httputil.NewSingleHostReverseProxy(parsed)
	proxy.Transport = defaultTransport
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		l.logger.Error(BackendError, "msg", err, "server_url", srv.URL, "client_ip", r.UserAgent())
		srv.SetHealthy(false)
	}
	proxies[srv.URL] = proxy
}

func (l *LB) SetLogger(logger *slog.Logger) {
	l.Lock()
	defer l.Unlock()
	l.logger = logger
}

func New(cfg *config.Config, logger *slog.Logger, ctx context.Context) (*LB, error) {
	lbStrategy, err := balancer.GetLBStrategy(cfg.Strategy, cfg.Servers)
	if err != nil {
		return nil, err
	}

	checker := NewHealthchecker(cfg.Servers, logger)
	checker.Start(cfg.HealthCheckIntervalSeconds, ctx)
	proxies := make(map[string]*httputil.ReverseProxy, len(cfg.Servers))

	lb := &LB{
		cfg:      cfg,
		strategy: lbStrategy,
		checker:  checker,
		logger:   logger,
		proxies:  proxies,
	}

	for _, srv := range cfg.Servers {
		lb.addProxy(lb.proxies, srv)
	}

	return lb, nil
}
