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

func signalListener(srv *http.Server, lb *loadbalancer.LB, logger *slog.Logger, cfgPath string, done chan error) {
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
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				done <- srv.Shutdown(ctx)
				close(done)
				close(sigs)
			})
			return
		case syscall.SIGHUP:
			logger.Info(loadbalancer.ConfigReload, "msg", "Received SIGHUP signal, reloading config...")
			newCfg, err := config.Load(cfgPath)
			if err != nil {
				panic(err)
			}
			lb.SetConfig(newCfg)
		}
	}
}

func main() {
	cfgPath := flag.String("config_path", "lb_config.json", "Path to load balancer config file (JSON | YAML)")
	watchConfig := flag.Bool("config_watch", true, "Watching for config writes")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		panic(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	lb, err := loadbalancer.New(cfg, logger)
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", lb) // TODO: add rate limit

	// cert, err := tls.LoadX509KeyPair("./cert/server.crt", "./cert/server.key")
	// if err != nil {
	// 	log.Fatalf("Failed to load certificates: %v", err)
	// }
	// var ticket [32]byte
	// rand.Read(ticket[:])
	// tlsConfig := &tls.Config{
	// 	Certificates: []tls.Certificate{cert},
	// 	MinVersion:   tls.VersionTLS12,
	// 	GetConfigForClient: func(*tls.ClientHelloInfo) (*tls.Config, error) {
	// 		return &tls.Config{
	// 			SessionTicketsDisabled: false,
	// 			SessionTicketKey:       ticket,
	// 		}, nil
	// 	},
	// }

	srv := &http.Server{
		Addr:         ":443",
		Handler:      mux,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		// TLSConfig:    tlsConfig,
	}

	done := make(chan error, 1)

	go signalListener(srv, lb, logger, *cfgPath, done)
	if *watchConfig {
		go cfg.Watch(done)
	}

	logger.Info(loadbalancer.LoadbalancerStarted, "port", srv.Addr, "strategy", cfg.Strategy, "healtcheck_interval", cfg.HealthCheckIntervalSeconds)

	err = srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}

	err = <-done
	if err != nil {
		log.Fatal(err)
	}
}
