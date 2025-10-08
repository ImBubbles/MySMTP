package main

import "MySMTP/server"

func main() {
	socket := server.NewServer("0.0.0.0", 2525)
	server.Listen(socket)

}
