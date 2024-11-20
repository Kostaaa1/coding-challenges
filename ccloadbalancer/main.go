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
	"syscall"
	"time"
)

type config struct {
	port       int
	pingPeriod int
}

type proxy struct {
	addr     string
	isAlive  bool
	isActive bool
}

type LBServer struct {
	config
	ln      net.Listener
	proxies []*proxy
	ticker  *time.Ticker
	client  *http.Client
}

func spawnServers() []*proxy {
	servers := []*proxy{
		&proxy{
			addr:     ":8001",
			isAlive:  true,
			isActive: false,
		},
		&proxy{
			addr:     ":8003",
			isAlive:  true,
			isActive: true,
		},
	}
	return servers
}

func NewLBServer(cfg config) *LBServer {
	return &LBServer{
		config:  cfg,
		proxies: spawnServers(),
		ticker:  time.NewTicker(time.Duration(cfg.pingPeriod) * time.Second),
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (lb *LBServer) Next() {
	id := -1
	for i, p := range lb.proxies {
		if p.isActive {
			id = i
			p.isActive = false
			break
		}
	}

	if id == -1 {
		fmt.Println("Handle this")
		return
	}

	next := (id + 1) % len(lb.proxies)
	nextProxy := lb.proxies[next]
	if nextProxy.isAlive {
		nextProxy.isActive = true
	}
}

func (lb *LBServer) healthChecker() {
	for {
		select {
		case <-lb.ticker.C:
			// send req concurrently?
			for i := range lb.proxies {
				p := lb.proxies[i]
				resp, err := lb.client.Get(fmt.Sprintf("http://localhost%s/healthcheck", p.addr))
				if err != nil {
					p.isActive = false
					p.isAlive = false
					fmt.Printf("Port: %s | Active: %t | p.isAlive: %t\n", p.addr, p.isActive, p.isAlive)
					continue
				}

				if resp.StatusCode != http.StatusOK {
					p.isAlive = false
				}

				if resp.StatusCode == http.StatusOK && !p.isAlive {
					p.isAlive = true
				}
				fmt.Printf("Port: %s | Active: %t | p.isAlive: %t\n", p.addr, p.isActive, p.isAlive)
			}
		}
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

	proxy := getActiveProxy(lb.proxies)
	fmt.Println("PROXIES: ", lb.proxies, "ACtive:", proxy)

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
	log.Println("Request forwarded to backend and response sent to client.")
}

func (lb *LBServer) Start() error {
	addr := fmt.Sprintf(":%d", lb.port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	go lb.healthChecker()

	lb.ln = ln
	slog.Info("server started on", "port", lb.config.port)

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
	flag.IntVar(&cfg.pingPeriod, "ping-period", 6, "")
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
		log.Printf("error while closing the listener: %w", err)
	}
}
