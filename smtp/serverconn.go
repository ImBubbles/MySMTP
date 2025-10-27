package smtp

import (
	"MySMTP/mail"
	"MySMTP/util/conn"
	smtputil "MySMTP/util/smtp"
	stringutil "MySMTP/util/string"
	"bufio"
	"fmt"
	"net"
	"strings"
)
import "MySMTP/smtp/protocol"

// ServerConn handle client connections to the SMTP server
type ServerConn struct {
	client     *net.Conn
	state      protocol.SMTPStates
	reader     *bufio.Reader
	relay      bool
	requireTLS bool
	size       uint64
	body       protocol.SMTPBody
	mail       mail.Mail
}

func NewServerConn(conn *net.Conn, relay bool, requireTLS bool) *ServerConn {
	serverConn := &ServerConn{
		client:     conn,
		state:      protocol.STATE_EHLO,
		reader:     bufio.NewReader(*conn),
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
			s.handleMailFrom(line)
		case string(protocol.COMMAND_RCPT):
			s.handleRctpTo(line)
		case string(protocol.COMMAND_DATA):
			s.handleData(line)
		case string(protocol.COMMAND_STARTTLS):
			s.handleStartTLS(line)
		default:
			s.write(protocol.PREPARED_S_BAD_COMMAND)
		}
	}
}

func (s *ServerConn) write(str string) {
	err := conn.Write(s.client, str)
	if err != nil {
		panic(err)
	}
}

func (s *ServerConn) read() string {
	return conn.Read(s.reader)
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

	if !strings.HasPrefix(line, "MAIL FROM:") {
		s.write(protocol.PREPARED_S_BAD_COMMAND)
		return
	}
	remainder := strings.TrimSpace(line[strings.Index(line, ": ")+1:])

	if remainder == "" {
		s.write(protocol.PREPARED_S_BAD_SYNTAX)
		return
	}

	// yay address
	addEnd := strings.Index(remainder, ">")
	address := smtputil.CleanEmail(remainder[:addEnd+1])
	s.mail.SetFrom(address)

	// even has params?
	if len(remainder) == addEnd {
		s.write(protocol.PREPARED_S_ACKNOWLEDGE)
		return
	}

	remainder = strings.ToUpper(remainder)
	// get paramters now
	params := strings.Split(strings.TrimSpace(remainder[addEnd+1:]), " ")

	for _, param := range params {
		key := param
		eqIndex := strings.Index(param, "=")
		var value string = ""
		if eqIndex == -1 {
			key = param[:eqIndex]
		} else {
			value = param[eqIndex+1:]
		}
		if protocol.SMTP_VALID_FLAGS.Contains(protocol.SMTPFromFlags(key)) { // is a valid flag
			s.mail.AppendFlag(*mail.NewFlag(key, value))
		}
	}

	s.write(protocol.PREPARED_S_ACKNOWLEDGE)
	s.state = protocol.STATE_RCPT_TO

}

func (s *ServerConn) handleRctpTo(line string) { // todo handle NOTIFY=SUCCESS,FAILURE,DENY
	if s.state != protocol.STATE_RCPT_TO {
		s.write(protocol.PREPARED_S_BAD_SEQUENCE)
		return
	}
	// Expected format is "RCPT TO:<address>"
	if !strings.HasPrefix(line, "RCPT TO:") {
		s.write(protocol.PREPARED_S_BAD_COMMAND)
		return
	}
	remainder := strings.TrimSpace(line[strings.Index(line, ": ")+1:])
	if remainder == "" {
		s.write(protocol.PREPARED_S_BAD_SYNTAX)
		return
	}
	// yay address
	addEnd := strings.Index(remainder, ">")
	address := smtputil.CleanEmail(remainder[:addEnd+1])
	s.mail.AppendTo(address)
	s.write(protocol.PREPARED_S_ACKNOWLEDGE)
}

func (s *ServerConn) handleData(line string) {
	if s.state == protocol.STATE_RCPT_TO {
		s.state = protocol.STATE_DATA
	}
	if s.state != protocol.STATE_DATA {
		s.write(protocol.PREPARED_S_BAD_SEQUENCE)
		return
	}
	s.write(protocol.PREPARED_S_START_DATA)
	raw := *stringutil.NewStringBuilder()

	// headers
	// todo finish

}

func (s *ServerConn) handleStartTLS(line string) {
	// TODO
}
