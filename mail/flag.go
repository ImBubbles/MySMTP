package mail

type Flag struct {
	key   string
	value string
}

func NewFlag(key string, value string) *Flag {
	return &Flag{string(key), value}
}
