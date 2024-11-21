package main

import (
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
)

func writeError(conn net.Conn, status int, message string, err error) {
	slog.Error(message, "err", err)
	responseBody := message + "\n"
	response := fmt.Sprintf(
		"HTTP/1.1 %d %s\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s\n", status, http.StatusText(status), len(responseBody), responseBody,
	)
	_, err = conn.Write([]byte(response))
	if err != nil {
		log.Printf("Failed to write error response: %v", err)
	}
}

func writeResponse(conn net.Conn, resp *http.Response, data []byte) {
	conn.Write([]byte(fmt.Sprintf("HTTP/1.1 %d %s\r\n", resp.StatusCode, http.StatusText(resp.StatusCode))))
	for key, values := range resp.Header {
		for _, value := range values {
			conn.Write([]byte(fmt.Sprintf("%s: %s\r\n", key, value)))
		}
	}
	conn.Write([]byte("\r\n"))
	conn.Write(data)
	conn.Write([]byte("\r\n"))
}

func getActiveProxy(proxies []*proxy) *proxy {
	for _, p := range proxies {
		if p.isActive {
			return p
		}
	}
	return nil
}
