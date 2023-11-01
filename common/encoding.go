package common

import (
	"fmt"
	"strconv"
	"strings"
)

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

func IPFormatToInt(ip string) (int32, uint16) {
	port := 0

	if strings.Contains(ip, ":") {
		ipSplit := strings.Split(ip, ":")

		var err error
		port, err = strconv.Atoi(ipSplit[1])
		if err != nil {
			panic(err)
		}

		ip = ipSplit[0]
	}

	var intIP int
	for i, s := range strings.Split(ip, ".") {
		val, err := strconv.Atoi(s)
		if err != nil {
			panic(err)
		}

		intIP |= val << (24 - i*8)
	}

	// TODO: Check if this handles negative numbers properly
	return int32(intIP), uint16(port)
}

func IPFormatToString(ip string) (string, string) {
	intIP, intPort := IPFormatToInt(ip)

	// TODO: Check if this handles negative numbers properly
	return strconv.FormatInt(int64(intIP), 10), strconv.FormatUint(uint64(intPort), 10)
}
