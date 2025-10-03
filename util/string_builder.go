package util

type StringBuilder struct {
	buffer string
}

func (sb *StringBuilder) Append(s string) *StringBuilder {
	sb.buffer += s
	return sb
}

func (sb *StringBuilder) AppendRune(r rune) *StringBuilder {
	sb.buffer += string(r)
	return sb
}

func (sb *StringBuilder) AsString() string {
	return sb.buffer
}

func NewStringBuilder() *StringBuilder {
	return &StringBuilder{""}
}
