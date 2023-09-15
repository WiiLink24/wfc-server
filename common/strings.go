package common

import (
	"bytes"
	"math/rand"
	"time"
)

var letterRunes = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandomString(n int) string {
	rand.Seed(time.Now().UnixNano())

	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

var hexRunes = []rune("0123456789abcdefabcdef")

func RandomHexString(n int) string {
	rand.Seed(time.Now().UnixNano())

	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(hexRunes))]
	}
	return string(b)
}

func GetString(buf []byte) string {
	nullTerminator := bytes.IndexByte(buf, 0)
	return string(buf[:nullTerminator])
}
