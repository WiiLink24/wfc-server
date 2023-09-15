package master

import (
	"encoding/binary"
	"net"
)

type Session struct {
	SessionID   uint32
	Challenge   string
	SecretKey   string
	IsConnected bool
}

func addSession(addr net.Addr, buffer []byte) {
	sessionId := binary.BigEndian.Uint32(buffer[1:5])

	mutex.Lock()
	if _, ok := sessions[sessionId]; !ok {
		sessions[sessionId] = &Session{
			SessionID: sessionId,
			Challenge: "",
			// TODO: This is hardcoded for Mario Kart Wii
			SecretKey:   "9r3Rmy",
			IsConnected: true,
		}
	}
	mutex.Unlock()
}
