package matchmaking

import (
	"encoding/binary"
	"fmt"
	"github.com/logrusorgru/aurora/v3"
	"net"
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/logging"
	"wwfc/qr2"
)

const (
	// Server flags
	UnsolicitedUDPFlag         = 1 << 0 // 0x01 / 1
	PrivateIPFlag              = 1 << 1 // 0x02 / 2
	ConnectNegotiateFlag       = 1 << 2 // 0x04 / 4
	ICMPIPFlag                 = 1 << 3 // 0x08 / 8
	NonstandardPortFlag        = 1 << 4 // 0x10 / 16
	NonstandardPrivatePortFlag = 1 << 5 // 0x20 / 32
	HasKeysFlag                = 1 << 6 // 0x40 / 64
	HasFullRulesFlag           = 1 << 7 // 0x80 / 128

	// Key Type list
	KeyTypeString = 0x00
	KeyTypeByte   = 0x01
	KeyTypeShort  = 0x02

	// Options for ServerListRequest
	SendFieldsForAllOption  = 1 << 0 // 0x01 / 1
	NoServerListOption      = 1 << 1 // 0x02 / 2
	PushUpdatesOption       = 1 << 2 // 0x04 / 4
	AlternateSourceIPOption = 1 << 3 // 0x08 / 8
	SendGroupsOption        = 1 << 5 // 0x20 / 32
	NoListCacheOption       = 1 << 6 // 0x40 / 64
	LimitResultCountOption  = 1 << 7 // 0x80 / 128
)

func FindServers(queryGame string, filter string) ([]map[string]string, error) {
	// TODO: Handle gueryGame, filter
	return FilterServers(qr2.GetSessionServers(), queryGame, filter), nil
}

func popString(buffer []byte, index int) (string, int) {
	str := common.GetString(buffer[index:])
	return str, index + len(str) + 1
}

func popBytes(buffer []byte, index int, size int) ([]byte, int) {
	return buffer[index : index+size], index + size
}

func popUint32(buffer []byte, index int) (uint32, int) {
	return binary.BigEndian.Uint32(buffer[index:]), index + 4
}

