package smtp

import (
	"MySMTP/mail"
	stringutil "MySMTP/util/string"
	"bufio"
	"fmt"
	"net"
	"strings"
)
import "MySMTP/smtp/protocol"

// ServerConn handle client connections to the SMTP server
type ServerConn struct {
	client     net.Conn
	state      protocol.SMTPStates
	reader     *bufio.Reader
	relay      bool
	requireTLS bool
	size       uint64
	body       protocol.SMTPBody
	mail       mail.Mail
}

func NewServerConn(conn net.Conn, relay bool, requireTLS bool) *ServerConn {
	serverConn := &ServerConn{
		client:     conn,
		state:      protocol.STATE_EHLO,
		reader:     bufio.NewReader(conn),
		relay:      relay,
		requireTLS: requireTLS,
		size:       0,
		body:       protocol.BODY_8BITMIME}
	serverConn.handle()
	return serverConn
}

func (s *ServerConn) handle() {
	s.write(protocol.PREPARED_S_ACCEPTANCE)
	for {
		line := s.read()
		command := strings.ToUpper(line)
		command = stringutil.FirstWord(command)

		// Ignore http
		if strings.Contains(command, "HTTP") {
			panic(1)
		}
		switch command {
		case string(protocol.COMMAND_EHLO), string(protocol.COMMAND_HELO):
			s.handleEHLO(line)
		case string(protocol.COMMAND_MAIL):
			return // todo
		case string(protocol.COMMAND_STARTTLS):
			s.handleStartTLS(line)
		default:
			s.write(protocol.PREPARED_S_BAD_COMMAND)
		}
	}
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

func (s *ServerConn) handleEHLO(line string) {
	if s.state != protocol.STATE_EHLO {
		return
	}
	// Extract domain
	parts := strings.Split(line, " ")
	var domain string
	if len(parts) > 1 {
		domain = parts[1]
	} else {
		s.write(protocol.PREPARED_S_BAD_SYNTAX)
		return
	}

	// Respond with success code and supported extensions
	s.write(fmt.Sprintf(protocol.PREPARED_S_ADVERTISE_HELLO, domain))
	//s.write(fmt.Sprintf(protocol.PREPARED_S_ADVERTISE_SIZE, parts[0]))
	if s.relay {
		s.write(protocol.PREPARED_S_ADVERTISE_AUTH)
	}
	//s.write(protocol.PREPARED_S_ADVERTISE_PIPELINING)
	//s.write(protocol.PREPARED_S_ADVERTISE_HELP)
	s.write(protocol.PREPARED_S_ADVERTISE_8BITMIME)
	//s.write(protocol.PREPARED_S_ADVERTISE_TLS)
	s.write(protocol.PREPARED_S_ACKNOWLEDGE)

	if s.relay {
		s.state = protocol.STATE_AUTH
		return
	}
	s.state = protocol.STATE_MAIL_FROM

}

// Expecting MAIL FROM:<address>
// OR something like MAIL FROM:<user@example.com> [SIZE=12345] [BODY=8BITMIME] [SMTPUTF8]

func (s *ServerConn) handleMailFrom(line string) {
	if s.state != protocol.STATE_MAIL_FROM {
		s.write(protocol.PREPARED_S_BAD_SEQUENCE)
		return
	}
}

func (s *ServerConn) handleStartTLS(line string) {
	// TODO
}
