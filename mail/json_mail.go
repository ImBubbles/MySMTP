package mail

import (
	"encoding/json"
)

// JSONMail represents a Mail struct in JSON format for easy serialization/deserialization
// This is used to easily convert between JSON (from backend) and Mail struct (for sending)
type JSONMail struct {
	From    string            `json:"from,omitempty"`
	To      []string          `json:"to,omitempty"`
	CC      []string          `json:"cc,omitempty"`
	BCC     []string          `json:"bcc,omitempty"`
	Subject string            `json:"subject,omitempty"`
	Body    string            `json:"body,omitempty"`    // Email body/content
	Headers map[string]string `json:"headers,omitempty"` // Additional custom headers
}

// ToMail converts JSONMail to Mail struct for sending
func (j *JSONMail) ToMail() *Mail {
	mail := NewBlankMail()

	if j.From != "" {
		mail.SetFrom(j.From)
	}

	if len(j.To) > 0 {
		mail.AppendTo(j.To...)
	}

	if len(j.CC) > 0 {
		mail.AppendCC(j.CC...)
	}

	if len(j.BCC) > 0 {
		mail.AppendBCC(j.BCC...)
	}

	if j.Subject != "" {
		mail.SetSubject(j.Subject)
	}

	// Set body as data
	if j.Body != "" {
		mail.SetData(j.Body)
	}

	// Convert custom headers to flags
	for key, value := range j.Headers {
		if key != "" && value != "" {
			flag := NewFlag(key, value)
			mail.AppendFlag(FromFlag(*flag))
		}
	}

	return mail
}

// FromMail converts Mail struct to JSONMail for JSON serialization
func FromMail(m *Mail) *JSONMail {
	jsonMail := &JSONMail{
		From:    m.GetFrom(),
		To:      m.GetTo(),
		CC:      m.GetCC(),
		BCC:     m.GetBCC(),
		Subject: m.GetSubject(),
		Body:    m.GetData(),
		Headers: make(map[string]string),
	}

	// Convert flags to headers
	flags := m.GetFlags()
	for _, flag := range flags {
		key := flag.GetKey()
		value := flag.GetValue()
		if key != "" && value != "" {
			jsonMail.Headers[key] = value
		}
	}

	return jsonMail
}

// FromJSON creates a Mail struct from JSON bytes
func FromJSON(jsonBytes []byte) (*Mail, error) {
	var jsonMail JSONMail
	if err := json.Unmarshal(jsonBytes, &jsonMail); err != nil {
		return nil, err
	}
	return jsonMail.ToMail(), nil
}

// ToJSON converts a Mail struct to JSON bytes
func ToJSON(m *Mail) ([]byte, error) {
	jsonMail := FromMail(m)
	return json.Marshal(jsonMail)
}

// ParseJSONMail parses JSON string to JSONMail struct
func ParseJSONMail(jsonStr string) (*JSONMail, error) {
	var jsonMail JSONMail
	if err := json.Unmarshal([]byte(jsonStr), &jsonMail); err != nil {
		return nil, err
	}
	return &jsonMail, nil
}
