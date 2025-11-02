package mail

import (
	"github.com/ImBubbles/MySMTP/util/smtp"
	"strings"
)

type Header struct {
	key   string
	value string
}

type NamedAddress struct {
	header  Header
	name    string
	address string
}

// GetAddress returns the email address
func (n *NamedAddress) GetAddress() string {
	return n.address
}

// GetName returns the display name
func (n *NamedAddress) GetName() string {
	return n.name
}

func ParseNamedAddress(line string) *[]NamedAddress {
	keyEndIndex := strings.Index(line, ":")
	if keyEndIndex == -1 {
		// Invalid format, return empty slice
		return &[]NamedAddress{}
	}
	key := strings.TrimSpace(line[:keyEndIndex])
	value := strings.TrimSpace(line[keyEndIndex+1:])
	var hasMultiple bool = strings.ContainsRune(value, ',')

	alloc := strings.Count(line, ",") + 1
	if alloc < 1 {
		alloc = 1
	}
	var result []NamedAddress = make([]NamedAddress, alloc)
	if !hasMultiple {
		result[0] = rawToNamedAddress(key, value, value)
		return &result
	}

	for i, entry := range strings.Split(value, ",") {
		result[i] = rawToNamedAddress(entry, key, value)
	}

	return &result

}

func rawToNamedAddress(key string, value string, raw string) NamedAddress {
	name, address := smtp.CleanFromData(value)
	result := NamedAddress{
		header:  Header{key, value},
		name:    name,
		address: address,
	}
	return result
}
