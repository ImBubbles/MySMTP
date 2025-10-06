package server

import (
	"MySMTP/smtp"
	"fmt"
	"net"
	"os"
)

type Server struct {
	listener net.Listener
	port     uint16
	active   bool
}

func NewServer(address string, port uint16) *Server {
	server, err := net.Listen("tcp", fmt.Sprintf("%s:%d", address, port))
	if err != nil {
		os.Exit(1)
	}
	defer server.Close()
	return &Server{server, port, false}
}

func Listen(server *Server) {
	if server.active {
		return
	}
	for {
		conn, err := server.listener.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	go smtp.NewServerConn(conn)
}
