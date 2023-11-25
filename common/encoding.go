package common

import (
	"encoding/base64"
	"fmt"
)

var Base64DwcEncoding = base64.NewEncoding("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789.-").WithPadding('*')

func Base32Encode(value int64) string {
	alpha := "0123456789abcdefghijklmnopqrstuv"

	encoded := ""
	for value > 0 {
		encoded += string(alpha[value&0x1f])
		value >>= 5

		fmt.Sprintf("%0.9s", encoded)
	}

	encoded = reverse(encoded)

	return encoded
}

func reverse(s string) string {
	rns := []rune(s)
	for i, j := 0, len(rns)-1; i < j; i, j = i+1, j-1 {
		rns[i], rns[j] = rns[j], rns[i]
	}

	return string(rns)
}
