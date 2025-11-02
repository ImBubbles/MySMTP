package smtp

import (
	"MySMTP/config"
	"MySMTP/mail"
	"MySMTP/smtp/protocol"
	"MySMTP/util/conn"
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strings"
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
	_, err := c.conn.Write([]byte(str))
	if err != nil {
		panic(err)
	}
}

func (c *ClientConn) read() string {
	return conn.Read(c.reader)
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
	// EHLO response starts with 250 and has multiple lines ending with 250 OK
	for {
		response := c.read()
		code := c.parseResponseCode(response)

		// Check if this is an error
		if code >= protocol.CODE_INTERNAL_SERVER_ERROR {
			panic(fmt.Sprintf("EHLO failed: %s", response))
		}

		// Check if this is the last line (250 OK) - usually the last 250 response
		if code == protocol.CODE_ACKNOWLEDGE {
			// Check if this looks like the final OK line
			responseUpper := strings.ToUpper(response)
			if strings.Contains(responseUpper, "OK") || len(response) < 10 {
				// Likely the final line
				break
			}
			// Continue reading more extension lines
		}
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
	for _, line := range lines {
		// Remove trailing \r if present (handle both \r\n and \n)
		line = strings.TrimRight(line, "\r")

		// Skip empty lines at the end
		if line == "" && len(lines) > 0 {
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
