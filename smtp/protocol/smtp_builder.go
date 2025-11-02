package protocol

import (
	string2 "github.com/ImBubbles/MySMTP/util/string"
	"strconv"
)

type SMTPBuilder struct {
	buffer *string2.Builder
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

func (b *SMTPBuilder) Message(val string) *SMTPBuilder {
	b.buffer = b.buffer.Append(val)
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
	return &SMTPBuilder{string2.NewStringBuilder()}
}
