package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Kostaaa1/loadbalancer/internal/config"
	loadbalancer "github.com/Kostaaa1/loadbalancer/lb"
)

func main() {
	cfg, err := config.Load("lb_config.json")
	if err != nil {
		panic(err)
	}

	fmt.Println(cfg.Servers)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	lb := loadbalancer.New(cfg, logger)

	mux := http.NewServeMux()
	mux.Handle("/", lb)
	mux.HandleFunc("/lb-add-server", lb.AddServerHandler)

	srv := http.Server{
		Addr:         cfg.Port,
		Handler:      mux,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	logger.Info(loadbalancer.LoadbalancerStarted, "port", srv.Addr, "strategy", cfg.Strategy, "healtcheck_interval", cfg.HealthCheckIntervalSeconds)
	log.Fatal(srv.ListenAndServe())
}
