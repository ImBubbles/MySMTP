package string

import "strings"

func FirstWord(s string) string {
	spaceIndex := strings.Index(s, " ")
	if spaceIndex == -1 {
		return s
	}
	return s[:spaceIndex]
}
