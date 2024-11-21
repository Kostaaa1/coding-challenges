package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
)

type Server struct {
	Port   string
	Server *http.Server
}

func Healtcheck(w http.ResponseWriter, r *http.Request, addr string) {
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

func Shutdown(w http.ResponseWriter, r *http.Request, addr string) {
	port := strings.TrimPrefix(addr, ":")
	srv := findServerByAddr(port)
	if srv == nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server not found"))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	err := srv.Shutdown(ctx)
	if err != nil {
		if err != context.DeadlineExceeded {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error shutting down server"))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Server shut down successfully"))
		}
	}
}

func NewServer(port string) *Server {
	router := httprouter.New()
	router.HandlerFunc(http.MethodGet, "/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		Healtcheck(w, r, port)
	})
	router.HandlerFunc(http.MethodGet, "/", func(w http.ResponseWriter, r *http.Request) {
		Index(w, r, port)
	})
	router.HandlerFunc(http.MethodGet, "/shutdown", func(w http.ResponseWriter, r *http.Request) {
		Shutdown(w, r, port)
	})

	return &Server{
		Port: port,
		Server: &http.Server{
			Addr:         ":" + port,
			Handler:      router,
			IdleTimeout:  time.Minute,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
	}
}

func (s *Server) Start(wg *sync.WaitGroup) {
	defer wg.Done()
	slog.Info(fmt.Sprintf("Server started on port %s", s.Server.Addr))

	err := s.Server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Printf("Server on port %s failed: %v", s.Server.Addr, err)
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info(fmt.Sprintf("Shutting down server on port %s", s.Port))
	return s.Server.Shutdown(ctx)
}

var serverInstances []*Server

func main() {
	servers := []string{
		"8001",
		"8002",
		"8003",
	}

	var wg sync.WaitGroup
	for _, port := range servers {
		wg.Add(1)
		go func(port string) {
			server := NewServer(port)
			serverInstances = append(serverInstances, server)
			go server.Start(&wg)
		}(port)
	}

	wg.Wait()
}

func findServerByAddr(port string) *Server {
	for _, server := range serverInstances {
		if server.Port == port {
			return server
		}
	}
	return nil
}

func removeServerByAddr(port string) *Server {
	for i, server := range serverInstances {
		if server.Port == port {
			serverInstances = append(serverInstances[:i], serverInstances[i+1:]...)
		}
	}
	return nil
}
