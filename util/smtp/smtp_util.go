package smtp

import (
	"regexp"
	"strings"
)

func CleanEmail(s string) string {
	// Take '<me@email.com>' and make 'me@email.com'
	var result string = ""
	var pre int = strings.Index(s, "<")
	if pre != -1 {
		result = s[pre:]
	}
	var suf int = strings.Index(result, ">")
	if suf != -1 {
		result = s[:suf]
	}
	return result
}

func RemoveAll(s string, regex string) string {
	re := regexp.MustCompile(regex)
	return re.ReplaceAllString(s, "")
}

func CleanFromData(s string) (name, address string) {
	spaceIndex := strings.Index(s, " ")
	var hasName bool = spaceIndex != -1
	name = ""
	if hasName {
		name = RemoveAll(s[:spaceIndex], "\"")
	}
	address = CleanEmail(s[spaceIndex+1:])
	return
}
