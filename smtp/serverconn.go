package smtp

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/ImBubbles/MySMTP/config"
	"github.com/ImBubbles/MySMTP/mail"
	"github.com/ImBubbles/MySMTP/smtp/protocol"
	smtputil "github.com/ImBubbles/MySMTP/util/smtp"
	stringutil "github.com/ImBubbles/MySMTP/util/string"
	"github.com/ImBubbles/MySMTP/util/verify"
)

// ServerConn handle client connections to the SMTP server
type ServerConn struct {
	client         net.Conn // Changed from *net.Conn to net.Conn - direct reference
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
func NewServerConn(conn net.Conn, cfg *config.Config) *ServerConn {
	return NewServerConnWithHandlers(conn, cfg, NewHandlers(), nil)
}

// NewServerConnWithHandlers creates a new server connection with custom handlers and optional TLS config
// If tlsConfig is provided, STARTTLS will be advertised in EHLO and clients can upgrade the connection
// Pass nil for tlsConfig if you don't want to enable STARTTLS
func NewServerConnWithHandlers(conn net.Conn, cfg *config.Config, handlers *Handlers, tlsConfig *tls.Config) *ServerConn {
	// Create sender verifier with default settings
	verifier := verify.NewEmailVerifier()
	verifier.SetCheckFormat(true)
	verifier.SetCheckMX(false) // Disable MX checking by default for performance

	// Ensure handlers is not nil
	if handlers == nil {
		handlers = NewHandlers()
	}

	// Set initial timeouts to prevent hanging
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}

	serverConn := &ServerConn{
		client:         conn, // Direct assignment, no pointer
		state:          protocol.STATE_EHLO,
		reader:         bufio.NewReader(conn),
		relay:          cfg.Relay,
		requireTLS:     cfg.RequireTLS,
		tlsConfig:      tlsConfig, // Use provided TLS config
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
	if !s.write(protocol.PREPARED_S_ACCEPTANCE) {
		return // Connection broken
	}
	for {
		line := s.read()
		// Check if connection was closed (empty read means connection closed)
		if line == "" {
			fmt.Printf("SERVER: Connection closed by client\n")
			return
		}

		// Trim whitespace and CRLF from line before parsing command
		// This ensures commands like "QUIT\r\n" are properly matched
		line = strings.TrimSpace(line)
		if line == "" {
			// Empty line after trimming - continue to next iteration
			continue
		}

		// Convert to uppercase and extract first word (the command)
		// TrimSpace ensures no leading/trailing whitespace in command
		upperLine := strings.ToUpper(line)
		command := strings.TrimSpace(stringutil.FirstWord(upperLine))

		// Debug: Print the command being matched (for troubleshooting)
		fmt.Printf("DEBUG: Parsed command: '%s' (from line: '%s', upper: '%s')\n", command, line, upperLine)

		// Ignore http requests (web browsers, etc.)
		// Just return bad command instead of panicking
		if strings.Contains(command, "HTTP") {
			if !s.write(protocol.PREPARED_S_BAD_COMMAND) {
				return
			}
			continue
		}

		// Match commands - use strings.EqualFold for case-insensitive comparison
		// This is more robust than string comparison
		switch {
		case command == "EHLO" || command == "HELO":
			s.handleEHLO(line)
		case command == "MAIL":
			s.handleMailFrom(line)
		case command == "RCPT":
			s.handleRctpTo(line)
		case command == "DATA":
			s.handleData(line)
		case command == "STARTTLS":
			s.handleStartTLS(line)
		case command == "QUIT":
			s.handleQuit(line)
			return // Connection will close
		case command == "RSET":
			s.handleRset(line)
		default:
			if !s.write(protocol.PREPARED_S_BAD_COMMAND) {
				return // Connection broken
			}
		}
	}
}

func (s *ServerConn) write(str string) bool {
	if s.client == nil {
		return false
	}

	// Ensure SMTP protocol compliance: all lines must end with \r\n
	// Add \r\n if not already present
	if len(str) < 2 || str[len(str)-2:] != "\r\n" {
		str += "\r\n"
	}

	// Set write deadline to prevent indefinite blocking
	s.client.SetWriteDeadline(time.Now().Add(30 * time.Second))

	// Print transmission to client (trim \r\n for cleaner output)
	output := strings.TrimRight(str, "\r\n")
	fmt.Printf("SERVER -> CLIENT: %s\n", output)

	// Write directly to connection - ensure all bytes are written
	data := []byte(str)
	for len(data) > 0 {
		n, err := s.client.Write(data)
		if err != nil {
			// Handle broken pipe and connection errors gracefully
			if netErr, ok := err.(*net.OpError); ok {
				// Check if it's a broken pipe (EPIPE) - this is normal when client closes
				if sysErr, ok := netErr.Err.(*os.SyscallError); ok && sysErr.Err == syscall.EPIPE {
					return false
				}
				// Other network errors
				fmt.Fprintf(os.Stderr, "SERVER: Write error (connection broken): %v\n", netErr)
				return false
			}
			// Check for syscall errors (broken pipe on Unix, or other syscall errors)
			if sysErr, ok := err.(*os.SyscallError); ok {
				if sysErr.Err == syscall.EPIPE {
					return false
				}
			}
			// For any other write error, log and return false
			fmt.Fprintf(os.Stderr, "SERVER: Write error: %v\n", err)
			return false
		}
		data = data[n:]
	}
	return true
}

func (s *ServerConn) read() string {
	if s.client == nil || s.reader == nil {
		return ""
	}

	// Set read deadline to prevent indefinite blocking
	// Use longer timeout for SMTP (clients might take time to respond)
	s.client.SetReadDeadline(time.Now().Add(60 * time.Second))

	// Use ReadLine directly instead of wrapper for better control
	line, isPrefix, err := s.reader.ReadLine()
	if err != nil {
		if err == io.EOF {
			fmt.Printf("SERVER <- CLIENT: (EOF - connection closed)\n")
			return ""
		}
		// Check if it's a timeout error
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			// Timeout - might be normal, but log it
			fmt.Printf("SERVER <- CLIENT: (timeout waiting for command)\n")
			return ""
		}
		// Other errors
		fmt.Printf("SERVER <- CLIENT: (read error: %v)\n", err)
		return ""
	}

	// Handle long lines (shouldn't happen in SMTP, but be safe)
	var fullLine []byte = line
	for isPrefix {
		line, isPrefix, err = s.reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			break
		}
		fullLine = append(fullLine, line...)
	}

	// Convert to string and append CRLF (SMTP standard)
	lineStr := string(fullLine) + "\r\n"

	// Print transmission from client (trim \r\n for cleaner output)
	output := strings.TrimRight(lineStr, "\r\n")
	fmt.Printf("SERVER <- CLIENT: %s\n", output)

	return lineStr
}

