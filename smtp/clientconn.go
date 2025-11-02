package smtp

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/ImBubbles/MySMTP/config"
	"github.com/ImBubbles/MySMTP/mail"
	"github.com/ImBubbles/MySMTP/smtp/protocol"
	"github.com/ImBubbles/MySMTP/util/conn"
)

// ClientConn handle client-side SMTP connections to send emails
type ClientConn struct {
	conn       net.Conn
	state      protocol.SMTPStates
	reader     *bufio.Reader
	mail       mail.Mail
	tlsConfig  *tls.Config
	hostname   string // Client hostname (for EHLO)
	serverName string // Server hostname (for TLS SNI)
	serverHost string // Server host from DialSMTP (for SNI fallback)
}

func NewClientConn(conn net.Conn, mail mail.Mail) (*ClientConn, error) {
	// Load config for hostname
	cfg := config.GetConfig()
	hostname := cfg.ClientHostname
	if hostname == "" {
		// Fallback to system hostname if not configured
		var err error
		hostname, err = os.Hostname()
		if err != nil {
			hostname = "localhost"
		}
	}

	clientConn := &ClientConn{
		conn:       conn,
		state:      protocol.STATE_EHLO,
		reader:     bufio.NewReader(conn),
		mail:       mail,
		hostname:   hostname,
		serverHost: "", // Will be set if using NewClientConnFromHost
	}
	err := clientConn.handle()
	return clientConn, err
}

// NewClientConnFromHost creates a new client connection and stores the server hostname
// This is useful for Gmail and other SMTP servers that require proper SNI in TLS
// The host parameter should be the SMTP server hostname (e.g., "smtp.gmail.com")
func NewClientConnFromHost(host string, conn net.Conn, mail mail.Mail) (*ClientConn, error) {
	client, err := NewClientConn(conn, mail)
	if err != nil {
		return nil, err
	}
	// Store the server host for SNI fallback if ServerName is not explicitly set
	client.serverHost = host
	// If ServerName is not set, use the host for SNI
	if client.serverName == "" {
		client.serverName = host
	}
	return client, nil
}

// SetTLSConfig sets the TLS configuration for STARTTLS
func (c *ClientConn) SetTLSConfig(config *tls.Config) {
	c.tlsConfig = config
}

// SetServerName sets the server hostname for TLS SNI (Server Name Indication)
// This should be the MX server hostname (e.g., "gmail-smtp-in.l.google.com")
func (c *ClientConn) SetServerName(serverName string) {
	c.serverName = serverName
}

// NewClientConnFromJSON creates a new client connection from JSON bytes
// This is a convenience function to easily create a ClientConn from JSON (e.g., from a backend API)
func NewClientConnFromJSON(conn net.Conn, jsonBytes []byte) (*ClientConn, error) {
	mail, err := mail.FromJSON(jsonBytes)
	if err != nil {
		return nil, err
	}
	return NewClientConn(conn, *mail)
}

// NewClientConnFromJSONString creates a new client connection from a JSON string
// This is a convenience function to easily create a ClientConn from a JSON string
func NewClientConnFromJSONString(conn net.Conn, jsonStr string) (*ClientConn, error) {
	return NewClientConnFromJSON(conn, []byte(jsonStr))
}

// NewClientConnFromJSONMail creates a new client connection from a JSONMail struct
// This is a convenience function to easily create a ClientConn from a JSONMail
func NewClientConnFromJSONMail(conn net.Conn, jsonMail *mail.JSONMail) (*ClientConn, error) {
	mail := jsonMail.ToMail()
	return NewClientConn(conn, *mail)
}

// NewClientConnFromHostAndJSON creates a new client connection from host and JSON bytes
// This stores the server hostname for proper SNI in TLS (required for Gmail and others)
// The host parameter should be the SMTP server hostname (e.g., "smtp.gmail.com")
func NewClientConnFromHostAndJSON(host string, conn net.Conn, jsonBytes []byte) (*ClientConn, error) {
	mail, err := mail.FromJSON(jsonBytes)
	if err != nil {
		return nil, err
	}
	return NewClientConnFromHost(host, conn, *mail)
}

