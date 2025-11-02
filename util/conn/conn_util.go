package conn

import (
	"bufio"
	"io"
	"net"
)

func Write(conn *net.Conn, str string) error {
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
	// Use ReadLine() for more robust SMTP line reading
	// ReadLine() handles both \n and \r\n and returns the line without the delimiter
	line, isPrefix, err := r.ReadLine()
	if err != nil {
		if err == io.EOF {
			// EOF means connection closed, return empty string
			return ""
		}
		// Check if it's a timeout error
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			// Timeout error - return empty string so caller can handle it
			return ""
		}
		// Other errors - panic
		panic(err)
	}

	// If the line is too long (isPrefix is true), we need to read more
	// SMTP lines shouldn't exceed 998 bytes per RFC 5321, but we'll handle longer lines
	var fullLine []byte = line
	for isPrefix {
		line, isPrefix, err = r.ReadLine()
		if err != nil {
			if err == io.EOF {
				// EOF during reading - return what we have
				break
			}
			// Other errors - panic
			panic(err)
		}
		fullLine = append(fullLine, line...)
	}

	// Convert to string and append \r\n (SMTP uses CRLF but ReadLine removes it)
	// We add \r\n here so callers get the proper SMTP line ending
	return string(fullLine) + "\r\n"
}