func (s *ServerConn) handleEHLO(line string) {
	if s.state != protocol.STATE_EHLO {
		return
	}
	// Extract domain (for validation)
	parts := strings.Split(line, " ")
	if len(parts) < 2 {
		if !s.write(protocol.PREPARED_S_BAD_SYNTAX) {
			return
		}
		return
	}

	// Get client domain from EHLO command
	clientDomain := parts[1]
	// Respond with success code and supported extensions
	// Use configured server domain instead of client's domain
	serverDomain := s.config.ServerDomain

	// First continuation line: 250-<serverDomain> Hello <clientDomain>
	// SMTP requires continuation lines to use hyphen (250-), not space (250 )
	firstLine := fmt.Sprintf("250-%s Hello %s\r\n", serverDomain, clientDomain)
	if !s.write(firstLine) {
		return
	}

	// Additional continuation lines: 250-<extension>

	// Advertise STARTTLS if TLS is configured (required by Gmail and many modern SMTP clients)
	if s.tlsConfig != nil {
		if !s.write("250-STARTTLS\r\n") {
			return
		}
	}

	if s.relay {
		// 250-AUTH <method>
		if !s.write("250-AUTH PLAIN LOGIN\r\n") {
			return
		}
	}
	// 250-8BITMIME
	if !s.write("250-8BITMIME\r\n") {
		return
	}

	// Final line: 250 <final message> (with space, not hyphen)
	if !s.write(protocol.PREPARED_S_ACKNOWLEDGE) {
		return
	}

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
		if !s.write(protocol.PREPARED_S_BAD_SEQUENCE) {
			return
		}
		return
	}

	// MAIL FROM command should be case-insensitive per SMTP spec
	upperLine := strings.ToUpper(line)
	if !strings.HasPrefix(upperLine, "MAIL FROM:") {
		if !s.write(protocol.PREPARED_S_BAD_COMMAND) {
			return
		}
		return
	}
	// Find the colon (should be at ": " or just ":")
	colonIndex := strings.Index(line, ":")
	if colonIndex == -1 {
		if !s.write(protocol.PREPARED_S_BAD_SYNTAX) {
			return
		}
		return
	}
	remainder := strings.TrimSpace(line[colonIndex+1:])

	if remainder == "" {
		if !s.write(protocol.PREPARED_S_BAD_SYNTAX) {
			return
		}
		return
	}

	// Extract address (should be in <address> format)
	addEnd := strings.Index(remainder, ">")
	if addEnd == -1 {
		// No closing bracket found
		if !s.write(protocol.PREPARED_S_BAD_SYNTAX) {
			return
		}
		return
	}
	address := smtputil.CleanEmail(remainder[:addEnd+1])
	if address == "" {
		// Invalid address
		if !s.write(protocol.PREPARED_S_BAD_SYNTAX) {
			return
		}
		return
	}

	// Verify sender email address
	if s.senderVerifier != nil && !s.senderVerifier.VerifyEmail(address) {
		if !s.write(protocol.PREPARED_S_BAD_SYNTAX) {
			return
		}
		return
	}

	s.mail.SetFrom(address)

	// Check if there are parameters after the address
	if len(remainder) > addEnd+1 {
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
	}

	// Send acknowledgment and change state to RCPT_TO
	// CRITICAL: Always set state to STATE_RCPT_TO regardless of whether parameters exist
	if !s.write(protocol.PREPARED_S_ACKNOWLEDGE) {
		return
	}
	s.state = protocol.STATE_RCPT_TO

}