// DialSMTP creates a TCP connection to an SMTP server
// It uses port 587 by default (submission port with STARTTLS), which is preferred over port 25
// Port 25 is often blocked by ISPs and is mainly used for server-to-server communication
// You can specify a custom port, or use the configured SMTP_CLIENT_PORT (default 587)
// Returns a net.Conn ready for use with NewClientConn
// IMPORTANT: For Gmail and other modern SMTP servers, use NewClientConnFromHost() or
// NewClientConnDialSMTP() instead to ensure proper SNI for TLS
func DialSMTP(host string, port ...uint16) (net.Conn, error) {
	var smtpPort uint16

	// Use provided port, or config port, or default to 587 (submission with STARTTLS)
	if len(port) > 0 && port[0] != 0 {
		smtpPort = port[0]
	} else {
		cfg := config.GetConfig()
		if cfg.ClientPort != 0 {
			smtpPort = cfg.ClientPort
		} else {
			smtpPort = 587 // Default to port 587 (submission with STARTTLS)
		}
	}

	address := net.JoinHostPort(host, fmt.Sprintf("%d", smtpPort))

	// Create dialer with timeout to prevent hanging
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}

	conn, err := dialer.Dial("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SMTP server %s: %w", address, err)
	}

	return conn, nil
}

// NewClientConnDialSMTP is a convenience function that dials SMTP and creates a client connection
// This automatically stores the hostname for proper SNI in TLS (required for Gmail and others)
// Usage: client, err := NewClientConnDialSMTP("smtp.gmail.com", mail)
func NewClientConnDialSMTP(host string, mail mail.Mail, port ...uint16) (*ClientConn, error) {
	conn, err := DialSMTP(host, port...)
	if err != nil {
		return nil, err
	}
	return NewClientConnFromHost(host, conn, mail)
}

func (c *ClientConn) handle() error {
	// Read server greeting (220 Service Ready)
	response, err := c.read()
	if err != nil {
		return fmt.Errorf("failed to read server greeting: %w", err)
	}
	if response == "" {
		return errors.New("connection closed by server")
	}
	if !c.isSuccessCode(response, protocol.CODE_READY) {
		return fmt.Errorf("server greeting failed: %s", response)
	}

	// Send EHLO command
	if err := c.sendEHLO(); err != nil {
		return fmt.Errorf("EHLO failed: %w", err)
	}

	// Send MAIL FROM command
	if err := c.sendMailFrom(); err != nil {
		return fmt.Errorf("MAIL FROM failed: %w", err)
	}

	// Send RCPT TO commands for all recipients
	if err := c.sendRcptTo(); err != nil {
		return fmt.Errorf("RCPT TO failed: %w", err)
	}

	// Send DATA command
	if err := c.sendData(); err != nil {
		return fmt.Errorf("DATA command failed: %w", err)
	}

	// Send email content (headers + body)
	if err := c.sendEmailContent(); err != nil {
		return fmt.Errorf("sending email content failed: %w", err)
	}

	// Read final acknowledgment
	response, err = c.read()
	if err != nil {
		return fmt.Errorf("failed to read final acknowledgment: %w", err)
	}
	if response == "" {
		return errors.New("connection closed by server before final acknowledgment")
	}
	if !c.isSuccessCode(response, protocol.CODE_ACKNOWLEDGE) {
		return fmt.Errorf("final acknowledgment failed: %s", response)
	}

	// Send QUIT command (ignore errors as connection will close)
	c.sendQuit()
	return nil
}

func (c *ClientConn) write(str string) error {
	// Update write deadline before each write (net.Conn interface supports SetWriteDeadline)
	c.conn.SetWriteDeadline(time.Now().Add(30 * time.Second))

	// Debug: Print what we're sending (trim \r\n for cleaner output)
	output := strings.TrimRight(str, "\r\n")
	fmt.Printf("CLIENT -> SERVER: %s\n", output)

	// Ensure all bytes are written (handle partial writes)
	data := []byte(str)
	for len(data) > 0 {
		n, err := c.conn.Write(data)
		if err != nil {
			// Handle broken pipe and connection errors gracefully
			// Check for syscall.EPIPE (broken pipe) or other connection errors
			if netErr, ok := err.(*net.OpError); ok {
				return fmt.Errorf("write error (connection broken): %w", netErr)
			}
			// Check for syscall errors (broken pipe on Unix, or other syscall errors)
			if sysErr, ok := err.(*os.SyscallError); ok {
				if sysErr.Err == syscall.EPIPE {
					return fmt.Errorf("write error (broken pipe): %w", sysErr)
				}
			}
			// For any other write error, return it
			return fmt.Errorf("write error: %w", err)
		}
		// Move forward in the data slice
		data = data[n:]
	}
	return nil
}

