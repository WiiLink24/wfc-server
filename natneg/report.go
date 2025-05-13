package natneg

import (
	"fmt"
	"net"
	"wwfc/logging"
	"wwfc/qr2"

	"github.com/logrusorgru/aurora/v3"
)

func (session *NATNEGSession) handleReport(conn net.PacketConn, addr net.Addr, buffer []byte, _moduleName string, version byte) {
	if len(buffer) < 2 {
		logging.Error(_moduleName, "Invalid packet size")
		return
	}

	response := createPacketHeader(version, NNReportReply, session.Cookie)
	response = append(response, buffer[:9]...)
	response[14] = 0
	conn.WriteTo(response, addr)

	// portType := buffer[0]
	clientIndex := buffer[1]
	result := buffer[2]
	// natType := buffer[3]
	// mappingScheme := buffer[7]
	// gameName, err := common.GetString(buffer[11:])

	moduleName := "NATNEG:" + fmt.Sprintf("%08x/", session.Cookie) + addr.String()
	logging.Notice(moduleName, "Report from", aurora.BrightCyan(clientIndex), "result:", aurora.Cyan(result))

	if client, exists := session.Clients[clientIndex]; exists {
		client.Result[client.ConnectingIndex] = result
		connecting := session.Clients[client.ConnectingIndex]
		client.ConnectingIndex = clientIndex
		client.ConnectAck = false

		if otherResult, hasResult := connecting.Result[clientIndex]; hasResult {
			if otherResult != 1 {
				result = otherResult
			}
			qr2.ProcessNATNEGReport(result, client.ServerIP, connecting.ServerIP)
		}
	}

	// Send remaining requests
	session.sendConnectRequests(moduleName)
}
