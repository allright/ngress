package utils

import "strings"

func GetStringValue(key string, str string) string {
	values := strings.Split(str, ",")
	for _, val := range values {
		s := strings.Split(val, "=")
		if len(s) != 2 || s[0] != key {
			continue
		}
		return s[1]
	}
	return ""
}