func (c *ClientConn) read() (string, error) {
	// Update read deadline before each read (net.Conn interface supports SetReadDeadline)
	c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	response := conn.Read(c.reader)

	// Debug: Print what we're receiving (trim \r\n for cleaner output)
	output := strings.TrimRight(response, "\r\n")
	fmt.Printf("CLIENT <- SERVER: %s\n", output)

	// Check for empty response (connection closed)
	if response == "" {
		return "", errors.New("read error: empty response (connection may have closed)")
	}

	// Check for blank line (just CRLF or whitespace)
	// Blank lines from server are unexpected except in DATA phase
	trimmed := strings.TrimSpace(response)
	if trimmed == "" {
		// This is a blank line - log it for debugging
		fmt.Fprintf(os.Stderr, "CLIENT: Received blank line (connection may be closing)\n")
		return "", errors.New("read error: blank response (server sent empty line)")
	}

	return response, nil
}

// isSuccessCode checks if the response code matches the expected code
func (c *ClientConn) isSuccessCode(response string, expectedCode protocol.SMTPCode) bool {
	// Response format: "250 OK\r\n" or "220 Service Ready\r\n"
	trimmed := strings.TrimSpace(response)

	// Check for blank response
	if trimmed == "" {
		return false
	}

	if len(trimmed) >= 3 {
		codeStr := trimmed[:3]
		var code protocol.SMTPCode
		n, err := fmt.Sscanf(codeStr, "%d", &code)
		if err != nil || n != 1 {
			// Failed to parse code - return false
			return false
		}
		return code == expectedCode
	}
	return false
}

// parseResponseCode extracts the response code from server response
func (c *ClientConn) parseResponseCode(response string) protocol.SMTPCode {
	trimmed := strings.TrimSpace(response)

	// Check for blank response
	if trimmed == "" {
		return protocol.CODE_INTERNAL_SERVER_ERROR
	}

	if len(trimmed) >= 3 {
		var code protocol.SMTPCode
		n, err := fmt.Sscanf(trimmed[:3], "%d", &code)
		if err != nil || n != 1 {
			// Failed to parse code - return error code
			return protocol.CODE_INTERNAL_SERVER_ERROR
		}
		return code
	}
	return protocol.CODE_INTERNAL_SERVER_ERROR
}

// sendEHLO sends EHLO command and optionally upgrades to TLS if available
// allowTLS controls whether to attempt STARTTLS (used to prevent recursion)
func (c *ClientConn) sendEHLO() error {
	return c.sendEHLOWithTLS(true)
}

