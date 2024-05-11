package natneg

import (
	"net"
	"wwfc/logging"
)

const (
	NNPreInitWaitingForClient  = 0x00
	NNPreInitWaitingForMatchup = 0x01
	NNPreInitReady             = 0x02
)

// typedef struct _PreinitPacket
// {
//     unsigned char clientindex;
//     unsigned char state;
//     int clientID;
// } PreinitPacket;

func (session *NATNEGSession) handlePreinit(conn net.PacketConn, addr net.Addr, buffer []byte, moduleName string, version byte) {
	if len(buffer) < 6 {
		logging.Error(moduleName, "Invalid packet size")
		return
	}

	if len(buffer) > 6 {
		logging.Warn(moduleName, "Stray", len(buffer)-6, "bytes after packet")
	}

	// clientIndex := buffer[0]
	// state := buffer[1]
	// clientID := binary.BigEndian.Uint32(buffer[2:6])

	// Not exactly sure how this is supposed to work.
	// NATNEG internally calls it "ACE queuing", but the games don't seem to take advantage of any potential "queuing" functionality.
	// Hopefully just returning "ready" will cause the games to continue with NATNEG as normal.

	packet := createPacketHeader(version, NNPreInitReply, session.Cookie)
	buffer[1] = NNPreInitReady
	packet = append(packet, buffer[:6]...)
	conn.WriteTo(packet, addr)
}
