package main

import (
	"context"
	"fmt"
	"time"
)

const PORT = "6969"

func main() {
	loads := make([]*Payload, 25)
	for i := range loads {
		loads[i] = &Payload{}
	}
	BurstRateLimitCall(context.Background(), loads, 20)

	// mux := http.NewServeMux()

	// mux.HandleFunc("/limited", func(w http.ResponseWriter, r *http.Request) {
	// 	w.Write([]byte("Limited, Let's Go!\n"))
	// })

	// mux.HandleFunc("/unlimited", func(w http.ResponseWriter, r *http.Request) {
	// 	w.Write([]byte("Unlimited, Let's Go!\n"))
	// })

	// TODO: Learn about some of the http.Server properties and experiment with them
	// srv := http.Server{
	// 	Addr:    ":" + PORT,
	// 	Handler: mux,
	// }

	// log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// log.Info("server started", "port", PORT, "status", "healthy")
	// panic(srv.ListenAndServe())
}

// Example of rate limiting with time.Ticker

const rateLimit = time.Second / 10

type Payload struct{}

func RateLimitCall(loads []Payload) {
	throttle := time.Tick(rateLimit)
	for i, load := range loads {
		<-throttle
		fmt.Printf("load %d called: %s\n", i, load)
	}
}

// allows burst rate limiting by adding buffer to the throttle
// BurstRateLimitCall allows burst rate limiting client calls with the
// payloads.
func BurstRateLimitCall(ctx context.Context, payloads []*Payload, burstLimit int) {
	throttle := make(chan time.Time, burstLimit)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		ticker := time.NewTicker(rateLimit)
		defer ticker.Stop()
		for t := range ticker.C {
			select {
			case throttle <- t:
			case <-ctx.Done():
				return // exit goroutine when surrounding function returns
			}
		}
	}()

	for i, payload := range payloads {
		<-throttle
		fmt.Printf("load %d called: %s\n", i, payload)
	}
}
