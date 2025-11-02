package server

import (
	"MySMTP/config"
	"MySMTP/smtp"
	"fmt"
	"net"
	"os"
)

type Server struct {
	listener net.Listener
	port     uint16
	active   bool
	config   *config.Config
}

func NewServer(address string, port uint16) *Server {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", address, port))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server on %s:%d: %v\n", address, port, err)
		os.Exit(1)
	}
	return &Server{listener, port, false, nil}
}

func Listen(srv *Server, cfg *config.Config) {
	srv.config = cfg
	if srv.active {
		return
	}
	srv.active = true
	for {
		conn, err := srv.listener.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn, cfg)
	}
}

func handleConnection(conn net.Conn, cfg *config.Config) {
	defer conn.Close()
	connPtr := &conn
	// Use default handlers if set, otherwise create new ones
	handlers := GetDefaultHandlers()
	smtp.NewServerConnWithHandlers(connPtr, cfg, handlers)
}

// SetHandlers sets handlers for all new connections
// Note: This sets default handlers. For per-connection handlers, use NewServerConnWithHandlers
var defaultHandlers *smtp.Handlers

// SetDefaultHandlers sets the default handlers for all new connections
func SetDefaultHandlers(handlers *smtp.Handlers) {
	defaultHandlers = handlers
}

// GetDefaultHandlers returns the default handlers
func GetDefaultHandlers() *smtp.Handlers {
	if defaultHandlers == nil {
		return smtp.NewHandlers()
	}
	return defaultHandlers
}
