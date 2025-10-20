package mail

type Mail struct {
	from    string
	to      []string
	cc      []string
	bcc     []string
	subject string
	data    string
}

func NewBlankMail() *Mail {
	return &Mail{}
}

func (m *Mail) From(from string) *Mail {
	m.from = from
	return m
}

func (m *Mail) Subject(subject string) *Mail {
	m.subject = subject
	return m
}

func (m *Mail) ToAppend(to ...string) *Mail {
	m.to = append(m.to, to...)
	return m
}

func (m *Mail) CCAppend(cc ...string) *Mail {
	m.cc = append(m.cc, cc...)
	return m
}

func (m *Mail) BCCAppend(bcc ...string) *Mail {
	m.bcc = append(m.bcc, bcc...)
	return m
}
