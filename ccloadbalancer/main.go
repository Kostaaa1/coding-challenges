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
	"time"
)

type config struct {
	port        int
	serverCount int
	pingPeriod  int
}

type proxy struct {
	addr     string
	isAlive  bool
	isActive bool
}

func getActiveProxy(proxies []*proxy) *proxy {
	for _, p := range proxies {
		if p.isActive {
			return p
		}
	}
	return nil
}

type server struct {
	config
	ln      net.Listener
	proxies []*proxy
	ticker  *time.Ticker
	client  *http.Client
}

func (s *server) Next() {
	var id int
	for i := range s.proxies {
		if s.proxies[i].isActive && s.proxies[i].isAlive {
			id = i
			break
		}
	}
	s.proxies[id].isActive = false
	id = (id + 1) % len(s.proxies)
	s.proxies[id].isActive = true
}

func NewServer(cfg config) *server {
	servers := make([]*proxy, cfg.serverCount)
	for i := range servers {
		servers[i] = &proxy{addr: fmt.Sprintf("http://localhost:800%d", i)}
	}

	duration := time.Duration(cfg.pingPeriod) * time.Second
	return &server{
		config:  cfg,
		proxies: servers,
		ticker:  time.NewTicker(duration),
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (s *server) Start() error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return err
	}

	go s.healthChecker()

	s.ln = ln
	slog.Info("server started on port", "Portess", s.config.port)
	return s.acceptLoop()
}

func (s *server) healthChecker() {
	for {
		select {
		case <-s.ticker.C:
			fmt.Println("tick")

			for i := range s.proxies {
				p := s.proxies[i]
				resp, err := s.client.Get(fmt.Sprintf("%s/healthcheck", p.addr))
				if err != nil {
					slog.Error("failed to get the response from healtchecker", "err", err)
					continue
				}

				if resp.StatusCode != http.StatusOK {
					p.isAlive = false
				}
			}
		}
	}
}

func (s *server) acceptLoop() error {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			slog.Error("accept error", "err", err)
			continue
		}
		go s.handleConn(conn)
	}
}

func (s *server) handleConn(conn net.Conn) {
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

	s.Next()

	proxy := getActiveProxy(s.proxies)

	req, err := http.NewRequest("GET", proxy.addr, bytes.NewReader(data))
	if err != nil {
		log.Println("Failed to create request:", err)
		return
	}

	resp, err := s.client.Do(req)
	if err != nil {
		log.Println("Failed to forward request:", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Failed to read response body:", err)
		return
	}

	conn.Write([]byte(fmt.Sprintf("HTTP/1.1 %d %s\r\n", resp.StatusCode, http.StatusText(resp.StatusCode))))
	for key, values := range resp.Header {
		for _, value := range values {
			conn.Write([]byte(fmt.Sprintf("%s: %s\r\n", key, value)))
		}
	}
	conn.Write([]byte("\r\n"))
	conn.Write(body)

	log.Println("Request forwarded to backend and response sent to client.")
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 8000, "")
	flag.IntVar(&cfg.pingPeriod, "ping-period", 6, "")
	flag.IntVar(&cfg.serverCount, "server-count", 4, "")
	flag.Parse()

	s := NewServer(cfg)
	if err := s.Start(); err != nil {
		log.Fatalf("error starting server: %v", err)
	}
}
