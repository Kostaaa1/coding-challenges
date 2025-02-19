package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Kostaaa1/loadbalancer/internal/config"
	"github.com/Kostaaa1/loadbalancer/lb"
)

func main() {
	cfg, err := config.Load("/home/kosta/go/src/loadbalancer/config.json")
	if err != nil {
		panic(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	loadbalancer := lb.NewLoadbalancer(cfg, logger)

	srv := http.Server{
		Addr:         cfg.Port,
		Handler:      loadbalancer,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	logger.Info(lb.LoadbalancerStarted, "port", srv.Addr, "strategy", cfg.Strategy, "healtcheck_interval", cfg.HealthCheckIntervalSeconds)
	log.Fatal(srv.ListenAndServe())
}
