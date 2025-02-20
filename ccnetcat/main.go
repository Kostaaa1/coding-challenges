package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
)

type config struct {
	l    bool
	port string
}

func main() {
	var cfg config
	flag.BoolVar(&cfg.l, "l", true, "TCP")
	flag.StringVar(&cfg.port, "p", "8888", "nc listen port")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	logger.Debug("Debug message")
	logger.Info("Info message")
	logger.Warn("Warning message")
	logger.Error("Error message")

	l, err := net.Listen("tcp", ":"+cfg.port)
	if err != nil {
		panic(fmt.Errorf("failed to listen on port: %s", cfg.port))
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			panic(err)
		}
		logger.Info("connection established", "", conn.LocalAddr().Network())
	}
}
