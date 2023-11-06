package natneg

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/logrusorgru/aurora/v3"
	"net"
	"wwfc/common"
	"wwfc/logging"
)

const (
	NNInitRequest         = 0x00
	NNInitReply           = 0x01
	NNErtTestRequest      = 0x02
	NNErtTestReply        = 0x03
	NNStateUpdate         = 0x04
	NNConnectRequest      = 0x05
	NNConnectReply        = 0x06
	NNConnectPing         = 0x07
	NNBackupTestRequest   = 0x08
	NNBackupTestReply     = 0x09
	NNAddressCheckRequest = 0x0A
	NNAddressCheckReply   = 0x0B
	NNNatifyRequest       = 0x0C
	NNReportRequest       = 0x0D
	NNReportReply         = 0x0E
	NNPreInitRequest      = 0x0F
	NNPreInitReply        = 0x10
)

func StartServer() {
	// Get config
	config := common.GetConfig()

	address := config.Address + ":27901"
	conn, err := net.ListenPacket("udp", address)
	if err != nil {
		panic(err)
	}

	// Close the listener when the application closes.
	defer conn.Close()
	logging.Notice("NATNEG", "Listening on", address)

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
	// Validate the packet magic
	if bytes.Equal(buffer[:6], []byte{0xfd, 0xfc, 0x1e, 0x66, 0x6a, 0xb2}) {
		logging.Error("NATNEG:"+addr.String(), "Invalid packet header")
		return
	}

	// Parse the NATNEG header
	// fd fc 1e 66 6a b2 - Packet Magic
	// xx                - Version
	// xx                - Packet Type / Command
	// xx xx xx xx       - Cookie

	// version := buffer[6]
	command := buffer[7]
	cookie := binary.BigEndian.Uint32(buffer[8:12])

	moduleName := "NATNEG:" + fmt.Sprintf("%08x", cookie) + addr.String() + ":"

	switch command {
	default:
		logging.Error(moduleName, "Received unknown command type:", aurora.Cyan(command))

	case NNInitRequest:
		logging.Notice(moduleName, "Command:", aurora.Yellow("NNInitRequest"))
		break

	case NNInitReply:
		logging.Warn(moduleName, "Received server command:", aurora.Yellow("NNInitReply"))
		break

	case NNErtTestRequest:
		logging.Warn(moduleName, "Received server command:", aurora.Yellow("NNErtTestRequest"))
		break

	case NNErtTestReply:
		logging.Notice(moduleName, "Command:", aurora.Yellow("NNErtReply"))
		break

	case NNStateUpdate:
		logging.Notice(moduleName, "Command:", aurora.Yellow("NNStateUpdate"))
		break

	case NNConnectRequest:
		logging.Warn(moduleName, "Received server command:", aurora.Yellow("NNConnectRequest"))
		break

	case NNConnectReply:
		logging.Notice(moduleName, "Command:", aurora.Yellow("NNConnectReply"))
		break

	case NNConnectPing:
		logging.Notice(moduleName, "Command:", aurora.Yellow("NNConnectPing"))
		break

	case NNBackupTestRequest:
		logging.Notice(moduleName, "Command:", aurora.Yellow("NNBackupTestRequest"))
		break

	case NNBackupTestReply:
		logging.Warn(moduleName, "Received server command:", aurora.Yellow("NNBackupTestReply"))
		break

	case NNAddressCheckRequest:
		logging.Notice(moduleName, "Command:", aurora.Yellow("NNAddressCheckRequest"))
		break

	case NNAddressCheckReply:
		logging.Warn(moduleName, "Received server command:", aurora.Yellow("NNAddressCheckReply"))
		break

	case NNNatifyRequest:
		logging.Notice(moduleName, "Command:", aurora.Yellow("NNNatifyRequest"))
		break

	case NNReportRequest:
		logging.Notice(moduleName, "Command:", aurora.Yellow("NNReportRequest"))
		break

	case NNReportReply:
		logging.Warn(moduleName, "Received server command:", aurora.Yellow("NNReportReply"))
		break

	case NNPreInitRequest:
		logging.Notice(moduleName, "Command:", aurora.Yellow("NNPreInitRequest"))
		break

	case NNPreInitReply:
		logging.Warn(moduleName, "Received server command:", aurora.Yellow("NNPreInitReply"))
		break
	}
}
