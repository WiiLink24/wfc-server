package matchmaking

import (
	"encoding/binary"
	"net"
	"strings"
	"wwfc/common"
	"wwfc/logging"
)

func serverList(conn net.Conn, buffer []byte) {
	logging.Notice(ModuleName, "Received server list command")
	// TODO: Make a custom decoder for this? Go's binary decoder does not support strings as they are not a fixed width.
	//listVersion := buffer[3]
	//encodingVersion := buffer[4]
	//gameVersion := binary.BigEndian.Uint32(buffer[5:])

	index := 9
	queryGame := common.GetString(buffer[index:])
	index += len(queryGame) + 1
	gameName := common.GetString(buffer[index:])
	index += len(gameName) + 1

	challenge := buffer[index : index+8]
	index += 8

	filter := common.GetString(buffer[index:])
	index += len(filter) + 1
	fields := common.GetString(buffer[index:])
	index += len(fields) + 1

	options := binary.BigEndian.Uint32(buffer[index:])
	index += 4

	logging.Notice(ModuleName, "Values", queryGame, gameName, string(challenge), string(options))

	// TODO: Find a game if possible, but there is nobody to do that with yet!
	output := []byte(strings.Replace(conn.RemoteAddr().String(), ".", "", -1))
	output = binary.BigEndian.AppendUint16(output, 6500)

	// encrypted := common.EncryptTypeX([]byte("9r3Rmy"), challenge, output)
	conn.Write(output)
}
