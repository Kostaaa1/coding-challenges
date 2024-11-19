package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"time"
)

type Config struct {
	Addr string
}

type Proxy struct {
	Addr string
}

func (s *Server) Next() {
	if s.ProxyActive == nil {
		s.ProxyActive = &s.ProxyMap[0]
		return
	}

	var id int
	for i := range s.ProxyMap {
		if s.ProxyMap[i].Addr == s.ProxyActive.Addr {
			id = i
			break
		}
	}

	id = (id + 1) % len(s.ProxyMap)
	s.ProxyActive = &s.ProxyMap[id]
}

type Server struct {
	Config
	ln          net.Listener
	ProxyMap    []Proxy
	ProxyActive *Proxy
}

func NewServer(cfg Config) *Server {
	if cfg.Addr == "" {
		cfg.Addr = ":8000"
	}
	return &Server{
		Config: cfg,
		ProxyMap: []Proxy{
			{Addr: "http://localhost:8001"},
			{Addr: "http://localhost:8002"},
			{Addr: "http://localhost:8003"},
		},
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	s.ln = ln
	slog.Info("server started on port", "address", s.Config.Addr)
	return s.acceptLoop()
}

func (s *Server) acceptLoop() error {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			slog.Error("accept error", "err", err)
			continue
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
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

	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest("GET", s.ProxyActive.Addr, bytes.NewReader(data))
	if err != nil {
		log.Println("Failed to create request:", err)
		return
	}

	resp, err := client.Do(req)
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
	cfg := Config{}
	s := NewServer(cfg)
	if err := s.Start(); err != nil {
		log.Fatalf("error starting server: %v", err)
	}
}
