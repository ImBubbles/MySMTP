package smtp

import (
	"MySMTP/mail"
	"MySMTP/smtp/protocol"
	"bufio"
	"net"
)

// TODO: everything and goon
type ClientConn struct {
	conn   net.Conn
	state  protocol.SMTPStates
	reader *bufio.Reader
	mail   mail.Mail
}

func NewClientConn(conn net.Conn, mail mail.Mail) *ClientConn {
	clientConn := &ClientConn{
		conn:   conn,
		state:  protocol.STATE_EHLO,
		reader: bufio.NewReader(conn),
		mail:   mail}
	clientConn.handle()
	return clientConn
}

func (c *ClientConn) handle() {

}