// sendEHLOWithTLS is the internal implementation that allows controlling TLS behavior
func (c *ClientConn) sendEHLOWithTLS(allowTLS bool) error {
	ehloCmd := fmt.Sprintf("%s %s\r\n", protocol.COMMAND_EHLO, c.hostname)
	if err := c.write(ehloCmd); err != nil {
		return fmt.Errorf("failed to write EHLO command: %w", err)
	}

	// Read server responses (may be multiple lines)
	// SMTP multi-line responses use "-" as 4th character for continuation
	// Final line has space (or sometimes nothing) as 4th character
	extensions := make([]string, 0)

	for {
		response, err := c.read()
		if err != nil {
			return fmt.Errorf("failed to read EHLO response: %w", err)
		}
		if response == "" {
			return errors.New("EHLO failed: empty response")
		}

		code := c.parseResponseCode(response)

		// Check if this is an error
		if code >= protocol.CODE_INTERNAL_SERVER_ERROR {
			return fmt.Errorf("EHLO failed: %s", response)
		}

		// Check if this is a continuation line (SMTP standard: 4th char is "-")
		// Continuation format: "250-extension\r\n"
		// Final line format: "250 OK\r\n" or "250 message\r\n"
		if len(response) >= 4 {
			// Check the 4th character (index 3) after the 3-digit code
			if response[3] == '-' {
				// Continuation line - extract extension
				// Format: "250-EXTENSION\r\n"
				if len(response) > 5 {
					extension := strings.TrimSpace(response[4:])
					if extension != "" {
						extensions = append(extensions, strings.ToUpper(extension))
					}
				}
				continue
			}
			// If 4th char is space, it's the final line
			if response[3] == ' ' {
				break
			}
		}

		// If code is 250 (ACKNOWLEDGE), this is the final line (even without space)
		// This handles cases like "250OK\r\n" where there's no space
		if code == protocol.CODE_ACKNOWLEDGE {
			break
		}

		// If we got a valid success code (not 250), break anyway to avoid infinite loop
		if code < protocol.CODE_INTERNAL_SERVER_ERROR && code != protocol.CODE_ACKNOWLEDGE {
			break
		}

		// Safety: if we got here, break to avoid infinite loop
		break
	}

	// Check if STARTTLS is supported and use it if available (only if allowTLS is true)
	// After STARTTLS, we send EHLO again but don't check for STARTTLS again
	// Many modern SMTP servers (like Gmail) require or strongly prefer STARTTLS
	if allowTLS {
		supportsSTARTTLS := false
		for _, ext := range extensions {
			// Check for STARTTLS extension (case-insensitive)
			if strings.Contains(strings.ToUpper(ext), "STARTTLS") {
				supportsSTARTTLS = true
				break
			}
		}

		// If STARTTLS is available, use it
		// Create default TLS config if none provided (required for many modern SMTP servers like Gmail)
		if supportsSTARTTLS {
			if c.tlsConfig == nil {
				// Create default TLS config for STARTTLS
				// For Gmail and other modern SMTP servers, we need proper ServerName
				// The ServerName should match the MX hostname
				c.tlsConfig = &tls.Config{
					ServerName:         c.serverName,
					InsecureSkipVerify: false, // Don't skip verification for security
				}
			}
			if err := c.sendSTARTTLS(); err != nil {
				return fmt.Errorf("STARTTLS failed: %w", err)
			}
			// After STARTTLS, must send EHLO again (but don't check for STARTTLS again)
			if err := c.sendEHLOWithTLS(false); err != nil {
				return fmt.Errorf("EHLO after STARTTLS failed: %w", err)
			}
			return nil
		}
	}

	c.state = protocol.STATE_MAIL_FROM
	return nil
}

// sendSTARTTLS sends STARTTLS command and upgrades connection to TLS
func (c *ClientConn) sendSTARTTLS() error {
	// Send STARTTLS command
	starttlsCmd := fmt.Sprintf("%s\r\n", protocol.COMMAND_STARTTLS)
	if err := c.write(starttlsCmd); err != nil {
		return fmt.Errorf("failed to write STARTTLS command: %w", err)
	}

	// Read server response (220 Ready to start TLS)
	response, err := c.read()
	if err != nil {
		return fmt.Errorf("failed to read STARTTLS response: %w", err)
	}
	if response == "" {
		return errors.New("STARTTLS failed: empty response")
	}

	code := c.parseResponseCode(response)
	if code != protocol.CODE_READY {
		return fmt.Errorf("STARTTLS failed: %s", response)
	}

	// Perform TLS handshake
	// Determine server name for TLS SNI (Server Name Indication)
	// For Gmail and other modern SMTP servers, the ServerName must match the MX hostname
	// Priority: 1) serverName (explicitly set), 2) serverHost (from DialSMTP), 3) connection address
	serverName := c.serverName
	if serverName == "" {
		// Use the server host from DialSMTP if available
		if c.serverHost != "" {
			serverName = c.serverHost
		} else {
			// Try to get hostname from connection's remote address
			if addr := c.conn.RemoteAddr(); addr != nil {
				// For TLS SNI, we need the hostname, not the IP
				addrStr := addr.String()
				if colonIdx := strings.LastIndex(addrStr, ":"); colonIdx > 0 {
					hostPart := addrStr[:colonIdx]
					// Check if it's an IP address
					if net.ParseIP(hostPart) != nil {
						// It's an IP - we can't use it for SNI
						// Use hostname as fallback (but this won't work well for Gmail)
						serverName = c.hostname
					} else {
						// It's a hostname - use it for SNI
						serverName = hostPart
					}
				}
			}
			if serverName == "" {
				serverName = c.hostname
			}
		}
	}

	// Create TLS config with server name if not already set
	tlsConfig := c.tlsConfig
	if tlsConfig == nil {
		// Use default TLS config if none provided
		tlsConfig = &tls.Config{
			ServerName:         serverName,
			InsecureSkipVerify: false,
		}
	} else if tlsConfig.ServerName == "" {
		// Clone config fields without copying mutex - create new config
		// This avoids copying the sync.RWMutex which shouldn't be copied
		tlsConfig = &tls.Config{
			ServerName:               serverName,
			InsecureSkipVerify:       tlsConfig.InsecureSkipVerify,
			MinVersion:               tlsConfig.MinVersion,
			MaxVersion:               tlsConfig.MaxVersion,
			CipherSuites:             tlsConfig.CipherSuites,
			PreferServerCipherSuites: tlsConfig.PreferServerCipherSuites,
			ClientAuth:               tlsConfig.ClientAuth,
			ClientCAs:                tlsConfig.ClientCAs,
			RootCAs:                  tlsConfig.RootCAs,
			Certificates:             tlsConfig.Certificates,
			GetCertificate:           tlsConfig.GetCertificate,
			GetClientCertificate:     tlsConfig.GetClientCertificate,
		}
	}

	tlsConn := tls.Client(c.conn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return fmt.Errorf("TLS handshake failed: %w", err)
	}

	// Update connection and reader with TLS-wrapped connection
	c.conn = tlsConn
	c.reader = bufio.NewReader(tlsConn)

	// State remains EHLO - client must send EHLO again after STARTTLS
	return nil
}

