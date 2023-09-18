package common

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
)

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

	digest := md5.Sum(buffer)
	return uint64(pid) | (uint64(digest[0]&0xfe) << 31)
}

func CalcFriendCodeString(pid uint32, gameId string) string {
	return GetFriendCodeString(CalcFriendCode(pid, gameId))
}

func GetFriendCodeString(fc uint64) string {
	s := fmt.Sprintf("%012d", fc)
	return s[len(s)-12:len(s)-8] + "-" + s[len(s)-8:len(s)-4] + "-" + s[len(s)-4:]
}
