package main

import (
	"github.com/ImBubbles/MySMTP/config"
	"github.com/ImBubbles/MySMTP/server"
	"fmt"
	"os"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Print configuration
	cfg.PrintConfig()

	// Create and start server
	socket := server.NewServer(cfg.ServerAddress, cfg.ServerPort)
	server.Listen(socket, cfg)
}
