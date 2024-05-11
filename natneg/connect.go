package natneg

import (
	"encoding/binary"
	"net"
	"time"
	"wwfc/common"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

func (session *NATNEGSession) sendConnectRequests(moduleName string) {
	for id, sender := range session.Clients {
		if !sender.isMapped() || sender.ConnectingIndex != id {
			continue
		}

		for destID, destination := range session.Clients {
			if id == destID || !destination.isMapped() || destination.ConnectingIndex != destID {
				continue
			}

			if _, hasResult := destination.Result[id]; hasResult {
				continue
			}

			logging.Notice(moduleName, "Exchange connect requests between", aurora.BrightCyan(id), "and", aurora.BrightCyan(destID))
			sender.ConnectingIndex = destID
			sender.ConnectAck = false
			destination.ConnectingIndex = id
			destination.ConnectAck = false

			go func(session *NATNEGSession, sender *NATNEGClient, destination *NATNEGClient) {
				for {
					if !session.Open {
						return
					}

					check := false

					if !destination.ConnectAck && destination.ConnectingIndex == sender.Index {
						check = true
						sender.sendConnectRequestPacket(natnegConn, destination, session.Version)
					}

					if !sender.ConnectAck && sender.ConnectingIndex == destination.Index {
						check = true
						destination.sendConnectRequestPacket(natnegConn, sender, session.Version)
					}

					if !check {
						return
					}

					time.Sleep(500 * time.Millisecond)
				}
			}(session, sender, destination)
		}
	}
}

func (client *NATNEGClient) sendConnectRequestPacket(conn net.PacketConn, destination *NATNEGClient, version byte) {
	connectHeader := createPacketHeader(version, NNConnectRequest, destination.Cookie)
	connectHeader = append(connectHeader, common.IPFormatBytes(client.ServerIP)...)
	_, port := common.IPFormatToInt(client.ServerIP)
	connectHeader = binary.BigEndian.AppendUint16(connectHeader, port)
	// Two bytes: "gotyourdata" and "finished"
	connectHeader = append(connectHeader, 0x42, 0x00)

	destIPAddr, err := net.ResolveUDPAddr("udp", destination.NegotiateIP)
	if err != nil {
		panic(err)
	}
	conn.WriteTo(connectHeader, destIPAddr)
}

func (session *NATNEGSession) handleConnectReply(conn net.PacketConn, addr net.Addr, buffer []byte, moduleName string, version byte) {
	// portType := buffer[0]
	clientIndex := buffer[1]
	// useGamePort := buffer[2]
	// localIPBytes := buffer[3:7]

	if client, exists := session.Clients[clientIndex]; exists {
		client.ConnectAck = true
	}
}
