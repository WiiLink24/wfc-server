package common

import (
	"strconv"
	"strings"
)

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

	return int32(intIP), uint16(port)
}

func IPFormatNoPortToInt(ip string) int32 {
	intIP, _ := IPFormatToInt(ip)

	return intIP
}

func IPFormatToString(ip string) (string, string) {
	intIP, intPort := IPFormatToInt(ip)

	return strconv.FormatInt(int64(intIP), 10), strconv.FormatUint(uint64(intPort), 10)
}

func IPFormatToStringLE(ip string) (string, string) {
	intIP, intPort := IPFormatToInt(ip)

	// Convert to little endian and print as big endian int
	intIP = int32((uint32(intIP) >> 24) | ((uint32(intIP) & 0x00FF0000) >> 8) | ((uint32(intIP) & 0x0000FF00) << 8) | ((uint32(intIP) & 0x000000FF) << 24))
	return strconv.FormatInt(int64(intIP), 10), strconv.FormatUint(uint64(intPort), 10)
}

func IPFormatBytes(ip string) []byte {
	if strings.Contains(ip, ":") {
		ip = strings.Split(ip, ":")[0]
	}

	var bytes []byte
	for _, s := range strings.Split(ip, ".") {
		val, err := strconv.Atoi(s)
		if err != nil {
			panic(err)
		}

		bytes = append(bytes, byte(val))
	}

	return bytes
}

var (
	reservedIPList = []struct {
		ip   int32
		mask int32
	}{
		{IPFormatNoPortToInt("0.0.0.0"), 8},       // RFC1122 "This host on this network"
		{IPFormatNoPortToInt("10.0.0.0"), 8},      // RFC1918 Private-Use
		{IPFormatNoPortToInt("100.64.0.0"), 10},   // RFC6598 Shared Address Space
		{IPFormatNoPortToInt("127.0.0.0"), 8},     // RFC1122 Loopback
		{IPFormatNoPortToInt("169.254.0.0"), 16},  // RFC3927 Link-Local
		{IPFormatNoPortToInt("172.16.0.0"), 12},   // RFC1918 Private-Use
		{IPFormatNoPortToInt("192.0.0.0"), 24},    // RFC6890 IETF Protocol Assignments
		{IPFormatNoPortToInt("192.0.2.0"), 24},    // RFC5737 Documentation (TEST-NET-1)
		{IPFormatNoPortToInt("192.31.196.0"), 24}, // RFC7535 AS112-v4
		{IPFormatNoPortToInt("192.52.193.0"), 24}, // RFC7450 AMT
		{IPFormatNoPortToInt("192.88.99.0"), 24},  // RFC7526 6to4 Relay Anycast
		{IPFormatNoPortToInt("192.168.0.0"), 16},  // RFC1918 Private-Use
		{IPFormatNoPortToInt("192.175.48.0"), 24}, // RFC7534 Direct Delegation AS112 Service
		{IPFormatNoPortToInt("198.18.0.0"), 15},   // RFC2544 Benchmarking
		{IPFormatNoPortToInt("198.51.100.0"), 24}, // RFC5737 Documentation (TEST-NET-2)
		{IPFormatNoPortToInt("203.0.113.0"), 24},  // RFC5737 Documentation (TEST-NET-3)
		{IPFormatNoPortToInt("224.0.0.0"), 4},     // RFC1112 Multicast
		{IPFormatNoPortToInt("240.0.0.0"), 4},     // RFC1112 Reserved for Future Use + RFC919 Limited Broadcast
	}
)

// TODO: Test this
func IsReservedIP(ip int32) bool {
	for _, reserved := range reservedIPList {
		rMask := 32 - reserved.mask
		if ip>>rMask == reserved.ip>>rMask {
			return true
		}
	}

	return false
}
