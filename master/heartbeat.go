package master

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/logging"
)

func heartbeat(conn net.PacketConn, addr net.Addr, buffer []byte) {
	sessionId := binary.BigEndian.Uint32(buffer[1:5])
	logging.Notice("AVAILABLE", "Received heartbeat from", addr.String())
	values := strings.Split(string(buffer[5:]), "\u0000")

	payload := map[string]string{}
	for i := 0; i < len(values); i += 2 {
		if values[i] == "" {
			break
		}

		payload[values[i]] = values[i+1]
	}

	// Generate challenge and send to server
	var hexIP string
	for _, i := range strings.Split(payload["localip0"], ".") {
		val, err := strconv.ParseUint(i, 10, 64)
		if err != nil {
			panic(err)
		}

		hexIP += fmt.Sprintf("%02X", val)
	}

	port, err := strconv.ParseUint(payload["localport"], 10, 64)
	if err != nil {
		panic(err)
	}

	hexPort := fmt.Sprintf("%04X", port)

	challenge := common.RandomString(6) + "00" + hexIP + hexPort
	mutex.Lock()
	session := sessions[sessionId]
	session.Challenge = challenge
	mutex.Unlock()

	response := []byte{0xfe, 0xfd, 0x01}
	response = binary.BigEndian.AppendUint32(response, sessionId)
	response = append(response, []byte(challenge)...)
	response = append(response, 0)

	conn.WriteTo(response, addr)
}
