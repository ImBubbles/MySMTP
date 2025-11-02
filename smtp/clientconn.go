package smtp

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/ImBubbles/MySMTP/config"
	"github.com/ImBubbles/MySMTP/mail"
	"github.com/ImBubbles/MySMTP/smtp/protocol"
	"github.com/ImBubbles/MySMTP/util/conn"
)

// ClientConn handle client-side SMTP connections to send emails
type ClientConn struct {
	conn      net.Conn
	state     protocol.SMTPStates
	reader    *bufio.Reader
	mail      mail.Mail
	tlsConfig *tls.Config
	hostname  string
}

func NewClientConn(conn net.Conn, mail mail.Mail) *ClientConn {
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
		conn:     conn,
		state:    protocol.STATE_EHLO,
		reader:   bufio.NewReader(conn),
		mail:     mail,
		hostname: hostname,
	}
	clientConn.handle()
	return clientConn
}

// SetTLSConfig sets the TLS configuration for STARTTLS
func (c *ClientConn) SetTLSConfig(config *tls.Config) {
	c.tlsConfig = config
}

// NewClientConnFromJSON creates a new client connection from JSON bytes
// This is a convenience function to easily create a ClientConn from JSON (e.g., from a backend API)
func NewClientConnFromJSON(conn net.Conn, jsonBytes []byte) (*ClientConn, error) {
	mail, err := mail.FromJSON(jsonBytes)
	if err != nil {
		return nil, err
	}
	return NewClientConn(conn, *mail), nil
}

// NewClientConnFromJSONString creates a new client connection from a JSON string
// This is a convenience function to easily create a ClientConn from a JSON string
func NewClientConnFromJSONString(conn net.Conn, jsonStr string) (*ClientConn, error) {
	return NewClientConnFromJSON(conn, []byte(jsonStr))
}

// NewClientConnFromJSONMail creates a new client connection from a JSONMail struct
// This is a convenience function to easily create a ClientConn from a JSONMail
func NewClientConnFromJSONMail(conn net.Conn, jsonMail *mail.JSONMail) *ClientConn {
	mail := jsonMail.ToMail()
	return NewClientConn(conn, *mail)
}

func (c *ClientConn) handle() {
	// Read server greeting (220 Service Ready)
	response := c.read()
	if !c.isSuccessCode(response, protocol.CODE_READY) {
		panic("Failed to receive server greeting")
	}

	// Send EHLO command
	c.sendEHLO()

	// Send MAIL FROM command
	c.sendMailFrom()

	// Send RCPT TO commands for all recipients
	c.sendRcptTo()

	// Send DATA command
	c.sendData()

	// Send email content (headers + body)
	c.sendEmailContent()

	// Read final acknowledgment
	response = c.read()
	if !c.isSuccessCode(response, protocol.CODE_ACKNOWLEDGE) {
		panic("Failed to send email data")
	}

	// Send QUIT command
	c.sendQuit()
}

func (c *ClientConn) write(str string) {
	// Update write deadline before each write (net.Conn interface supports SetWriteDeadline)
	c.conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	_, err := c.conn.Write([]byte(str))
	if err != nil {
		panic(fmt.Sprintf("Write error: %v", err))
	}
}

func (c *ClientConn) read() string {
	// Update read deadline before each read (net.Conn interface supports SetReadDeadline)
	c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	response := conn.Read(c.reader)
	if response == "" {
		panic("Read error: empty response (connection may have closed)")
	}
	return response
}

// isSuccessCode checks if the response code matches the expected code
func (c *ClientConn) isSuccessCode(response string, expectedCode protocol.SMTPCode) bool {
	// Response format: "250 OK\r\n" or "220 Service Ready\r\n"
	response = strings.TrimSpace(response)
	if len(response) >= 3 {
		codeStr := response[:3]
		var code protocol.SMTPCode
		fmt.Sscanf(codeStr, "%d", &code)
		return code == expectedCode
	}
	return false
}

// parseResponseCode extracts the response code from server response
func (c *ClientConn) parseResponseCode(response string) protocol.SMTPCode {
	response = strings.TrimSpace(response)
	if len(response) >= 3 {
		var code protocol.SMTPCode
		fmt.Sscanf(response[:3], "%d", &code)
		return code
	}
	return protocol.CODE_INTERNAL_SERVER_ERROR
}

