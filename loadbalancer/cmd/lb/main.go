package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Kostaaa1/loadbalancer/internal/config"
	loadbalancer "github.com/Kostaaa1/loadbalancer/internal/server"
)

func main() {
	cfgPath := flag.String("lb_config", "lb_config.json", "Path to load balancer config file (JSON | YAML)")
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
	mux.Handle("/", lb)

	srv := http.Server{
		Addr:         cfg.Port,
		Handler:      mux,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	shutdownErr := make(chan error, 1)

	go func() {
		sigs := make(chan os.Signal, 1)

		// Capture SIGHUP (reload config signal) / SIGTERM and SIGINT for graceful shutdown
		signal.Notify(sigs, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)

		for {
			sig := <-sigs
			fmt.Println("Received signal:", sig)

			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				logger.Info("")

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				shutdownErr <- srv.Shutdown(ctx)

			case syscall.SIGHUP:
				fmt.Println("Reloading config...")
				c, err := config.Load(*cfgPath)
				if err != nil {
					panic(err)
				}
				*cfg = *c
			}
		}
	}()

	logger.Info(loadbalancer.LoadbalancerStarted, "port", srv.Addr, "strategy", cfg.Strategy, "healtcheck_interval", cfg.HealthCheckIntervalSeconds)

	err = srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}

	err = <-shutdownErr
	if err != nil {
		log.Fatal(err)
	}
}
