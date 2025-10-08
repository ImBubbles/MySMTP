package string

// Builder StringBuilder
type Builder struct {
	buffer string
}

func (sb *Builder) Append(s string) *Builder {
	sb.buffer += s
	return sb
}

func (sb *Builder) AppendRune(r rune) *Builder {
	sb.buffer += string(r)
	return sb
}

func (sb *Builder) AsString() string {
	return sb.buffer
}

func NewStringBuilder() *Builder {
	return &Builder{""}
}