func (c *ClientConn) sendMailFrom() error {
	from := c.mail.GetFrom()
	if from == "" {
		return errors.New("no FROM address specified")
	}

	// Format: MAIL FROM:<address>
	mailCmd := fmt.Sprintf("%s FROM:<%s>\r\n", protocol.COMMAND_MAIL, from)
	if err := c.write(mailCmd); err != nil {
		return fmt.Errorf("failed to write MAIL FROM command: %w", err)
	}

	response, err := c.read()
	if err != nil {
		return fmt.Errorf("failed to read MAIL FROM response: %w", err)
	}
	if response == "" {
		return errors.New("MAIL FROM failed: empty response (connection closed)")
	}
	if !c.isSuccessCode(response, protocol.CODE_ACKNOWLEDGE) {
		return fmt.Errorf("MAIL FROM failed: %s", response)
	}

	c.state = protocol.STATE_RCPT_TO
	return nil
}

func (c *ClientConn) sendRcptTo() error {
	// Send RCPT TO for all "to" recipients
	for _, to := range c.mail.GetTo() {
		rcptCmd := fmt.Sprintf("%s TO:<%s>\r\n", protocol.COMMAND_RCPT, to)
		if err := c.write(rcptCmd); err != nil {
			return fmt.Errorf("failed to write RCPT TO for %s: %w", to, err)
		}

		response, err := c.read()
		if err != nil {
			return fmt.Errorf("failed to read RCPT TO response for %s: %w", to, err)
		}
		if response == "" {
			return fmt.Errorf("RCPT TO failed for %s: empty response (connection closed)", to)
		}
		if !c.isSuccessCode(response, protocol.CODE_ACKNOWLEDGE) {
			return fmt.Errorf("RCPT TO failed for %s: %s", to, response)
		}
	}

	// Send RCPT TO for CC recipients (they also need RCPT TO)
	for _, cc := range c.mail.GetCC() {
		rcptCmd := fmt.Sprintf("%s TO:<%s>\r\n", protocol.COMMAND_RCPT, cc)
		if err := c.write(rcptCmd); err != nil {
			return fmt.Errorf("failed to write RCPT TO for CC %s: %w", cc, err)
		}

		response, err := c.read()
		if err != nil {
			return fmt.Errorf("failed to read RCPT TO response for CC %s: %w", cc, err)
		}
		if response == "" {
			return fmt.Errorf("RCPT TO failed for CC %s: empty response (connection closed)", cc)
		}
		if !c.isSuccessCode(response, protocol.CODE_ACKNOWLEDGE) {
			return fmt.Errorf("RCPT TO failed for CC %s: %s", cc, response)
		}
	}

	// BCC recipients also need RCPT TO, but are not included in headers
	for _, bcc := range c.mail.GetBCC() {
		rcptCmd := fmt.Sprintf("%s TO:<%s>\r\n", protocol.COMMAND_RCPT, bcc)
		if err := c.write(rcptCmd); err != nil {
			return fmt.Errorf("failed to write RCPT TO for BCC %s: %w", bcc, err)
		}

		response, err := c.read()
		if err != nil {
			return fmt.Errorf("failed to read RCPT TO response for BCC %s: %w", bcc, err)
		}
		if response == "" {
			return fmt.Errorf("RCPT TO failed for BCC %s: empty response (connection closed)", bcc)
		}
		if !c.isSuccessCode(response, protocol.CODE_ACKNOWLEDGE) {
			return fmt.Errorf("RCPT TO failed for BCC %s: %s", bcc, response)
		}
	}

	c.state = protocol.STATE_DATA
	return nil
}

