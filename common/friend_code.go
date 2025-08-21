package common

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"strings"
)

const (
	fcCRC8 = iota
	fcMD5  = iota
)

var crc8Table = func() [256]byte {
	var table [256]byte
	poly := byte(0x07)

	for i := 0; i < 256; i++ {
		crc := byte(i)

		for j := 0; j < 8; j++ {
			if crc&0x80 != 0 {
				crc = (crc << 1) ^ poly
			} else {
				crc <<= 1
			}
		}

		table[i] = crc
	}

	return table
}()

func crc8(data []byte) byte {
	crc := byte(0)

	for _, b := range data {
		crc = crc8Table[crc^b]
	}

	return crc
}

func getCRCType(gameId string) (crcType byte, reverse bool) {
	reverse = false

	if strings.HasPrefix(gameId, "RSB") || strings.HasPrefix(gameId, "RPB") {
		crcType = fcCRC8
		return
	}

	if strings.HasPrefix(gameId, "HDM") || strings.HasPrefix(gameId, "WDM") {
		crcType = fcCRC8
		reverse = true
		return
	}

	switch gameId[0] {
	case 'R', 'S', 'H', 'W', 'X', 'Y':
		crcType = fcMD5

	default:
		crcType = fcCRC8
	}

	return
}

func CalcFriendCode(pid uint32, gameId string) uint64 {
	if pid == 0 {
		return 0
	}

	buffer := make([]byte, 8)
	binary.LittleEndian.PutUint32(buffer, pid)
	buffer[4] = gameId[3]
	buffer[5] = gameId[2]
	buffer[6] = gameId[1]
	buffer[7] = gameId[0]

	crc, _ := getCRCType(gameId)

	if crc == fcCRC8 {
		return uint64(pid) | (uint64(crc8(buffer)&0x7f) << 32)
	}

	digest := md5.Sum(buffer)
	return uint64(pid) | (uint64(digest[0]&0xfe) << 31)
}

func CalcFriendCodeString(pid uint32, gameId string) string {
	_, reverse := getCRCType(gameId)

	return GetRawFriendCodeString(CalcFriendCode(pid, gameId), reverse)
}

func GetRawFriendCodeString(fc uint64, reverse bool) string {
	s := fmt.Sprintf("%012d", max(min(fc, 999999999999), 0))

	if reverse {
		// Reverse the digit order
		s = s[11:12] + s[10:11] + s[9:10] + s[8:9] + s[7:8] + s[6:7] + s[5:6] + s[4:5] + s[3:4] + s[2:3] + s[1:2] + s[0:1]
	}

	return s[len(s)-12:len(s)-8] + "-" + s[len(s)-8:len(s)-4] + "-" + s[len(s)-4:]
}