func (s *ServerConn) handleRctpTo(line string) { // todo handle NOTIFY=SUCCESS,FAILURE,DENY
	if s.state != protocol.STATE_RCPT_TO {
		if !s.write(protocol.PREPARED_S_BAD_SEQUENCE) {
			return
		}
		return
	}
	// RCPT TO command should be case-insensitive per SMTP spec
	// Expected format is "RCPT TO:<address>"
	upperLine := strings.ToUpper(line)
	if !strings.HasPrefix(upperLine, "RCPT TO:") {
		if !s.write(protocol.PREPARED_S_BAD_COMMAND) {
			return
		}
		return
	}
	// Find the colon (should be at ": " or just ":")
	colonIndex := strings.Index(line, ":")
	if colonIndex == -1 {
		if !s.write(protocol.PREPARED_S_BAD_SYNTAX) {
			return
		}
		return
	}
	remainder := strings.TrimSpace(line[colonIndex+1:])
	if remainder == "" {
		if !s.write(protocol.PREPARED_S_BAD_SYNTAX) {
			return
		}
		return
	}
	// Extract address (should be in <address> format)
	addEnd := strings.Index(remainder, ">")
	if addEnd == -1 {
		// No closing bracket found
		if !s.write(protocol.PREPARED_S_BAD_SYNTAX) {
			return
		}
		return
	}
	address := smtputil.CleanEmail(remainder[:addEnd+1])
	if address == "" {
		// Invalid address
		if !s.write(protocol.PREPARED_S_BAD_SYNTAX) {
			return
		}
		return
	}

	// Check if email exists using handler (default returns false)
	if s.handlers != nil && s.handlers.EmailExistsChecker != nil {
		if !s.handlers.EmailExistsChecker(address) {
			// Email does not exist
			if !s.write(protocol.PREPARED_S_BAD_SYNTAX) {
				return
			}
			return
		}
	}

	s.mail.AppendTo(address)
	if !s.write(protocol.PREPARED_S_ACKNOWLEDGE) {
		return
	}
}

