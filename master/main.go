package master

import (
	"encoding/binary"
	"log"
	"net"
	"sync"
)

var (
	// I would use a sync.Map instead of the map mutex combo, but this performs better.
	sessions = map[uint32]*Session{}
	mutex    = sync.RWMutex{}
)

func StartServer() {
	conn, err := net.ListenPacket("udp", ":27900")
	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close()

	for {
		buf := make([]byte, 1024)
		_, addr, err := conn.ReadFrom(buf)
		if err != nil {
			continue
		}

		go handleConnection(conn, addr, buf)
	}
}

func handleConnection(conn net.PacketConn, addr net.Addr, buffer []byte) {
	if buffer[0] != 9 {
		addSession(addr, buffer)
	}

	switch buffer[0] {
	case 1:
		// Challenge
		sessionId := binary.BigEndian.Uint32(buffer[1:5])
		response := []byte{0xfe, 0xfd, 0x0a}
		response = binary.BigEndian.AppendUint32(response, sessionId)
		conn.WriteTo(response, addr)
		break
	case 3:
		heartbeat(conn, addr, buffer)
		break
	case 9:
		conn.WriteTo([]byte{0xfe, 0xfd, 0x09, 0x00, 0x00, 0x00, 0x00}, addr)
		break
	default:
		return
	}
}
