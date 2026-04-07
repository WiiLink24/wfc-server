package common

import (
	"encoding/base64"
	"encoding/binary"
	"unicode/utf16"
)

var (
	Base64DwcEncoding                = base64.NewEncoding("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789.-").WithPadding('*')
	Base64GamespyAlternativeEncoding = base64.NewEncoding("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789[]").WithPadding('_')
)

func Base32Encode(value uint64) string {
	alpha := "0123456789abcdefghijklmnopqrstuv"

	encoded := ""
	for value > 0 {
		encoded += string(alpha[value&0x1f])
		value >>= 5
	}

	encoded = reverse(encoded)

	return encoded
}

func Base64Convert(input string, fromEncoding, toEncoding *base64.Encoding) (string, error) {
	decoded, err := fromEncoding.DecodeString(input)
	if err != nil {
		return "", err
	}

	return toEncoding.EncodeToString(decoded), nil
}

func reverse(s string) string {
	rns := []rune(s)
	for i, j := 0, len(rns)-1; i < j; i, j = i+1, j-1 {
		rns[i], rns[j] = rns[j], rns[i]
	}

	return string(rns)
}

func UTF16Encode(s string, order binary.ByteOrder) []byte {
	encoded := utf16.Encode([]rune(s))
	buf := make([]byte, len(encoded)*2)
	for i, v := range encoded {
		order.PutUint16(buf[i*2:], v)
	}
	return buf
}

func UTF16Decode(u []byte, order binary.ByteOrder) string {
	decoded := make([]uint16, len(u)/2)
	for i := range decoded {
		v := order.Uint16(u[i*2:])
		if v == 0 {
			decoded = decoded[:i]
			break
		}
		decoded[i] = v
	}
	return string(utf16.Decode(decoded))
}

func NullTerminatedString(u []byte) string {
	for i, b := range u {
		if b == 0 {
			return string(u[:i])
		}
	}
	return string(u)
}
