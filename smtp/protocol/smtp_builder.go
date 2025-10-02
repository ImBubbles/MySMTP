package protocol

import (
	"MySMTP/util"
	"strconv"
)

type SMTPBuilder struct {
	buffer *util.StringBuilder
}

func NewSMTPBuilder() *SMTPBuilder {
	return &SMTPBuilder{util.NewStringBuilder()}
}

func Code(b *SMTPBuilder, val SMTPCode, hyphen bool) *SMTPBuilder {
	b.buffer = b.buffer.Append(strconv.Itoa(int(val)))
	if hyphen {
		b.buffer = b.buffer.AppendRune('-')
	} else {
		b.buffer = b.buffer.AppendRune(' ')
	}
	return b
}

func Command(b *SMTPBuilder, val SMTPCommands) *SMTPBuilder {
	b.buffer = b.buffer.Append(string(val))
	return b
}

func Body(b *SMTPBuilder, val SMTPBody) *SMTPBuilder {
	b.buffer = b.buffer.Append(string(val))
	return b
}