func (s *ServerConn) handleData(line string) {
	if s.state == protocol.STATE_RCPT_TO {
		s.state = protocol.STATE_DATA
	}
	if s.state != protocol.STATE_DATA {
		if !s.write(protocol.PREPARED_S_BAD_SEQUENCE) {
			return
		}
		return
	}
	if !s.write(protocol.PREPARED_S_START_DATA) {
		return
	}

	// Read headers until blank line
	headers := *stringutil.NewStringBuilder()
	body := *stringutil.NewStringBuilder()
	inHeaders := true
	bodyStarted := false
	currentHeaderName := ""
	currentHeaderValue := *stringutil.NewStringBuilder()

	for {
		rawLine := s.read()
		// Check if connection was closed (empty read means connection closed)
		if rawLine == "" {
			fmt.Printf("SERVER: Connection closed by client during DATA\n")
			return // Connection broken during DATA phase
		}

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
			if !s.write(protocol.PREPARED_S_TRANSACTION_FAILED) {
				return
			}
			return
		}
	}

	// Acknowledge successful data reception
	if !s.write(protocol.PREPARED_S_ACKNOWLEDGE) {
		return
	}

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
	// STARTTLS can only be used before authentication or mail transaction
	// Allow it in STATE_EHLO or STATE_MAIL_FROM (after EHLO but before MAIL FROM is sent)
	// According to RFC 3207, STARTTLS must be sent after EHLO and before any mail transaction
	if s.state != protocol.STATE_EHLO && s.state != protocol.STATE_MAIL_FROM {
		if !s.write(protocol.PREPARED_S_BAD_SEQUENCE) {
			return
		}
		return
	}

	// Check if TLS config is available
	if s.tlsConfig == nil {
		if !s.write(protocol.PREPARED_S_BAD_COMMAND) {
			return
		}
		return
	}

	// Send "220 Ready to start TLS"
	if !s.write(protocol.PREPARED_S_STARTTLS_READY) {
		return
	}

	// Perform TLS handshake
	tlsConn := tls.Server(s.client, s.tlsConfig)
	err := tlsConn.Handshake()
	if err != nil {
		// TLS handshake failed - log and return
		fmt.Fprintf(os.Stderr, "SERVER: TLS handshake failed: %v\n", err)
		return
	}

	// Update connection and reader with TLS-wrapped connection
	// Direct assignment - tls.Conn implements net.Conn
	s.client = tlsConn
	s.reader = bufio.NewReader(tlsConn)

	// Reset state to EHLO - client must send EHLO again after STARTTLS
	s.state = protocol.STATE_EHLO
}

func (s *ServerConn) handleQuit(line string) {
	// Send "221 Bye" response
	// Connection will be closed by caller, so ignore write errors
	s.write(protocol.PREPARED_S_BYE)
}

func (s *ServerConn) handleRset(line string) {
	// Reset the mail transaction
	s.mail = mail.Mail{}
	s.state = protocol.STATE_EHLO

	// Send acknowledgment
	if !s.write(protocol.PREPARED_S_ACKNOWLEDGE) {
		return
	}
}

// Close closes the server connection and underlying client connection
// This should be called when the connection is no longer needed
// Note: The connection is typically closed automatically when handle() returns
func (s *ServerConn) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

// GetConn returns the underlying connection (useful for external management)
func (s *ServerConn) GetConn() net.Conn {
	return s.client
}
