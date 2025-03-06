package loadbalancer

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Kostaaa1/loadbalancer/internal/config"
	"github.com/Kostaaa1/loadbalancer/internal/models"
	"github.com/Kostaaa1/loadbalancer/internal/strategy"
)

var (
	RequestForwarded      = "REQUEST_FORWARDED"
	RequestFailed         = "REQUEST_FAILED"
	RequestRetry          = "REQUEST_RETRY"
	RequestReceived       = "REQUEST_RECEIVED"
	MaxConnectionExceeded = "MAX_CONNECTIONS_EXCEEDED"
	RequestCompleted      = "REQUEST_COMPLETED"
	LoadbalancerStarted   = "LOADBALANCER_STARTED"

	MarkedHealthy   = "SERVER_MARKED_HEALTHY"
	MarkedUnhealthy = "SERVER_MARKED_UNHEALTHY"

	NoBackendAvailable = "NO_BACKEND_AVAILABLE"
	NoHealthyBackend   = "NO_HEALTHY_BACKEND"
	BackendError       = "BACKEND_ERROR"
	ConfigReload       = "CONFIG_RELOAD"

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
	strategy strategy.ILBStrategy
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

	atomic.AddInt64(&srv.TotalConns, 1)

	activeConns := atomic.AddInt64(&srv.ActiveConns, 1)
	defer atomic.AddInt64(&srv.ActiveConns, -1)

	if activeConns == srv.MaxConnections {
		l.logger.Warn(MaxConnectionExceeded,
			"server", srv.Name,
			"active_conns", activeConns,
			"max_conns", srv.MaxConnections,
			"client_ip", r.RemoteAddr)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("503 Service Unavailable - Server at capacity"))
		return
	}

	l.logger.Info(RequestForwarded, "server", srv.URL, "client_ip", r.RemoteAddr, "active_conns", atomic.LoadInt64(&srv.ActiveConns))

	statusRecorder := &statusResponseWriter{ResponseWriter: w}
	proxy := l.proxies[srv.URL]
	proxy.ServeHTTP(statusRecorder, r)

	// handle passive checks (avoid disabling server if error occurs)
	if statusRecorder.status >= 500 {
		srv.SetHealthy(false)
	}

	l.logger.Info(
		RequestCompleted,
		"server", srv.Name,
		"status", statusRecorder.status,
		"healthy", srv.IsHealthy(),
		"active_conns", atomic.LoadInt64(&srv.ActiveConns),
		"total_conns", atomic.LoadInt64(&srv.TotalConns),
		"latency_ms", time.Since(start).Milliseconds(),
	)
}

func (l *LB) Next(w http.ResponseWriter, r *http.Request) *models.Server {
	l.Lock()
	defer l.Unlock()
	return l.strategy.Next()
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

func (l *LB) SetConfig(cfg *config.Config) error {
	l.Lock()
	defer l.Unlock()

	l.cfg = cfg

	lbStrategy, err := strategy.GetFromConfig(cfg)
	if err != nil {
		return err
	}
	l.strategy = lbStrategy
	l.strategy.UpdateServers(cfg.Servers)

	// l.checker.UpdateServers(cfg.Servers)
	// l.checker.Restart(l.checkerCtx, cfg)

	proxies := make(map[string]*httputil.ReverseProxy, len(cfg.Servers))
	for _, srv := range cfg.Servers {
		l.addProxy(proxies, srv)
	}
	l.proxies = proxies

	return nil
}

func New(cfg *config.Config, logger *slog.Logger) (*LB, error) {
	lbStrategy, err := strategy.GetFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	proxies := make(map[string]*httputil.ReverseProxy, len(cfg.Servers))

	lb := &LB{
		cfg:      cfg,
		strategy: lbStrategy,
		logger:   logger,
		proxies:  proxies,
	}

	for _, srv := range cfg.Servers {
		lb.addProxy(lb.proxies, srv)
	}

	return lb, nil
}
