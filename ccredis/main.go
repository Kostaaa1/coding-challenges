package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	redis "github.com/Kostaaa1/redis-clone/internal/resp"
)

func main() {
	port := flag.Int("port", 6380, "redis server port")
	flag.Parse()

	addr := fmt.Sprintf("127.0.0.1:%d", *port)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Redis server started: %s\n", addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			if err == io.EOF {
				fmt.Println("client closed the connection")
				continue
			}
			log.Fatal(err)
		}
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	for {
		r := redis.NewReader(conn)

		v, err := r.Read()
		if err != nil {
			if err == io.EOF {
				fmt.Println("client closed the connection")
				return
			}
			log.Fatal(err)
		}

		if v.Type != "array" {
			fmt.Println("invalid request, expected array")
			continue
		}

		if len(v.Array) == 0 {
			fmt.Println("invalid request, expected array length > 0")
			continue
		}

		cmd := strings.ToUpper(v.Array[0].Bulk)
		args := v.Array

		w := redis.NewWriter(conn)

		handler, ok := redis.Handlers[cmd]
		if !ok {
			w.Write(redis.UnknownCmd(cmd, args[1:]))
			continue
		}

		// sending all args, middleware func extracts the command from other arguments (command included)
		w.Write(handler(args))
	}
}
