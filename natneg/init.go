package natneg

import (
	"encoding/binary"
	"fmt"
	"net"
	"wwfc/common"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

func (session *NATNEGSession) handleInit(conn net.PacketConn, addr net.Addr, buffer []byte, moduleName string, version byte) {
	if len(buffer) < 10 {
		logging.Error(moduleName, "Invalid packet size")
		return
	}

	portType := buffer[0]
	clientIndex := buffer[1]
	useGamePort := buffer[2]
	localIPBytes := buffer[3:7]
	localPort := binary.BigEndian.Uint16(buffer[7:9])
	gameName, err := common.GetString(buffer[9:])
	if err != nil {
		logging.Error(moduleName, "Invalid gameName")
		return
	}

	expectedSize := 9 + len(gameName) + 1
	if len(buffer) != expectedSize {
		logging.Warn(moduleName, "Stray", aurora.BrightCyan(len(buffer)-expectedSize), "bytes after packet")
	}

	localIPStr := fmt.Sprintf("%d.%d.%d.%d:%d", localIPBytes[0], localIPBytes[1], localIPBytes[2], localIPBytes[3], localPort)

	if portType > 0x03 {
		logging.Error(moduleName, "Invalid port type")
		return
	}
	if useGamePort > 1 {
		logging.Error(moduleName, "Invalid", aurora.BrightGreen("Use Game Port"), "value")
		return
	}
	if useGamePort == 0 && portType == PortTypeGamePort {
		logging.Error(moduleName, "Request uses game port but use game port is disabled")
		return
	}

	// Write the init acknowledgement to the requester address
	ackHeader := createPacketHeader(version, NNInitReply, session.Cookie)
	ackHeader = append(ackHeader, portType, clientIndex)
	ackHeader = append(ackHeader, 0xff, 0xff, 0x6d, 0x16, 0xb5, 0x7d, 0xea)
	conn.WriteTo(ackHeader, addr)

	sender, exists := session.Clients[clientIndex]
	if !exists {
		logging.Notice(moduleName, "Creating client index", aurora.Cyan(clientIndex))

		for _, other := range session.Clients {
			if other.GameName != gameName {
				logging.Error(moduleName, "Game name mismatch", aurora.Cyan(other.GameName), "!=", aurora.Cyan(gameName))
				return
			}
		}

		sender = &NATNEGClient{
			Cookie:          session.Cookie,
			Index:           clientIndex,
			ConnectingIndex: clientIndex,
			Result:          map[byte]byte{},
			NegotiateIP:     "",
			LocalIP:         "",
			ServerIP:        "",
			GameName:        "",
		}
		session.Clients[clientIndex] = sender
	}

	sender.GameName = gameName

	if portType != PortTypeGamePort {
		sender.NegotiateIP = addr.String()
	}
	if localPort != 0 {
		sender.LocalIP = localIPStr
	}
	if useGamePort == 0 || portType == PortTypeGamePort {
		sender.ServerIP = addr.String()
	}

	if !sender.isMapped() {
		return
	}
	// logging.Info(moduleName, "Mapped", aurora.BrightCyan(sender.NegotiateIP), aurora.BrightCyan(sender.LocalIP), aurora.BrightCyan(sender.ServerIP))

	// Send the connect requests
	session.sendConnectRequests(moduleName)
}
