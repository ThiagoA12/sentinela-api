package auth

import "unicode/utf8"

const (
	MinPasswordLength = 15
	MaxPasswordLength = 128
)

func ValidPasswordLength(password string) bool {
	n := utf8.RuneCountInString(password)
	return n >= MinPasswordLength && n <= MaxPasswordLength
}
