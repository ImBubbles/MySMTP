package smtp

import (
	"bufio"
	"net"
)
import "MySMTP/smtp/protocol"

// ServerConn handle client connections to the SMTP server
type ServerConn struct {
	client net.Conn
	state  protocol.SMTPStates
	reader *bufio.Reader
}

func NewServerConn(conn net.Conn) {
	serverConn := &ServerConn{client: conn, state: protocol.STATE_EHLO, reader: bufio.NewReader(conn)}
	serverConn.handle()
}

func (s *ServerConn) handle() {
	s.write(protocol.PREPARED_S_ACCEPTANCE)
}

func write(s *ServerConn, str string) error {
	_, err := s.client.Write([]byte(str))
	return err
}

func (s *ServerConn) write(str string) {
	err := write(s, str)
	if err != nil {
		panic(err)
	}
}

func (s *ServerConn) read() string {
	for {
		message, err := s.reader.ReadString('\n')
		if err != nil {
			panic(err)
		}
		return message
	}
}
