package smtp

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strings"

	"github.com/ImBubbles/MySMTP/config"
	"github.com/ImBubbles/MySMTP/mail"
	"github.com/ImBubbles/MySMTP/smtp/protocol"
	"github.com/ImBubbles/MySMTP/util/conn"
	smtputil "github.com/ImBubbles/MySMTP/util/smtp"
	stringutil "github.com/ImBubbles/MySMTP/util/string"
	"github.com/ImBubbles/MySMTP/util/verify"
)

// ServerConn handle client connections to the SMTP server
type ServerConn struct {
	client         *net.Conn
	state          protocol.SMTPStates
	reader         *bufio.Reader
	relay          bool
	requireTLS     bool
	tlsConfig      *tls.Config
	size           uint64
	body           protocol.SMTPBody
	mail           mail.Mail
	config         *config.Config
	senderVerifier *verify.EmailVerifier
	handlers       *Handlers
}

// NewServerConn creates a new server connection
func NewServerConn(conn *net.Conn, cfg *config.Config) *ServerConn {
	return NewServerConnWithHandlers(conn, cfg, NewHandlers())
}

// NewServerConnWithHandlers creates a new server connection with custom handlers
func NewServerConnWithHandlers(conn *net.Conn, cfg *config.Config, handlers *Handlers) *ServerConn {
	// Create sender verifier with default settings
	verifier := verify.NewEmailVerifier()
	verifier.SetCheckFormat(true)
	verifier.SetCheckMX(false) // Disable MX checking by default for performance

	// Ensure handlers is not nil
	if handlers == nil {
		handlers = NewHandlers()
	}

	serverConn := &ServerConn{
		client:         conn,
		state:          protocol.STATE_EHLO,
		reader:         bufio.NewReader(*conn),
		relay:          cfg.Relay,
		requireTLS:     cfg.RequireTLS,
		tlsConfig:      nil,
		size:           0,
		body:           protocol.BODY_8BITMIME,
		config:         cfg,
		senderVerifier: verifier,
		handlers:       handlers}
	serverConn.handle()
	return serverConn
}

// SetSenderVerifier sets a custom sender verifier
func (s *ServerConn) SetSenderVerifier(verifier *verify.EmailVerifier) {
	s.senderVerifier = verifier
}

// SetHandlers sets the handlers for this connection
func (s *ServerConn) SetHandlers(handlers *Handlers) {
	if handlers != nil {
		s.handlers = handlers
	}
}

// SetTLSConfig sets the TLS configuration for STARTTLS
func (s *ServerConn) SetTLSConfig(config *tls.Config) {
	s.tlsConfig = config
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
		case string(protocol.COMMAND_QUIT):
			s.handleQuit(line)
			return // Connection will close
		case string(protocol.COMMAND_RSET):
			s.handleRset(line)
		default:
			s.write(protocol.PREPARED_S_BAD_COMMAND)
		}
	}
}

func (s *ServerConn) write(str string) {
	// Print transmission to client (trim \r\n for cleaner output)
	output := strings.TrimRight(str, "\r\n")
	fmt.Printf("SERVER -> CLIENT: %s\n", output)
	err := conn.Write(s.client, str)
	if err != nil {
		panic(err)
	}
}

func (s *ServerConn) read() string {
	line := conn.Read(s.reader)
	// Print transmission from client (trim \r\n for cleaner output)
	output := strings.TrimRight(line, "\r\n")
	fmt.Printf("SERVER <- CLIENT: %s\n", output)
	return line
}

