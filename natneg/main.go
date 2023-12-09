package natneg

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/logrusorgru/aurora/v3"
	"net"
	"sync"
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

	// Port type
	PortTypeGamePort = 0x00
	PortTypeNATNEG1  = 0x01
	PortTypeNATNEG2  = 0x02
	PortTypeNATNEG3  = 0x03

	// NAT type
	NATTypeNoNat              = 0x00
	NATTypeFirewallOnly       = 0x01
	NATTypeFullCone           = 0x02
	NATTypeRestrictedCone     = 0x03
	NATTypePortRestrictedCone = 0x04
	NATTypeSymmetric          = 0x05
	NATTypeUnknown            = 0x06

	// NAT mapping scheme
	NATMappingUnknown           = 0x00
	NATMappingSamePrivatePublic = 0x01
	NATMappingConsistent        = 0x02
	NATMappingIncremental       = 0x03
	NATMappingMixed             = 0x04
)

type NATNEGSession struct {
	Cookie  uint32
	Mutex   sync.RWMutex
	Clients map[byte]*NATNEGClient
}

type NATNEGClient struct {
	Cookie      uint32
	Connected   bool
	NegotiateIP string
	LocalIP     string
	ServerIP    string
	GameName    string
}

var (
	sessions = map[uint32]*NATNEGSession{}
	mutex    = sync.RWMutex{}
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
		buffer := make([]byte, 1024)
		size, addr, err := conn.ReadFrom(buffer)
		if err != nil {
			continue
		}

		go handleConnection(conn, addr, buffer[:size])
	}
}

func handleConnection(conn net.PacketConn, addr net.Addr, buffer []byte) {
	// Validate the packet magic
	if !bytes.Equal(buffer[:6], []byte{0xfd, 0xfc, 0x1e, 0x66, 0x6a, 0xb2}) {
		logging.Error("NATNEG:"+addr.String(), "Invalid packet header")
		return
	}

	// Parse the NATNEG header
	// fd fc 1e 66 6a b2 - Packet Magic
	// xx                - Version
	// xx                - Packet Type / Command
	// xx xx xx xx       - Cookie

	version := buffer[6]
	command := buffer[7]
	cookie := binary.BigEndian.Uint32(buffer[8:12])

	moduleName := "NATNEG:" + fmt.Sprintf("%08x/", cookie) + addr.String()

	mutex.Lock()
	session, exists := sessions[cookie]
	if !exists {
		// TODO: Figure out removing the session if this request is nonsense or something
		logging.Notice(moduleName, "Creating session")
		session = &NATNEGSession{
			Cookie:  cookie,
			Mutex:   sync.RWMutex{},
			Clients: map[byte]*NATNEGClient{},
		}
		sessions[cookie] = session
	}
	mutex.Unlock()

	session.Mutex.Lock()
	defer session.Mutex.Unlock()

	switch command {
	default:
		logging.Error(moduleName, "Received unknown command type:", aurora.Cyan(command))
		break

	case NNInitRequest:
		logging.Notice(moduleName, "Command:", aurora.Yellow("NNInitRequest"))
		session.handleInit(conn, addr, buffer[12:], moduleName, version)
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
		// TODO: Set the client Connected value to true here
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
		session.handleReport(conn, addr, buffer[12:], moduleName, version)
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

func getPortTypeName(portType byte) string {
	switch portType {
	default:
		return fmt.Sprintf("Unknown (0x%02x)", portType)

	case PortTypeGamePort:
		return "GamePort"

	case PortTypeNATNEG1:
		return "NATNEG1"

	case PortTypeNATNEG2:
		return "NATNEG2"

	case PortTypeNATNEG3:
		return "NATNEG3"
	}
}

func (session *NATNEGSession) handleInit(conn net.PacketConn, addr net.Addr, buffer []byte, moduleName string, version byte) {
	portType := buffer[0]
	clientIndex := buffer[1]
	useGamePort := buffer[2]
	localIPBytes := buffer[3:7]
	localPort := binary.BigEndian.Uint16(buffer[7:9])
	gameName := common.GetString(buffer[9:])

	expectedSize := 9 + len(gameName) + 1
	if len(buffer) != expectedSize {
		logging.Warn(moduleName, "Stray", aurora.BrightCyan(len(buffer)-expectedSize), "bytes after packet")
	}

	localIPStr := fmt.Sprintf("%d.%d.%d.%d:%d", localIPBytes[0], localIPBytes[1], localIPBytes[2], localIPBytes[3], localPort)

	logging.Info(moduleName, "Game Name:", aurora.Cyan(gameName), "Version:", aurora.Cyan(version), "Port Type:", aurora.Yellow(getPortTypeName(portType)), "Client Index:", aurora.Cyan(clientIndex), "Use Game Port:", aurora.Cyan(useGamePort))
	logging.Info(moduleName, "Local IP:", aurora.Cyan(localIPStr))

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
		sender = &NATNEGClient{
			Cookie:      session.Cookie,
			Connected:   false,
			NegotiateIP: "",
			LocalIP:     "",
			ServerIP:    "",
			GameName:    "",
		}
		session.Clients[clientIndex] = sender
	}

	sender.Connected = false
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
	logging.Notice(moduleName, "Mapped", aurora.BrightCyan(sender.NegotiateIP), aurora.BrightCyan(sender.LocalIP), aurora.BrightCyan(sender.ServerIP))

	for id, destination := range session.Clients {
		if id == clientIndex || destination.Connected || !destination.isMapped() {
			continue
		}

		logging.Notice(moduleName, "Exchange connect requests")

		// Send the requests back and forth
		// TODO: Send again if no reply received from client
		sender.sendConnectRequest(conn, destination, version)
		destination.sendConnectRequest(conn, sender, version)
	}
}

func (client *NATNEGClient) isMapped() bool {
	if client.NegotiateIP == "" || client.LocalIP == "" || client.ServerIP == "" {
		return false
	}

	return true
}

func createPacketHeader(version byte, command byte, cookie uint32) []byte {
	header := []byte{0xfd, 0xfc, 0x1e, 0x66, 0x6a, 0xb2, version, command}
	return binary.BigEndian.AppendUint32(header, cookie)
}

func (client *NATNEGClient) sendConnectRequest(conn net.PacketConn, destination *NATNEGClient, version byte) {
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

func (session *NATNEGSession) handleReport(conn net.PacketConn, addr net.Addr, buffer []byte, _ string, version byte) {
	response := createPacketHeader(version, NNReportReply, session.Cookie)
	response = append(response, buffer[:9]...)
	response[14] = 0
	conn.WriteTo(response, addr)
}
