package common

import (
	"bytes"
	"errors"
	"math/rand"
)

var letterRunes = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

var hexRunes = []rune("0123456789abcdefabcdef")

func RandomHexString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(hexRunes))]
	}
	return string(b)
}

func GetString(buf []byte) (string, error) {
	nullTerminator := bytes.IndexByte(buf, 0)

	if nullTerminator == -1 {
		return "", errors.New("buf is not null-terminated")
	}

	return string(buf[:nullTerminator]), nil
}

// IsUppercaseAlphanumeric checks if the given string is composed exclusively of uppercase alphanumeric characters.
func IsUppercaseAlphanumeric(str string) bool {
	strLength := len(str)

	if strLength == 0 {
		return false
	}

	for i := 0; i < strLength; i++ {
		c := str[i]

		if (c < '0' || c > '9') && (c < 'A' || c > 'Z') {
			return false
		}
	}

	return true
}

func StringInSlice(str string, slice []string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