func (s *ServerConn) handleEHLO(line string) {
	if s.state != protocol.STATE_EHLO {
		return
	}
	// Extract domain (for validation)
	parts := strings.Split(line, " ")
	if len(parts) < 2 {
		s.write(protocol.PREPARED_S_BAD_SYNTAX)
		return
	}

	// Respond with success code and supported extensions
	// Use configured server domain instead of client's domain
	serverDomain := s.config.ServerDomain
	s.write(fmt.Sprintf(protocol.PREPARED_S_ADVERTISE_HELLO, serverDomain))
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
	// Find the colon (should be at ": " or just ":")
	colonIndex := strings.Index(line, ":")
	if colonIndex == -1 {
		s.write(protocol.PREPARED_S_BAD_SYNTAX)
		return
	}
	remainder := strings.TrimSpace(line[colonIndex+1:])

	if remainder == "" {
		s.write(protocol.PREPARED_S_BAD_SYNTAX)
		return
	}

	// Extract address (should be in <address> format)
	addEnd := strings.Index(remainder, ">")
	if addEnd == -1 {
		// No closing bracket found
		s.write(protocol.PREPARED_S_BAD_SYNTAX)
		return
	}
	address := smtputil.CleanEmail(remainder[:addEnd+1])
	if address == "" {
		// Invalid address
		s.write(protocol.PREPARED_S_BAD_SYNTAX)
		return
	}

	// Verify sender email address
	if s.senderVerifier != nil && !s.senderVerifier.VerifyEmail(address) {
		s.write(protocol.PREPARED_S_BAD_SYNTAX)
		return
	}

	s.mail.SetFrom(address)

	// Check if there are parameters after the address
	if len(remainder) <= addEnd+1 {
		s.write(protocol.PREPARED_S_ACKNOWLEDGE)
		return
	}

	remainder = strings.ToUpper(remainder)
	// get paramters now
	params := strings.Split(strings.TrimSpace(remainder[addEnd+1:]), " ")

	for _, param := range params {
		param = strings.TrimSpace(param)
		if param == "" {
			continue
		}
		eqIndex := strings.Index(param, "=")
		var key string
		var value string = ""
		if eqIndex == -1 {
			// No value, just key
			key = param
		} else {
			// Has value
			key = strings.TrimSpace(param[:eqIndex])
			value = strings.TrimSpace(param[eqIndex+1:])
		}
		if key != "" && protocol.SMTP_VALID_FLAGS.Contains(protocol.SMTPFromFlags(key)) {
			// Valid flag
			flag := mail.NewFlag(key, value)
			s.mail.AppendFlag(mail.FromFlag(*flag))
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
	// Find the colon (should be at ": " or just ":")
	colonIndex := strings.Index(line, ":")
	if colonIndex == -1 {
		s.write(protocol.PREPARED_S_BAD_SYNTAX)
		return
	}
	remainder := strings.TrimSpace(line[colonIndex+1:])
	if remainder == "" {
		s.write(protocol.PREPARED_S_BAD_SYNTAX)
		return
	}
	// Extract address (should be in <address> format)
	addEnd := strings.Index(remainder, ">")
	if addEnd == -1 {
		// No closing bracket found
		s.write(protocol.PREPARED_S_BAD_SYNTAX)
		return
	}
	address := smtputil.CleanEmail(remainder[:addEnd+1])
	if address == "" {
		// Invalid address
		s.write(protocol.PREPARED_S_BAD_SYNTAX)
		return
	}

	// Check if email exists using handler (default returns false)
	if s.handlers != nil && s.handlers.EmailExistsChecker != nil {
		if !s.handlers.EmailExistsChecker(address) {
			// Email does not exist
			s.write(protocol.PREPARED_S_BAD_SYNTAX)
			return
		}
	}

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

	// Read headers until blank line
	headers := *stringutil.NewStringBuilder()
	body := *stringutil.NewStringBuilder()
	inHeaders := true
	bodyStarted := false
	currentHeaderName := ""
	currentHeaderValue := *stringutil.NewStringBuilder()

	for {
		rawLine := s.read()
		trimmedLine := strings.TrimSpace(rawLine)

		// Check for terminator first (line containing only ".")
		// The terminator is a line that when trimmed is just "."
		isTerminator := trimmedLine == "."
		if isTerminator {
			// Process any pending header before breaking
			if inHeaders && currentHeaderName != "" {
				headerValue := strings.TrimSpace(currentHeaderValue.AsString())
				s.processHeader(currentHeaderName, headerValue)
			}
			break
		}

		// Handle SMTP transparency: lines starting with "." need the leading "." removed
		// The terminator was already handled above, so we can safely remove leading dots
		line := rawLine
		if len(line) > 0 && line[0] == '.' {
			// This is a transparent line, remove the leading "."
			line = line[1:]
			trimmedLine = strings.TrimSpace(line)
		}

		// Check if we've reached the end of headers (blank line)
		if inHeaders && trimmedLine == "" {
			// Process any pending header before moving to body
			if currentHeaderName != "" {
				headerValue := strings.TrimSpace(currentHeaderValue.AsString())
				s.processHeader(currentHeaderName, headerValue)
				currentHeaderName = ""
				currentHeaderValue = *stringutil.NewStringBuilder()
			}
			inHeaders = false
			bodyStarted = true
			headers.Append(line)
			continue
		}

		if inHeaders {
			// Parse header line (format: "Header-Name: value")
			colonIndex := strings.Index(line, ":")
			if colonIndex > 0 {
				// Process previous header if any
				if currentHeaderName != "" {
					headerValue := strings.TrimSpace(currentHeaderValue.AsString())
					s.processHeader(currentHeaderName, headerValue)
				}

				// Start new header
				currentHeaderName = strings.TrimSpace(line[:colonIndex])
				headerValuePart := strings.TrimSpace(line[colonIndex+1:])
				currentHeaderValue = *stringutil.NewStringBuilder()
				if headerValuePart != "" {
					currentHeaderValue.Append(headerValuePart)
				}
				headers.Append(line)
			} else {
				// Continuation of previous header (starts with space or tab)
				if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
					if currentHeaderName != "" {
						// Append continuation to current header value
						currentHeaderValue.Append(" " + strings.TrimSpace(line))
					}
					headers.Append(line)
				}
			}
		} else {
			// We're in the body
			body.Append(line)
		}
	}

	// Store the complete email data (headers + body)
	fullData := headers.AsString()
	if bodyStarted {
		fullData += body.AsString()
	}
	s.mail.SetData(fullData)

	// Process mail using handler if set (override handling of finished email)
	if s.handlers != nil && s.handlers.MailHandler != nil {
		if err := s.handlers.MailHandler(&s.mail); err != nil {
			// Handler rejected the mail
			s.write(protocol.PREPARED_S_TRANSACTION_FAILED)
			return
		}
	}

	// Acknowledge successful data reception
	s.write(protocol.PREPARED_S_ACKNOWLEDGE)

	// Reset state for next transaction
	s.state = protocol.STATE_EHLO
}

// processHeader parses and stores header information
func (s *ServerConn) processHeader(headerName string, headerValue string) {
	headerNameUpper := strings.ToUpper(headerName)

	// Parse common headers
	switch headerNameUpper {
	case "SUBJECT":
		s.mail.SetSubject(headerValue)
	case "CC":
		// Parse CC addresses (can be comma-separated)
		addresses := mail.ParseNamedAddress(headerName + ": " + headerValue)
		for _, addr := range *addresses {
			s.mail.AppendCC(addr.GetAddress())
		}
	case "BCC":
		// Parse BCC addresses (can be comma-separated)
		addresses := mail.ParseNamedAddress(headerName + ": " + headerValue)
		for _, addr := range *addresses {
			s.mail.AppendBCC(addr.GetAddress())
		}
	}
}

func (s *ServerConn) handleStartTLS(line string) {
	// STARTTLS can only be used in EHLO state (before authentication or mail transaction)
	if s.state != protocol.STATE_EHLO {
		s.write(protocol.PREPARED_S_BAD_SEQUENCE)
		return
	}

	// Check if TLS config is available
	if s.tlsConfig == nil {
		s.write(protocol.PREPARED_S_BAD_COMMAND)
		return
	}

	// Send "220 Ready to start TLS"
	s.write(protocol.PREPARED_S_STARTTLS_READY)

	// Perform TLS handshake
	tlsConn := tls.Server(*s.client, s.tlsConfig)
	err := tlsConn.Handshake()
	if err != nil {
		// TLS handshake failed
		return
	}

	// Update connection and reader with TLS-wrapped connection
	// Convert *tls.Conn to *net.Conn by storing the interface value in a heap-allocated location
	// Allocate space for the interface value by using a temporary struct field approach
	netConn := net.Conn(tlsConn)
	// Store in a new heap-allocated variable
	heapConn := &netConn
	s.client = heapConn
	s.reader = bufio.NewReader(tlsConn)

	// Reset state to EHLO - client must send EHLO again after STARTTLS
	s.state = protocol.STATE_EHLO
}

func (s *ServerConn) handleQuit(line string) {
	// Send "221 Bye" response
	s.write(protocol.PREPARED_S_BYE)
	// Connection will be closed by caller
}

func (s *ServerConn) handleRset(line string) {
	// Reset the mail transaction
	s.mail = mail.Mail{}
	s.state = protocol.STATE_EHLO

	// Send acknowledgment
	s.write(protocol.PREPARED_S_ACKNOWLEDGE)
}
