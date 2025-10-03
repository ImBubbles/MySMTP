package util

import (
	"encoding/base64"
	"fmt"
)

func String64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func _64String(s string) (string, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		fmt.Println("Error decoding base64 string")
		return "", err
	}
	decoded := string(decodedBytes)
	return decoded, err
}
