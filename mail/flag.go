package mail

type Flag struct {
	key   string
	value string
}

// FromFlag is an alias for Flag for SMTP FROM flags
type FromFlag Flag

func NewFlag(key string, value string) *Flag {
	return &Flag{string(key), value}
}

// GetKey returns the flag key
func (f *Flag) GetKey() string {
	return f.key
}

// GetValue returns the flag value
func (f *Flag) GetValue() string {
	return f.value
}

// GetKey returns the FromFlag key
func (f *FromFlag) GetKey() string {
	return f.key
}

// GetValue returns the FromFlag value
func (f *FromFlag) GetValue() string {
	return f.value
}
