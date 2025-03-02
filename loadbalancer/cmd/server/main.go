package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type envelope map[string]interface{}

func writeJSON(w http.ResponseWriter, status int, data interface{}, headers http.Header) error {
	js, err := json.Marshal(data)
	if err != nil {
		return err
	}

	for k, v := range headers {
		w.Header()[k] = v
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_, err = w.Write(js)
	return err
}

func main() {
	var port string
	flag.StringVar(&port, "port", "8080", "")
	flag.Parse()

	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("received healthcheck request")
		data := envelope{
			"status": "available",
		}
		if err := writeJSON(w, http.StatusOK, data, nil); err != nil {
			logger.Error("failed to write healtcheck response", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(filepath.Join("cmd", "server", "templates", "index.html"))
		if err != nil {
			fmt.Println(err)
			http.Error(w, "error loading template", http.StatusInternalServerError)
			return
		}

		data := struct {
			Port string
		}{
			Port: port,
		}

		w.Header().Set("Content-Type", "text/html")

		if err := tmpl.Execute(w, data); err != nil {
			fmt.Println("failed to execute tmpl!", err)
		}
	})

	srv := http.Server{
		Addr:         port,
		Handler:      mux,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	logger.Info("backend server running on", "port", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
