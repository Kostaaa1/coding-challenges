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
	loadbalancer "github.com/Kostaaa1/loadbalancer/internal/server"
)

func signalListener(srv *http.Server, lb *loadbalancer.LB, logger *slog.Logger, cfgPath string, cancel context.CancelFunc) {
	// Capture SIGHUP (reload config signal) / SIGTERM and SIGINT for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)

	var shutdownOnce sync.Once

	for {
		sig := <-sigs

		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			shutdownOnce.Do(func() {
				logger.Info("Gracefully shutting down...")
				cancel()

				ctx, cancelTimeout := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancelTimeout()

				if err := srv.Shutdown(ctx); err != nil {
					logger.Error("Error shutting down server", "error", err)
				}
			})
			return
		case syscall.SIGHUP:
			logger.Info(loadbalancer.ConfigReload, "msg", "Received SIGHUP signal, reloading config...")
			newCfg, err := config.Load(cfgPath)
			if err != nil {
				log.Fatal(err)
			}
			if err = lb.SetConfig(newCfg); err != nil {
				log.Fatal(err)
			}
		}
	}
}

func main() {
	cfgPath := flag.String("config_path", "lb_config.json", "Path to load balancer config file (JSON | YAML)")
	watchConfig := flag.Bool("config_watch", true, "Watching for config writes")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	lb, err := loadbalancer.New(cfg, logger, ctx)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", lb) /* TODO: add rate limit */

	srv := &http.Server{
		Addr:         cfg.Port,
		Handler:      mux,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go signalListener(srv, lb, logger, *cfgPath, cancel)
	if *watchConfig {
		go cfg.Watch(ctx)
	}

	logger.Info(loadbalancer.LoadbalancerStarted, "port", srv.Addr, "strategy", cfg.Strategy, "healtcheck_interval", cfg.HealthCheckIntervalSeconds)

	err = srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}

	<-ctx.Done()
	logger.Info("Shutdown complete")
}
