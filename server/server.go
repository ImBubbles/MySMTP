package server

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"

	"github.com/ImBubbles/MySMTP/config"
	"github.com/ImBubbles/MySMTP/smtp"
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
			// Log error but don't exit - continue accepting
			// This handles temporary network errors gracefully
			fmt.Fprintf(os.Stderr, "Failed to accept connection: %v\n", err)
			continue
		}
		// Each connection is handled in its own goroutine
		// The connection is closed by defer in handleConnection
		go handleConnection(conn, cfg)
	}
}

func handleConnection(conn net.Conn, cfg *config.Config) {
	// Ensure connection is closed when done
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()

	// Use default handlers if set, otherwise create new ones
	handlers := GetDefaultHandlers()

	// Load TLS certificate if TLS is enabled
	var tlsConfig *tls.Config
	if cfg.TLSEnabled {
		cert, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load TLS certificate (%s, %s): %v\n", cfg.TLSCertFile, cfg.TLSKeyFile, err)
			// Continue without TLS
		} else {
			tlsConfig = &tls.Config{
				Certificates: []tls.Certificate{cert},
				MinVersion:   tls.VersionTLS12, // Require TLS 1.2 or higher
			}
		}
	}

	// Pass the connection directly - no pointer indirection needed
	// The connection stays alive in this goroutine's scope
	smtp.NewServerConnWithHandlers(conn, cfg, handlers, tlsConfig)
	// handle() is called inside NewServerConnWithHandlers and blocks until connection closes
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
