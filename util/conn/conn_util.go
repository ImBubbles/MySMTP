package conn

import (
	"bufio"
	"net"
)

func Write(conn *net.Conn, str string) error {
	_, err := (*conn).Write([]byte(str))
	return err
}

func NewReader(conn *net.Conn) *bufio.Reader {
	return bufio.NewReader(*conn)
}

func Read(r *bufio.Reader) string {
	for {
		message, err := (*r).ReadString('\n')
		if err != nil {
			panic(err)
		}
		return message
	}
}
