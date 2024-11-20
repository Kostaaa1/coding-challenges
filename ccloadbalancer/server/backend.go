package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type Server struct {
	Port   string
	Server *http.Server
}

func Healthcheck(w http.ResponseWriter, r *http.Request, addr string) {
	log.Printf("Received request %s %s %s", r.Method, r.URL.Path, addr)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("Hello from backend server."))
}

func Index(w http.ResponseWriter, r *http.Request, addr string) {
	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, "unable to load template", http.StatusInternalServerError)
		return
	}

	data := struct {
		Addr string
	}{
		Addr: addr,
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "unable to render template", http.StatusInternalServerError)
	}
}

func NewServer(addr string) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		Healthcheck(w, r, addr)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		Index(w, r, addr)
	})
	return &Server{
		Port: addr,
		Server: &http.Server{
			Addr:         addr,
			Handler:      mux,
			IdleTimeout:  time.Minute,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
	}
}

func (s *Server) Start(wg *sync.WaitGroup) {
	defer wg.Done()
	slog.Info(fmt.Sprintf("server started on port %s", s.Server.Addr))
	log.Fatal(s.Server.ListenAndServe())
}

func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down server on %s", "port", s.Port)
	return s.Server.Shutdown(ctx)
}

func main() {
	addrs := []string{
		":8001",
		":8003",
	}

	var wg sync.WaitGroup

	for _, addr := range addrs {
		wg.Add(1)
		go func(addr string) {
			server := NewServer(addr)
			server.Start(&wg)
		}(addr)
	}

	wg.Wait()
}
