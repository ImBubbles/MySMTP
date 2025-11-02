package conn

import (
	"bufio"
	"io"
	"net"
	"strings"
)

func Write(conn *net.Conn, str string) error {
	// Ensure SMTP protocol compliance: all lines must end with \r\n
	// Add \r\n if not already present
	if len(str) < 2 || str[len(str)-2:] != "\r\n" {
		str += "\r\n"
	}

	data := []byte(str)
	// Write all bytes - Write() may write only part of the data
	// Loop until all bytes are written or an error occurs
	for len(data) > 0 {
		n, err := (*conn).Write(data)
		if err != nil {
			return err
		}
		// Move forward in the data slice
		data = data[n:]
	}
	return nil
}

func NewReader(conn *net.Conn) *bufio.Reader {
	return bufio.NewReader(*conn)
}

func Read(r *bufio.Reader) string {
	// ReadString reads until \n (works with both \n and \r\n)
	line, err := r.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return ""
		}
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return ""
		}
		return ""
	}

	// Trim any \r\n or \n, then add back \r\n for consistency
	line = strings.TrimRight(line, "\r\n")
	return line + "\r\n"
}
