package protocol

import (
	"MySMTP/util"
	"strconv"
)

type SMTPBuilder struct {
	buffer *util.StringBuilder
}

func (b *SMTPBuilder) CodeHyphen(val SMTPCode, hyphen bool) *SMTPBuilder {
	b.buffer = b.buffer.Append(strconv.Itoa(int(val)))
	if hyphen {
		b.buffer = b.buffer.AppendRune('-')
	} else {
		b.buffer = b.buffer.AppendRune(' ')
	}
	return b
}

func (b *SMTPBuilder) Code(val SMTPCode) *SMTPBuilder {
	return b.CodeHyphen(val, false)
}

func (b *SMTPBuilder) Command(val SMTPCommands) *SMTPBuilder {
	b.buffer = b.buffer.Append(string(val))
	return b
}

func (b *SMTPBuilder) Message(val SMTPBody) *SMTPBuilder {
	b.buffer = b.buffer.Append(string(val))
	return b
}

func (b *SMTPBuilder) Domain() *SMTPBuilder {
	// TODO env file
	b.buffer = b.buffer.Append("domain.com")
	return b
}

func (b *SMTPBuilder) Get() string {
	return b.buffer.AsString()
}

func NewSMTPBuilder() *SMTPBuilder {
	return &SMTPBuilder{util.NewStringBuilder()}
}