func (c *ClientConn) sendData() error {
	// Send DATA command
	dataCmd := fmt.Sprintf("%s\r\n", protocol.COMMAND_DATA)
	if err := c.write(dataCmd); err != nil {
		return fmt.Errorf("failed to write DATA command: %w", err)
	}

	// Read server response (354 Start mail input)
	response, err := c.read()
	if err != nil {
		return fmt.Errorf("failed to read DATA response: %w", err)
	}
	if response == "" {
		return errors.New("DATA command failed: empty response (connection closed)")
	}
	if !c.isSuccessCode(response, protocol.CODE_START_MAIL_INPUT) {
		return fmt.Errorf("DATA command failed: %s", response)
	}
	return nil
}

func (c *ClientConn) sendEmailContent() error {
	// Build email headers
	headers := c.buildHeaders()

	// Send headers
	if err := c.write(headers); err != nil {
		return fmt.Errorf("failed to write headers: %w", err)
	}

	// Send blank line to separate headers from body
	if err := c.write("\r\n"); err != nil {
		return fmt.Errorf("failed to write header-body separator: %w", err)
	}

	// Send body with SMTP transparency handling
	body := c.mail.GetData()
	if err := c.writeBody(body); err != nil {
		return fmt.Errorf("failed to write body: %w", err)
	}

	// Send terminator (line containing only ".")
	if err := c.write(".\r\n"); err != nil {
		return fmt.Errorf("failed to write terminator: %w", err)
	}
	return nil
}

func (c *ClientConn) buildHeaders() string {
	var builder strings.Builder

	// From header
	from := c.mail.GetFrom()
	if from != "" {
		builder.WriteString(fmt.Sprintf("From: <%s>\r\n", from))
	}

	// To header
	toList := c.mail.GetTo()
	if len(toList) > 0 {
		builder.WriteString(fmt.Sprintf("To: <%s>\r\n", strings.Join(toList, ">, <")))
	}

	// CC header
	ccList := c.mail.GetCC()
	if len(ccList) > 0 {
		builder.WriteString(fmt.Sprintf("Cc: <%s>\r\n", strings.Join(ccList, ">, <")))
	}

	// Subject header
	subject := c.mail.GetSubject()
	if subject != "" {
		builder.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	}

	// Other headers from flags
	flags := c.mail.GetFlags()
	for _, flag := range flags {
		// Format custom headers from flags if needed
		if flag.GetKey() != "" && flag.GetValue() != "" {
			builder.WriteString(fmt.Sprintf("%s: %s\r\n", flag.GetKey(), flag.GetValue()))
		}
	}

	return builder.String()
}

func (c *ClientConn) writeBody(body string) error {
	if body == "" {
		return nil
	}

	// Handle SMTP transparency: lines starting with "." need to have "." prepended
	lines := strings.Split(body, "\n")
	lastIndex := len(lines) - 1
	for i, line := range lines {
		// Remove trailing \r if present (handle both \r\n and \n)
		line = strings.TrimRight(line, "\r")

		// Skip empty lines at the end (but not if it's the only line)
		if line == "" && i == lastIndex && len(lines) > 1 {
			continue
		}

		// If line starts with ".", prepend another "." (SMTP transparency)
		if strings.HasPrefix(line, ".") {
			line = "." + line
		}

		// Send line with CRLF
		if err := c.write(line + "\r\n"); err != nil {
			return fmt.Errorf("failed to write body line %d: %w", i, err)
		}
	}
	return nil
}

func (c *ClientConn) sendQuit() {
	quitCmd := fmt.Sprintf("%s\r\n", protocol.COMMAND_QUIT)
	// Ignore write errors as connection will close anyway
	c.write(quitCmd)

	// Read server response (221 Bye)
	// Ignore read errors as connection will close anyway
	response, _ := c.read()
	if response != "" && !c.isSuccessCode(response, protocol.CODE_QUIT) {
		// Log but don't return error - connection will close anyway
		fmt.Fprintf(os.Stderr, "QUIT response unexpected: %s\n", response)
	}
}

// Close closes the client connection
func (c *ClientConn) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// GetConn returns the underlying connection (useful for external management)
func (c *ClientConn) GetConn() net.Conn {
	return c.conn
}