func (c *ClientConn) sendEHLO() {
	ehloCmd := fmt.Sprintf("%s %s\r\n", protocol.COMMAND_EHLO, c.hostname)
	c.write(ehloCmd)

	// Read server responses (may be multiple lines)
	// SMTP multi-line responses use "-" as 4th character for continuation
	// Final line has space (or sometimes nothing) as 4th character
	for {
		response := c.read()
		if response == "" {
			panic("EHLO failed: empty response")
		}

		code := c.parseResponseCode(response)

		// Check if this is an error
		if code >= protocol.CODE_INTERNAL_SERVER_ERROR {
			panic(fmt.Sprintf("EHLO failed: %s", response))
		}

		// Check if this is a continuation line (SMTP standard: 4th char is "-")
		// Continuation format: "250-extension\r\n"
		// Final line format: "250 OK\r\n" or "250 message\r\n"
		if len(response) >= 4 {
			// Check the 4th character (index 3) after the 3-digit code
			if response[3] == '-' {
				// Continuation line, continue reading
				continue
			}
			// If 4th char is space or the code is 250, it's the final line
			if response[3] == ' ' || code == protocol.CODE_ACKNOWLEDGE {
				break
			}
		} else if len(response) >= 3 {
			// Short response, check if it's a valid 250 response
			if code == protocol.CODE_ACKNOWLEDGE {
				break
			}
		}

		// Safety: if we got here and code is 250, assume it's final
		if code == protocol.CODE_ACKNOWLEDGE {
			break
		}

		// If we got a non-250 success code or unexpected response, break to avoid infinite loop
		break
	}

	c.state = protocol.STATE_MAIL_FROM
}

func (c *ClientConn) sendMailFrom() {
	from := c.mail.GetFrom()
	if from == "" {
		panic("No FROM address specified")
	}

	// Format: MAIL FROM:<address>
	mailCmd := fmt.Sprintf("%s FROM:<%s>\r\n", protocol.COMMAND_MAIL, from)
	c.write(mailCmd)

	response := c.read()
	if !c.isSuccessCode(response, protocol.CODE_ACKNOWLEDGE) {
		panic(fmt.Sprintf("MAIL FROM failed: %s", response))
	}

	c.state = protocol.STATE_RCPT_TO
}

func (c *ClientConn) sendRcptTo() {
	// Send RCPT TO for all "to" recipients
	for _, to := range c.mail.GetTo() {
		rcptCmd := fmt.Sprintf("%s TO:<%s>\r\n", protocol.COMMAND_RCPT, to)
		c.write(rcptCmd)

		response := c.read()
		if !c.isSuccessCode(response, protocol.CODE_ACKNOWLEDGE) {
			panic(fmt.Sprintf("RCPT TO failed for %s: %s", to, response))
		}
	}

	// Send RCPT TO for CC recipients (they also need RCPT TO)
	for _, cc := range c.mail.GetCC() {
		rcptCmd := fmt.Sprintf("%s TO:<%s>\r\n", protocol.COMMAND_RCPT, cc)
		c.write(rcptCmd)

		response := c.read()
		if !c.isSuccessCode(response, protocol.CODE_ACKNOWLEDGE) {
			panic(fmt.Sprintf("RCPT TO failed for CC %s: %s", cc, response))
		}
	}

	// BCC recipients also need RCPT TO, but are not included in headers
	for _, bcc := range c.mail.GetBCC() {
		rcptCmd := fmt.Sprintf("%s TO:<%s>\r\n", protocol.COMMAND_RCPT, bcc)
		c.write(rcptCmd)

		response := c.read()
		if !c.isSuccessCode(response, protocol.CODE_ACKNOWLEDGE) {
			panic(fmt.Sprintf("RCPT TO failed for BCC %s: %s", bcc, response))
		}
	}

	c.state = protocol.STATE_DATA
}

func (c *ClientConn) sendData() {
	// Send DATA command
	dataCmd := fmt.Sprintf("%s\r\n", protocol.COMMAND_DATA)
	c.write(dataCmd)

	// Read server response (354 Start mail input)
	response := c.read()
	if !c.isSuccessCode(response, protocol.CODE_START_MAIL_INPUT) {
		panic(fmt.Sprintf("DATA command failed: %s", response))
	}
}

func (c *ClientConn) sendEmailContent() {
	// Build email headers
	headers := c.buildHeaders()

	// Send headers
	c.write(headers)

	// Send blank line to separate headers from body
	c.write("\r\n")

	// Send body with SMTP transparency handling
	body := c.mail.GetData()
	c.writeBody(body)

	// Send terminator (line containing only ".")
	c.write(".\r\n")
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

func (c *ClientConn) writeBody(body string) {
	if body == "" {
		return
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
		c.write(line + "\r\n")
	}
}

func (c *ClientConn) sendQuit() {
	quitCmd := fmt.Sprintf("%s\r\n", protocol.COMMAND_QUIT)
	c.write(quitCmd)

	// Read server response (221 Bye)
	response := c.read()
	if !c.isSuccessCode(response, protocol.CODE_QUIT) {
		// Log but don't panic - connection will close anyway
		fmt.Fprintf(os.Stderr, "QUIT response unexpected: %s\n", response)
	}
}
