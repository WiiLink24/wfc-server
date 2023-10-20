package matchmaking

import (
	"encoding/binary"
	"github.com/logrusorgru/aurora/v3"
	"net"
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/logging"
)

func popString(buffer []byte, index int) (string, int) {
	str := common.GetString(buffer[index:])
	return str, index + len(str) + 1
}

func popBytes(buffer []byte, index int, size int) ([]byte, int) {
	return buffer[index : index+size], index + size
}

func popUint32(buffer []byte, index int) (uint32, int) {
	return binary.BigEndian.Uint32(buffer[index:]), index + 4
}

func serverList(conn net.Conn, buffer []byte) {
	const (
		FlagNoServerList      = 1 << 1 // 0x02
		FlagAlternateSourceIP = 1 << 3 // 0x08
		FlagLimitResultCount  = 1 << 7 // 0x80
	)

	index := 9
	queryGame, index := popString(buffer, index)
	gameName, index := popString(buffer, index)
	challenge, index := popBytes(buffer, index, 8)
	filter, index := popString(buffer, index)
	fields, index := popString(buffer, index)
	flags, index := popUint32(buffer, index)

	logging.Notice(ModuleName, "queryGame:", aurora.Cyan(queryGame).String(), "- gameName:", aurora.Cyan(gameName).String(), "- filter:", aurora.Cyan(filter).String(), "- fields:", aurora.Cyan(fields).String())

	var output []byte
	for _, s := range strings.Split(strings.Split(conn.RemoteAddr().String(), ":")[0], ".") {
		val, err := strconv.Atoi(s)
		if err != nil {
			panic(err)
		}

		output = append(output, byte(val))
	}

	var fieldList []string
	for _, field := range strings.Split(fields, "\\") {
		if len(field) == 0 || field == " " {
			continue
		}

		fieldList = append(fieldList, field)
	}

	if flags&FlagNoServerList != 0 || len(fieldList) == 0 {
		// The client requests its own public IP and game port
		logging.Notice(ModuleName, "Reply without server list", aurora.Cyan(conn.RemoteAddr()).String())

		// The default game port 6500
		output = binary.BigEndian.AppendUint16(output, 6500)

		// Write the encrypted reply
		conn.Write(common.EncryptTypeX([]byte("9r3Rmy"), challenge, output))
		return
	}

	logging.Notice(ModuleName, "Reply with server list", aurora.Cyan(conn.RemoteAddr()).String())

	// The client's port
	port, err := strconv.Atoi(strings.Split(conn.RemoteAddr().String(), ":")[1])
	if err != nil {
		panic(err)
	}
	output = binary.BigEndian.AppendUint16(output, uint16(port))

	output = append(output, byte(len(fieldList)))
	for _, field := range fieldList {
		output = append(output, 0x00) // Value?
		output = append(output, []byte(field)...)
		output = append(output, 0x00) // String end
	}
	output = append(output, 0x00) // Zero length string to end the list

	// TODO: Send server list here

	// Server with no flags and -1 tells the client the message has ended
	output = append(output, []byte{0x00, 0xff, 0xff, 0xff, 0xff}...)

	// Write the encrypted reply
	conn.Write(common.EncryptTypeX([]byte("9r3Rmy"), challenge, output))
}
