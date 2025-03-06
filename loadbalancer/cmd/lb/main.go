package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Kostaaa1/loadbalancer/internal/config"
	"github.com/Kostaaa1/loadbalancer/internal/healthcheck"
	loadbalancer "github.com/Kostaaa1/loadbalancer/internal/server"
)

type Server struct {
	cfg     *config.Config
	lb      *loadbalancer.LB
	checker *healthcheck.Checker
	httpSrv *http.Server
	logger  *slog.Logger
	cfgPath string
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewLBServer(cfgPath string, logger *slog.Logger) (*Server, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	lb, err := loadbalancer.New(cfg, logger)
	if err != nil {
		cancel()
		return nil, err
	}

	checker := healthcheck.New(ctx, cfg, logger)

	mux := http.NewServeMux()
	mux.Handle("/", lb) /* TODO: add rate limit */

	httpSrv := &http.Server{
		Addr:         cfg.Port,
		Handler:      mux,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return &Server{
		cfg:     cfg,
		lb:      lb,
		checker: checker,
		httpSrv: httpSrv,
		logger:  logger,
		cfgPath: cfgPath,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

func (s *Server) Start(watchConfig bool) error {
	s.checker.Start()

	go s.handleSignals()
	if watchConfig {
		go s.cfg.Watch(s.ctx)
	}

	s.logger.Info(loadbalancer.LoadbalancerStarted,
		"port", s.httpSrv.Addr,
		"strategy", s.cfg.Strategy,
		"healthcheck_interval", s.cfg.Healthcheck.Interval)

	err := s.httpSrv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	<-s.ctx.Done()
	s.logger.Info("Shutdown complete")
	return nil
}

func (s *Server) handleSignals() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)

	var shutdownOnce sync.Once
	for {
		sig := <-sigs
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			shutdownOnce.Do(func() {
				s.shutdown()
			})
			return
		case syscall.SIGHUP:
			s.reloadConfig()
		}
	}
}

func (s *Server) shutdown() {
	s.logger.Info("Gracefully shutting down...")
	s.cancel()

	ctx, cancelTimeout := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelTimeout()

	if err := s.httpSrv.Shutdown(ctx); err != nil {
		s.logger.Error("Error shutting down server", "error", err)
	}
}

func (s *Server) reloadConfig() {
	s.logger.Info(loadbalancer.ConfigReload, "msg", "Received SIGHUP signal, reloading config...")

	newCfg, err := config.Load(s.cfgPath)
	if err != nil {
		s.logger.Error("Failed to reload config", "error", err)
		return
	}

	s.checker.Restart(newCfg)
	if err = s.lb.SetConfig(newCfg); err != nil {
		s.logger.Error("Failed to update load balancer config", "error", err)
		return
	}

	s.cfg = newCfg
}

func main() {
	cfgPath := flag.String("config_path", "lb_config.json", "Path to load balancer config file (JSON | YAML)")
	watchConfig := flag.Bool("config_watch", true, "Watching for config writes")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	server, err := NewLBServer(*cfgPath, logger)
	if err != nil {
		log.Fatal(err)
	}

	if err := server.Start(*watchConfig); err != nil {
		log.Fatal(err)
	}
}
