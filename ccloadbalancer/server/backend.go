package main

import (
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// func Healthcheck(w http.ResponseWriter, r *http.Request) {
// 	if r.Method == http.MethodGet {
// 		fmt.Println(r.Method)
// 		b, err := json.Marshal(map[string]string{
// 			"status":  "available",
// 			"message": "kobas",
// 		})
// 		if err != nil {
// 			w.WriteHeader(http.StatusInternalServerError)
// 			w.Write([]byte("error while marshaling"))
// 		}
// 		w.WriteHeader(http.StatusOK)
// 		w.Header().Set("Content-Type", "application/json")
// 		w.Write(b)
// 	}
// }

func Healthcheck(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request: %s %s", r.Method, r.URL.Path)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("Hello from backend server."))
}

type PageData struct {
	Addr string
}

func Index(w http.ResponseWriter, r *http.Request, addr string) {
	fmt.Println("Request on port: ", addr)

	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, "unable to load template", http.StatusInternalServerError)
		return
	}

	data := PageData{
		Addr: addr,
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "unable to render template", http.StatusInternalServerError)
	}
}

func main() {
	addrs := []string{
		":8001",
		":8002",
		":8003",
	}

	var wg sync.WaitGroup

	for _, addr := range addrs {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()

			mux := http.NewServeMux()
			mux.HandleFunc("/healthcheck", Healthcheck)
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				Index(w, r, addr)
			})

			srv := http.Server{
				Addr:         addr,
				Handler:      mux,
				IdleTimeout:  time.Minute,
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 10 * time.Second,
			}

			slog.Info(fmt.Sprintf("server started on port %s", srv.Addr))
			log.Fatal(srv.ListenAndServe())
		}(addr)
	}
	wg.Wait()
}
