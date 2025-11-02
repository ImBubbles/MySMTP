package conn

import (
	"bufio"
	"io"
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
	message, err := r.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			// EOF means connection closed, return empty string or handle gracefully
			return ""
		}
		panic(err)
	}
	return message
}
