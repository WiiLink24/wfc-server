package master

import (
	"encoding/binary"
	"net"
	"strings"
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

	publicip, ok := payload["publicip"]
	if ok && publicip != "0" {
		return
	}

	sendChallenge(conn, addr, sessionId)
}