func handleServerListRequest(conn net.Conn, buffer []byte) {
	index := 9
	queryGame, index := popString(buffer, index)
	gameName, index := popString(buffer, index)
	challenge, index := popBytes(buffer, index, 8)
	filter, index := popString(buffer, index)
	fields, index := popString(buffer, index)
	options, index := popUint32(buffer, index)

	logging.Info(ModuleName, "queryGame:", aurora.Cyan(queryGame).String(), "- gameName:", aurora.Cyan(gameName).String(), "- filter:", aurora.Cyan(filter).String(), "- fields:", aurora.Cyan(fields).String())

	var output []byte
	for _, s := range strings.Split(strings.Split(conn.RemoteAddr().String(), ":")[0], ".") {
		val, err := strconv.Atoi(s)
		if err != nil {
			panic(err)
		}

		output = append(output, byte(val))
	}

	var fieldList []string
	for _, field := range strings.Split(fields, "\\") {
		if len(field) == 0 || field == " " {
			continue
		}

		fieldList = append(fieldList, field)
	}

	if options&NoServerListOption != 0 || len(fieldList) == 0 {
		// The client requests its own public IP and game port
		logging.Info(ModuleName, "Reply without server list", aurora.Cyan(conn.RemoteAddr()))

		// The default game port 6500
		output = binary.BigEndian.AppendUint16(output, 6500)

		// Write the encrypted reply
		conn.Write(common.EncryptTypeX([]byte("9r3Rmy"), challenge, output))
		return
	}

	logging.Info(ModuleName, "Reply with server list", aurora.Cyan(conn.RemoteAddr()))

	// The client's port
	port, err := strconv.Atoi(strings.Split(conn.RemoteAddr().String(), ":")[1])
	if err != nil {
		panic(err)
	}
	output = binary.BigEndian.AppendUint16(output, uint16(port))

	output = append(output, byte(len(fieldList)))
	for _, field := range fieldList {
		output = append(output, 0x00)             // Key type (0 = string, 1 = byte, 2 = short)
		output = append(output, []byte(field)...) // String
		output = append(output, 0x00)             // String terminator
	}
	output = append(output, 0x00) // Zero length string to end the list

	servers, err := FindServers(queryGame, filter)
	if err != nil {
		panic(err)
	}

	for _, server := range servers {
		var flags byte
		var flagsBuffer []byte

		// Server will always have keys
		flags |= HasKeysFlag

		var natneg string
		var exists bool
		if natneg, exists = server["natneg"]; exists && natneg != "0" {
			flags |= ConnectNegotiateFlag
		}

		var publicip string
		if publicip, exists = server["publicip"]; !exists {
			logging.Error(ModuleName, "Server exists without public IP")
			continue
		}

		ip, err := strconv.ParseUint(publicip, 10, 32)
		if err != nil {
			logging.Error(ModuleName, "Server has invalid public IP value:", aurora.Cyan(publicip))
		}

		flagsBuffer = binary.BigEndian.AppendUint32(flagsBuffer, uint32(ip))

		var port string
		port, exists = server["publicport"]
		if !exists {
			// Fall back to local port if public port doesn't exist
			if port, exists = server["localport"]; !exists {
				logging.Error(ModuleName, "Server exists without port (publicip =", aurora.Cyan(publicip).String()+")")
				continue
			}
		}

		portValue, err := strconv.ParseUint(port, 10, 16)
		if err != nil {
			logging.Error(ModuleName, "Server has invalid port value:", aurora.Cyan(port))
			continue
		}

		flags |= NonstandardPortFlag
		flagsBuffer = binary.BigEndian.AppendUint16(flagsBuffer, uint16(portValue))

		// Use the first local IP if it exists, this is used to skip natneg if multiple players are on the same network
		if localip0, exists := server["localip0"]; exists {
			flags |= PrivateIPFlag

			// localip is written like "192.168.255.255" for example, so it needs to be parsed
			ipSplit := strings.Split(localip0, ".")
			if len(ipSplit) != 4 {
				logging.Error(ModuleName, "Server has invalid local IP:", aurora.Cyan(localip0))
				continue
			}

			err = nil
			for _, s := range ipSplit {
				val, err := strconv.ParseUint(s, 10, 8)
				if err != nil {
					break
				}

				flagsBuffer = append(flagsBuffer, byte(val))
			}

			if err != nil {
				logging.Error(ModuleName, "Server has invalid local IP value:", aurora.Cyan(localip0))
				continue
			}
		}

		if localport, exists := server["localport"]; exists {
			portValue, err = strconv.ParseUint(localport, 10, 16)
			if err != nil {
				logging.Error(ModuleName, "Server has invalid local port value:", aurora.Cyan(localport))
				continue
			}

			flags |= NonstandardPrivatePortFlag
			flagsBuffer = binary.BigEndian.AppendUint16(flagsBuffer, uint16(portValue))
		}

		// Just a dummy IP? This is taken from dwc_network_server_emulator
		// TODO: Check if this is actually needed
		flags |= ICMPIPFlag
		flagsBuffer = append(flagsBuffer, []byte{0, 0, 0, 0}...)

		// Finally, write the server buffer to the output
		output = append(output, flags)
		output = append(output, flagsBuffer...)

		if (flags & HasKeysFlag) == 0 {
			// Server does not have keys, so skip them
			logging.Info(ModuleName, "Wrote server without keys")
			continue
		}

		// Add the requested fields
		for _, field := range fieldList {
			output = append(output, 0xff)

			if str, exists := server[field]; exists {
				output = append(output, []byte(str)...)
			}

			// Add null terminator so the string will be empty if the field doesn't exist
			output = append(output, 0x00)
		}

		logging.Info(ModuleName, "Wrote server with keys")
	}

	// Server with 0 flags and IP of 0xffffffff terminates the list
	output = append(output, []byte{0x00, 0xff, 0xff, 0xff, 0xff}...)

	// Write the encrypted reply
	conn.Write(common.EncryptTypeX([]byte("9r3Rmy"), challenge, output))
}

func handleSendMessageRequest(conn net.Conn, buffer []byte) {
	// Read destination IP from buffer
	destIP := fmt.Sprintf("%d.%d.%d.%d:%d", buffer[3], buffer[4], buffer[5], buffer[6], binary.BigEndian.Uint16(buffer[7:9]))

	logging.Notice(ModuleName, "Send message from", aurora.BrightCyan(conn.RemoteAddr()), "to", aurora.BrightCyan(destIP).String())

	// TODO: Perform basic packet verification
	// TODO SECURITY: Check if the selected IP is actually online, or at least make sure it's not a local IP

	qr2.SendClientMessage(destIP, buffer[9:])
}
