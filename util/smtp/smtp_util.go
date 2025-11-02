package smtp

import (
	"regexp"
	"strings"
)

func CleanEmail(s string) string {
	// Take '<me@email.com>' and make 'me@email.com'
	pre := strings.Index(s, "<")
	if pre == -1 {
		// No opening bracket, return as is or empty
		return s
	}
	suf := strings.Index(s[pre:], ">")
	if suf == -1 {
		// No closing bracket, return as is
		return s
	}
	// Extract the email address between < and >
	result := s[pre+1 : pre+suf]
	return strings.TrimSpace(result)
}

func RemoveAll(s string, regex string) string {
	re := regexp.MustCompile(regex)
	return re.ReplaceAllString(s, "")
}

func CleanFromData(s string) (name, address string) {
	spaceIndex := strings.Index(s, " ")
	name = ""
	if spaceIndex != -1 {
		// Has name part
		name = RemoveAll(s[:spaceIndex], "\"")
		name = strings.TrimSpace(name)
		address = CleanEmail(s[spaceIndex+1:])
	} else {
		// No name, just address
		address = CleanEmail(s)
	}
	return
}
