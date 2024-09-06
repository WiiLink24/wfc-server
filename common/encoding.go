package common

import (
	"encoding/base64"
	"errors"
	"strings"
)

type GameSpyBase64Encoding int

const (
	GameSpyBase64EncodingDefault   = iota // 0
	GameSpyBase64EncodingAlternate        // 1
	GameSpyBase64EncodingURLSafe          // 2
)

var Base64DwcEncoding = base64.NewEncoding("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789.-").WithPadding('*')

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

func DecodeGameSpyBase64(gameSpyBase64 string, gameSpyBase64Encoding GameSpyBase64Encoding) ([]byte, error) {
	base64String, err := GameSpyBase64ToBase64(gameSpyBase64, gameSpyBase64Encoding)
	if err != nil {
		return nil, err
	}

	return base64.StdEncoding.DecodeString(base64String)
}

func GameSpyBase64ToBase64(gameSpyBase64 string, gameSpyBase64Encoding GameSpyBase64Encoding) (string, error) {
	switch gameSpyBase64Encoding {
	case GameSpyBase64EncodingDefault:
		return gameSpyBase64, nil

	case GameSpyBase64EncodingAlternate:
		return strings.NewReplacer("[", "+", "]", "/", "_", "=").Replace(gameSpyBase64), nil

	case GameSpyBase64EncodingURLSafe:
		return strings.NewReplacer("-", "+", "_", "/" /*, "=", "="*/).Replace(gameSpyBase64), nil

	default:
		return "", errors.New("invalid GameSpy Base64 encoding specified")
	}
}

func reverse(s string) string {
	rns := []rune(s)
	for i, j := 0, len(rns)-1; i < j; i, j = i+1, j-1 {
		rns[i], rns[j] = rns[j], rns[i]
	}

	return string(rns)
}
