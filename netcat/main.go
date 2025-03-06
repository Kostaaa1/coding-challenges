package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

type UDPServer struct {
	pc          net.PacketConn
	allowedAddr net.Addr
}

func NewUDPServer(addr string) *UDPServer {
	pc, err := net.ListenPacket("udp", addr)
	if err != nil {
		log.Fatal(err)
	}
	return &UDPServer{
		pc: pc,
	}
}

func (s *UDPServer) stop() {
	s.pc.Close()
}

func (s *UDPServer) serve(ctx context.Context) {
	defer s.stop()

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
				msg := scanner.Text()
				_, err := s.pc.WriteTo([]byte(msg+"\n"), s.allowedAddr)
				if err != nil {
					fmt.Println("failed to write message to connection", err)
				}
			}
		}
	}()

	buf := make([]byte, 1024)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, addr, err := s.pc.ReadFrom(buf)
			if err != nil {
				if err == io.EOF {
					fmt.Println("connection closed by client")
					s.allowedAddr = nil
				} else {
					fmt.Println("error while reading from UDP conn")
				}
				return
			}

			if s.allowedAddr == nil {
				s.allowedAddr = addr
			}

			if addr.String() != s.allowedAddr.String() {
				fmt.Println("Ignoring message from:", addr)
				continue
			}

			msg := strings.TrimSpace(string(buf[:n]))
			fmt.Println(msg)
		}
	}
}

// func (s *UDPServer) serve(ctx context.Context) {
// 	defer s.stop()

// 	var lastAddr net.Addr

// 	go func() {
// 		scanner := bufio.NewScanner(os.Stdin)
// 		for scanner.Scan() {
// 			select {
// 			case <-ctx.Done():
// 				return

// 			default:
// 				fmt.Println("writing udp message to: ", lastAddr)
// 				msg := scanner.Text()
// 				_, err := s.pc.WriteTo([]byte(msg+"\n"), lastAddr)
// 				if err != nil {
// 					fmt.Println("failed to write message to connection", err)
// 				}
// 			}
// 		}
// 	}()

// 	for {
// 		buf := make([]byte, 1024)
// 		select {
// 		case <-ctx.Done():
// 			s.stop()
// 			return
// 		default:
// 			n, addr, err := s.pc.ReadFrom(buf)
// 			if err != nil {
// 				fmt.Println("error while reading from UDP conn")
// 				continue
// 			}

// 			lastAddr = addr

// 			msg := strings.TrimSpace(string(buf[:n]))
// 			fmt.Println(msg)

// 			_, err = s.pc.WriteTo([]byte(msg+"\n"), addr)
// 			if err != nil {
// 				fmt.Println("failed to write back to UDP addr: ", addr)
// 			}
// 		}
// 	}
// }

type TCPServer struct {
	l net.Listener
}

func NewTCPServer(addr string) *TCPServer {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	return &TCPServer{
		l: l,
	}
}

func (s *TCPServer) stop() {
	s.l.Close()
}

func (s *TCPServer) serve(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("context cancelled, shutting down the listener...")
			s.stop()
			return
		default:
			conn, err := s.l.Accept()
			if err != nil {
				fmt.Println("error while accepting new conn")
				return
			}
			defer s.stop()
			go s.handleConn(conn, ctx)
		}
	}
}

func (s *TCPServer) handleConn(conn net.Conn, ctx context.Context) {
	defer conn.Close()

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return

			default:
				msg := scanner.Text()
				_, err := conn.Write([]byte(msg + "\n"))
				if err != nil {
					fmt.Println("failed to write message to connection", err)
					return
				}
			}
		}
	}()

	buf := make([]byte, 1024)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := conn.Read(buf)
			if err != nil {
				if err == io.EOF {
					fmt.Println("client closed the connection")
				} else {
					fmt.Println("error reading conn to buff", err)
				}
				return
			}
			msg := strings.TrimSpace(string(buf[:n]))
			fmt.Println(msg)
		}
	}
}

func main() {
	udp := flag.Bool("u", false, "")
	listen := flag.Bool("l", false, "")
	// verbose := flag.Bool("v", false, "")
	checker := flag.Bool("z", false, "")
	port := flag.String("port", "8888", "")
	flag.Parse()

	if !strings.HasPrefix(*port, ":") {
		*port = ":" + *port
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if *checker {
		conn, err := net.Dial("tcp", *port)
		if err != nil {
			fmt.Println("error occurred", err)
		}

		buf := make([]byte, 1024)
		_, err = conn.Read(buf)
		if err != nil {
			fmt.Println("error occurred when reading", err)
		}

		return
	}

	if *listen {
		if *udp {
			fmt.Println("UDP")
			srv := NewUDPServer(*port)
			srv.serve(ctx)
		} else {
			fmt.Println("TCP")
			srv := NewTCPServer(*port)
			srv.serve(ctx)
		}
	} else {

	}
}
