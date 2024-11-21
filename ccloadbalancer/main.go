package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type config struct {
	port       int
	pingPeriod int
}

type proxy struct {
	addr    string
	isAlive bool
}

type LBServer struct {
	config
	ln            net.Listener
	proxies       []*proxy
	activeProxyID int
	ticker        *time.Ticker
	client        *http.Client
	mu            sync.Mutex
}

func NewLBServer(cfg config) *LBServer {
	return &LBServer{
		config: cfg,
		ticker: time.NewTicker(time.Duration(cfg.pingPeriod) * time.Second),
		client: &http.Client{Timeout: 5 * time.Second},
		proxies: []*proxy{
			{
				addr:    ":8001",
				isAlive: false,
			},
			{
				addr:    ":8002",
				isAlive: false,
			},
			{
				addr:    ":8003",
				isAlive: false,
			},
		},
	}
}

func (lb *LBServer) Next() {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for i := 0; i < len(lb.proxies); i++ {
		next := (lb.activeProxyID + 1 + i) % len(lb.proxies)
		if lb.proxies[next].isAlive {
			lb.activeProxyID = next
			return
		}
	}
}

func (lb *LBServer) sendHeathcheck(p *proxy) {
	resp, err := lb.client.Get(fmt.Sprintf("http://localhost%s/healthcheck", p.addr))
	if err != nil {
		// TODO: Dodaj eksponencijalni backoff
		p.isAlive = false
		fmt.Printf("Port%s | Alive: %t\n", p.addr, p.isAlive)
		return
	}

	p.isAlive = resp.StatusCode == http.StatusOK
	fmt.Printf("Port%s | Alive: %t\n", p.addr, p.isAlive)
}

func (lb *LBServer) healthChecker() {
	for range lb.ticker.C {
		var wg sync.WaitGroup
		for i := range lb.proxies {
			wg.Add(1)
			go func(p *proxy) {
				defer wg.Done()
				lb.sendHeathcheck(p)
			}(lb.proxies[i])
		}
		wg.Wait()
	}
}

func (lb *LBServer) handleConn(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 4096)

	n, err := conn.Read(buffer)
	if err != nil {
		if err == io.EOF {
			log.Println("Connection closed by client")
		} else {
			slog.Error("read error", "err", err)
		}
		return
	}

	data := buffer[:n]

	lb.Next()
	proxy := lb.proxies[lb.activeProxyID]

	if proxy == nil {
		fmt.Println("No avaiable proxies", lb.proxies[0], lb.proxies[1])
		writeError(conn, http.StatusServiceUnavailable, "No avaiable proxies", nil)
		return
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost%s", proxy.addr), bytes.NewReader(data))
	if err != nil {
		writeError(conn, http.StatusInternalServerError, "Failed to create request:", err)
		return
	}

	resp, err := lb.client.Do(req)
	if err != nil {
		writeError(conn, http.StatusBadGateway, "Backend connection failed", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeError(conn, http.StatusInternalServerError, "Failed to read backend response", err)
		return
	}

	writeResponse(conn, resp, body)
	log.Println("Request received from backend and response sent to client.")
}

func (lb *LBServer) Start() error {
	addr := fmt.Sprintf(":%d", lb.port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	lb.ln = ln
	slog.Info("server started on", "port", lb.config.port)

	go lb.healthChecker()

	for {
		conn, err := lb.ln.Accept()
		if err != nil {
			slog.Error("accept error", "err", err)
			continue
		}
		go lb.handleConn(conn)
	}
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 8000, "")
	flag.IntVar(&cfg.pingPeriod, "ping-period", 7, "")
	flag.Parse()

	lb := NewLBServer(cfg)
	if err := lb.Start(); err != nil {
		log.Fatalf("error starting server: %v", err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop

	log.Println("Received termination signal, shutting down...")
	if err := lb.ln.Close(); err != nil {
		fmt.Println("error while closing the listener: %w", err)
	}
}
