package mail

type Mail struct {
	from    string
	flags   []FromFlag
	to      []string
	cc      []string
	bcc     []string
	subject string
	data    string
}

func NewBlankMail() *Mail {
	return &Mail{}
}

func (m *Mail) SetFrom(from string) *Mail {
	m.from = from
	return m
}

func (m *Mail) SetSubject(subject string) *Mail {
	m.subject = subject
	return m
}

func (m *Mail) AppendTo(to ...string) *Mail {
	m.to = append(m.to, to...)
	return m
}

func (m *Mail) AppendCC(cc ...string) *Mail {
	m.cc = append(m.cc, cc...)
	return m
}

func (m *Mail) AppendBCC(bcc ...string) *Mail {
	m.bcc = append(m.bcc, bcc...)
	return m
}

func (m *Mail) AppendFlag(flags ...FromFlag) *Mail {
	m.flags = append(m.flags, flags...)
	return m
}

func (m *Mail) SetData(data string) *Mail {
	m.data = data
	return m
}

// Getter methods
func (m *Mail) GetFrom() string {
	return m.from
}

func (m *Mail) GetTo() []string {
	return m.to
}

func (m *Mail) GetCC() []string {
	return m.cc
}

func (m *Mail) GetBCC() []string {
	return m.bcc
}

func (m *Mail) GetSubject() string {
	return m.subject
}

func (m *Mail) GetData() string {
	return m.data
}

func (m *Mail) GetFlags() []FromFlag {
	return m.flags
}